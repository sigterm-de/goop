/**!
 * @name          Decode PEM Certificate
 * @description   Fully decodes PEM certificates (RSA and EC): subject, issuer, SANs, fingerprints, and more
 * @author        Daniel Ciaglia + Claude
 * @icon          fingerprint
 * @tags          pem,certificate,decode,ssl,tls,x509,openssl,forge
 */

const forge = require('@boop/node-forge');

// OIDs missing from forge.pki.oids (EC key types and curves, ECDSA signature algorithms)
const EXTRA_OIDS = {
    '1.2.840.10045.2.1':   'ecPublicKey',
    '1.2.840.10045.3.1.7': 'P-256 (secp256r1)',
    '1.3.132.0.34':        'P-384 (secp384r1)',
    '1.3.132.0.35':        'P-521 (secp521r1)',
    '1.3.132.0.10':        'secp256k1',
    '1.2.840.10040.4.1':   'id-dsa',
    '1.2.840.10045.4.3.1': 'ecdsa-with-SHA224',
    '1.2.840.10045.4.3.2': 'ecdsa-with-SHA256',
    '1.2.840.10045.4.3.3': 'ecdsa-with-SHA384',
    '1.2.840.10045.4.3.4': 'ecdsa-with-SHA512',
};

const DN_LABELS = {
    commonName: 'CN', organizationName: 'O', organizationalUnitName: 'OU',
    countryName: 'C', stateOrProvinceName: 'ST', localityName: 'L',
    emailAddress: 'E', serialNumber: 'SN',
};

// ASN.1 universal type numbers used below
var T_BOOLEAN = 1, T_INTEGER = 2, T_BITSTRING = 3, T_OCTETSTRING = 4, T_OID = 6;
var T_UTCTIME = 23, T_GENTIME = 24;
// The bundled browserified forge stores raw class bits (0,64,128,192) rather than
// the normalized 0-3 values used by the forge npm package.  Use the constant from
// the library itself so the comparison works regardless of the bundle strategy.
var CLS_CONTEXT = forge.asn1.Class.CONTEXT_SPECIFIC; // 128 in this bundle

function oidName(oid) {
    return forge.pki.oids[oid] || EXTRA_OIDS[oid] || oid;
}

function toBigHex(bytes) {
    var h = '';
    for (var i = 0; i < bytes.length; i++) h += bytes.charCodeAt(i).toString(16).padStart(2, '0');
    return h;
}

function colonHex(hex) {
    return hex.toUpperCase().match(/.{2}/g).join(':');
}

// Parse an X.509 Name SEQUENCE into a readable DN string.
function parseName(nameSeq) {
    var attrs = [];
    nameSeq.value.forEach(function(rdnSet) {      // SET
        rdnSet.value.forEach(function(atv) {       // SEQUENCE { OID, value }
            var oid = forge.asn1.derToOid(atv.value[0].value);
            var long = forge.pki.oids[oid] || oid;
            var label = DN_LABELS[long] || long;
            attrs.push(label + '=' + atv.value[1].value);
        });
    });
    return attrs.join(', ');
}

// Parse UTCTime (YYMMDDHHMMSSZ) or GeneralizedTime (YYYYMMDDHHMMSSZ) to a Date.
function parseTime(t) {
    var s = t.value, year, rest;
    if (t.type === T_UTCTIME) {
        var y2 = parseInt(s.substring(0, 2), 10);
        year = y2 >= 50 ? 1900 + y2 : 2000 + y2;
        rest = s.substring(2);
    } else {
        year = parseInt(s.substring(0, 4), 10);
        rest = s.substring(4);
    }
    return new Date(Date.UTC(year,
        parseInt(rest.substring(0, 2), 10) - 1,
        parseInt(rest.substring(2, 4), 10),
        parseInt(rest.substring(4, 6), 10),
        parseInt(rest.substring(6, 8), 10),
        parseInt(rest.substring(8, 10), 10)));
}

function formatDate(d) {
    return d.toISOString().replace('T', ' ').substring(0, 19) + ' UTC';
}

// Parse SubjectPublicKeyInfo SEQUENCE and return formatted lines.
function parseSPKI(spki) {
    var algOid = forge.asn1.derToOid(spki.value[0].value[0].value);
    var out = '  Algorithm:      ' + oidName(algOid) + '\n';

    if (algOid === '1.2.840.113549.1.1.1') {
        // RSA: unwrap BIT STRING → parse RSAPublicKey SEQUENCE { modulus, exponent }.
        // forge may auto-decode the BIT STRING (decodeBitStrings:true default), so
        // value[1].value is either a raw byte string or a pre-parsed ASN.1 array.
        var keyNode = spki.value[1];
        var rsaAsn1;
        if (typeof keyNode.value === 'string') {
            rsaAsn1 = forge.asn1.fromDer(keyNode.value.substring(1)); // skip unused-bits byte
        } else {
            rsaAsn1 = keyNode.value[0]; // auto-decoded: value[0] is RSAPublicKey SEQUENCE
        }
        var mod = rsaAsn1.value[0].value;
        var modBytes = mod.charCodeAt(0) === 0 ? mod.length - 1 : mod.length;
        var firstSigByte = mod.charCodeAt(mod.charCodeAt(0) === 0 ? 1 : 0);
        var leadingZeros = 0;
        for (var b = 0x80; b > firstSigByte; b >>= 1) leadingZeros++;
        var bits = modBytes * 8 - leadingZeros;
        var expBytes = rsaAsn1.value[1].value;
        var exp = 0;
        for (var j = 0; j < expBytes.length; j++) exp = exp * 256 + expBytes.charCodeAt(j);
        out += '  Key Size:       ' + bits + ' bit\n';
        out += '  Exponent:       ' + exp + ' (0x' + exp.toString(16) + ')\n';

    } else if (algOid === '1.2.840.10045.2.1') {
        // EC: curve OID is the second element of the AlgorithmIdentifier
        var algParams = spki.value[0].value;
        if (algParams.length > 1 && algParams[1].type === T_OID) {
            var curveOid = forge.asn1.derToOid(algParams[1].value);
            out += '  Curve:          ' + oidName(curveOid) + '\n';
        }
    }
    return out;
}

// Format the value of a single extension given its parsed inner ASN.1.
function formatExtValue(oid, inner, critical) {
    // subjectAltName
    if (oid === '2.5.29.17') {
        var typeLabel = { 1: 'email', 2: 'DNS', 6: 'URI', 7: 'IP' };
        return inner.value.map(function(alt) {
            var val = alt.value;
            if (alt.type === 7) {
                val = alt.value.length === 4
                    ? [0,1,2,3].map(function(k) { return alt.value.charCodeAt(k); }).join('.')
                    : '[IPv6]';
            }
            return '      ' + (typeLabel[alt.type] || 'other') + ': ' + val + '\n';
        }).join('');
    }
    // basicConstraints
    if (oid === '2.5.29.19') {
        var isCA = false, pathLen;
        inner.value.forEach(function(v) {
            if (v.type === T_BOOLEAN) isCA = v.value.charCodeAt(0) !== 0;
            if (v.type === T_INTEGER) {
                pathLen = 0;
                for (var k = 0; k < v.value.length; k++) pathLen = pathLen * 256 + v.value.charCodeAt(k);
            }
        });
        return '      CA: ' + (isCA ? 'TRUE' : 'FALSE') + '\n' +
               (pathLen !== undefined ? '      Path Length: ' + pathLen + '\n' : '');
    }
    // keyUsage (BIT STRING: first byte = unused-bit count, remaining = bit flags)
    if (oid === '2.5.29.15') {
        var b1 = inner.value.length > 1 ? inner.value.charCodeAt(1) : 0;
        var b2 = inner.value.length > 2 ? inner.value.charCodeAt(2) : 0;
        var allBits = (b1 << 8) | b2;
        var usageNames = ['digitalSignature','nonRepudiation','keyEncipherment','dataEncipherment',
                          'keyAgreement','keyCertSign','cRLSign','encipherOnly','decipherOnly'];
        var set = usageNames.filter(function(_, k) { return allBits & (0x8000 >> k); });
        return '      ' + (set.join(', ') || 'none') + '\n';
    }
    // extendedKeyUsage
    if (oid === '2.5.29.37') {
        var usages = inner.value.map(function(v) { return oidName(forge.asn1.derToOid(v.value)); });
        return '      ' + usages.join(', ') + '\n';
    }
    // subjectKeyIdentifier
    if (oid === '2.5.29.14') {
        return '      ' + colonHex(toBigHex(inner.value)) + '\n';
    }
    // authorityKeyIdentifier
    if (oid === '2.5.29.35') {
        var aki = '';
        inner.value.forEach(function(v) {
            if (v.tagClass === CLS_CONTEXT && v.type === 0) {
                aki += '      keyid: ' + colonHex(toBigHex(v.value)) + '\n';
            }
        });
        return aki || '      (present)\n';
    }
    // certificatePolicies
    if (oid === '2.5.29.32') {
        return inner.value.map(function(policy) {
            var pOid = forge.asn1.derToOid(policy.value[0].value);
            return '      ' + oidName(pOid) + '\n';
        }).join('');
    }
    return '      Critical: ' + (critical ? 'Yes' : 'No') + '\n';
}

// Parse the Extensions SEQUENCE OF and return formatted lines.
function parseExtensions(extsSeq) {
    var out = '';
    extsSeq.value.forEach(function(ext) {
        var idx = 0;
        var oid = forge.asn1.derToOid(ext.value[idx++].value);
        var name = oidName(oid);

        var critical = false;
        if (ext.value[idx].type === T_BOOLEAN) {
            critical = ext.value[idx++].value.charCodeAt(0) !== 0;
        }
        var innerDer = ext.value[idx].value; // OCTET STRING wraps the actual extension DER

        out += '    ' + name + (critical ? ' (critical)' : '') + ':\n';
        try {
            out += formatExtValue(oid, forge.asn1.fromDer(innerDer, {strict: false, decodeBitStrings: false}), critical);
        } catch (e) {
            out += '      (parse error: ' + e.message + ')\n';
        }
    });
    return out || '    None\n';
}

function main(state) {
    try {
        var text = state.text.trim();
        if (!text.includes('BEGIN CERTIFICATE')) {
            state.postError('Input does not appear to be a PEM certificate');
            return;
        }

        var pemBody = text
            .replace(/-----BEGIN CERTIFICATE-----/g, '')
            .replace(/-----END CERTIFICATE-----/g, '')
            .replace(/\s+/g, '');

        // Use forge's own base64 decoder instead of the global atob(): atob() is
        // implemented in Go (executor.go) as string([]byte), which produces UTF-8.
        // Bytes > 127 get encoded as multi-byte sequences, corrupting the binary
        // DER data that forge.asn1.fromDer() expects. forge.util.decode64() uses
        // String.fromCharCode() internally, which gives clean Latin-1 code points.
        var der = forge.util.decode64(pemBody);

        // decodeBitStrings:false keeps BIT STRING values as raw byte strings so
        // that we can use .substring() on the key and signature bit strings.
        // strict:false tolerates some minor DER violations seen in the wild.
        var ASN1_OPTS = {strict: false, decodeBitStrings: false};

        var asn1Cert;
        try { asn1Cert = forge.asn1.fromDer(der, ASN1_OPTS); }
        catch (e) { state.postError('Failed to parse certificate ASN.1: ' + e.message); return; }

        // Certificate SEQUENCE { TBSCertificate, signatureAlgorithm, signature BIT STRING }
        var tbs = asn1Cert.value[0];
        var i = 0;

        // version [0] EXPLICIT INTEGER (optional, default v1)
        var version = 1;
        if (tbs.value[i].tagClass === CLS_CONTEXT && tbs.value[i].type === 0) {
            version = tbs.value[i].value[0].value.charCodeAt(0) + 1;
            i++;
        }

        var serialInt   = tbs.value[i++]; // INTEGER
        var sigAlgSeq   = tbs.value[i++]; // AlgorithmIdentifier
        var issuerSeq   = tbs.value[i++]; // Name
        var validitySeq = tbs.value[i++]; // Validity
        var subjectSeq  = tbs.value[i++]; // Name
        var spkiSeq     = tbs.value[i++]; // SubjectPublicKeyInfo

        // Scan remaining fields for [3] EXPLICIT extensions (skip unique IDs)
        var extsNode = null;
        while (i < tbs.value.length) {
            if (tbs.value[i].tagClass === CLS_CONTEXT && tbs.value[i].type === 3) {
                extsNode = tbs.value[i].value[0]; // unwrap EXPLICIT → SEQUENCE OF Extension
            }
            i++;
        }

        var notBefore = parseTime(validitySeq.value[0]);
        var notAfter  = parseTime(validitySeq.value[1]);
        var now = new Date();
        var status = now < notBefore ? 'NOT YET VALID' : now > notAfter ? 'EXPIRED' : 'VALID';
        var sigAlgOid = forge.asn1.derToOid(sigAlgSeq.value[0].value);

        var sep = '='.repeat(70);
        var out = sep + '\nCERTIFICATE INFORMATION\n' + sep + '\n\n';

        out += 'Version:          ' + version + ' (0x' + (version - 1).toString(16) + ')\n';
        out += 'Serial Number:    ' + colonHex(toBigHex(serialInt.value)) + '\n';
        out += 'Signature Alg:    ' + oidName(sigAlgOid) + '\n\n';
        out += 'Issuer:           ' + parseName(issuerSeq) + '\n';
        out += 'Subject:          ' + parseName(subjectSeq) + '\n\n';
        out += 'Validity:\n';
        out += '  Not Before:     ' + formatDate(notBefore) + '\n';
        out += '  Not After:      ' + formatDate(notAfter) + '\n';
        out += '  Status:         ' + status + '\n\n';
        out += 'Public Key:\n';
        out += parseSPKI(spkiSeq);
        out += '\n';
        out += 'X.509v3 Extensions:\n';
        out += extsNode ? parseExtensions(extsNode) : '    None\n';
        out += '\n';

        // Fingerprints over the raw DER bytes
        out += 'Fingerprints:\n';
        out += '  MD5:    ' + colonHex(forge.md.md5.create().update(der).digest().toHex()) + '\n';
        out += '  SHA1:   ' + colonHex(forge.md.sha1.create().update(der).digest().toHex()) + '\n';
        out += '  SHA256: ' + colonHex(forge.md.sha256.create().update(der).digest().toHex()) + '\n';
        out += '\n';

        // Raw signature bytes from the outer Certificate BIT STRING.
        // Same as the key node: value may be a raw string or a pre-parsed ASN.1 array
        // (ECDSA signatures are valid ASN.1 and get auto-decoded by forge).
        var sigNode = asn1Cert.value[2];
        var sigHex;
        if (typeof sigNode.value === 'string') {
            sigHex = toBigHex(sigNode.value.substring(1)).toUpperCase(); // skip unused-bits byte
        } else {
            // Re-serialise the auto-decoded items to recover the raw DER bytes.
            var sigDerBytes = '';
            sigNode.value.forEach(function(item) { sigDerBytes += forge.asn1.toDer(item).getBytes(); });
            sigHex = toBigHex(sigDerBytes).toUpperCase();
        }
        out += 'Signature (' + Math.floor(sigHex.length / 2) + ' bytes):\n';
        for (var k = 0; k < sigHex.length; k += 48) {
            out += '  ' + sigHex.substring(k, k + 48).match(/.{2}/g).join(':') + '\n';
        }
        out += '\n' + sep + '\n';

        state.text = out;

    } catch (err) {
        state.postError('Error decoding certificate: ' + err.message);
    }
}
