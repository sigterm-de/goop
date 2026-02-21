package integration_test

import (
	"strings"
	"testing"

	"codeberg.org/sigterm-de/goop/internal/engine"
)

// testCertPEM is a static self-signed RSA-2048 certificate with known fields,
// generated for test purposes only. It is valid until 2036.
//
//	Subject: C=DE, ST=Berlin, L=Berlin, O=Test Org, OU=Test Unit, CN=test.example.com
//	SANs:    DNS:test.example.com, DNS:www.test.example.com
//	Key:     RSA 2048
const testCertPEM = `-----BEGIN CERTIFICATE-----
MIID+DCCAuCgAwIBAgIUb11VBHJpzgz1xDJ68TJWfgFUifowDQYJKoZIhvcNAQEL
BQAwcTELMAkGA1UEBhMCREUxDzANBgNVBAgMBkJlcmxpbjEPMA0GA1UEBwwGQmVy
bGluMREwDwYDVQQKDAhUZXN0IE9yZzESMBAGA1UECwwJVGVzdCBVbml0MRkwFwYD
VQQDDBB0ZXN0LmV4YW1wbGUuY29tMB4XDTI2MDIyMTE0NDMxNFoXDTM2MDIxOTE0
NDMxNFowcTELMAkGA1UEBhMCREUxDzANBgNVBAgMBkJlcmxpbjEPMA0GA1UEBwwG
QmVybGluMREwDwYDVQQKDAhUZXN0IE9yZzESMBAGA1UECwwJVGVzdCBVbml0MRkw
FwYDVQQDDBB0ZXN0LmV4YW1wbGUuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAqjDOZ209n4J8tT7vsa3rYrDY5kwLhr7E3BUenxvzQEeEw3qEqKUq
X4wDL5BNQXBs9y5/gwLwlx86jtil2EDGdhtTlVwQ4C6Nrkwi5ZlUzKWIiHYbEpTI
PkgV1/o9/w/vmRVRsieES0H+sydy9FVsY8l0pGd3LdaZhj5GU2vGIybSLcDO36Ds
4Vol04qF3DIoAzAaSVlbetaPB8gtylvOyRm+0LdbTNGZEo2VS267mK3vfPVSUTkK
JBC86fP9W3FPpYJew5hcX6QCaBkrE4sZo803WvMS3P1H6yAmsLAfBr/xQ58k3Clz
wOjEuD0M8OdpJrsYyFWKGHge/xcDjqVz7QIDAQABo4GHMIGEMB0GA1UdDgQWBBRm
hnUQGa3tCtonsWNmtIwqExktfzAfBgNVHSMEGDAWgBRmhnUQGa3tCtonsWNmtIwq
ExktfzAPBgNVHRMBAf8EBTADAQH/MDEGA1UdEQQqMCiCEHRlc3QuZXhhbXBsZS5j
b22CFHd3dy50ZXN0LmV4YW1wbGUuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQAubkyt
dGB6+BMQP6UYnn4dxAZvov7OrpPyTcyBhSBp6q2MjpY9MqTnCBls4doiEfxYBndw
FsbcWSgKkZZWvtm+r6Tc+PVMpoPiWWaGeuqsFM++3o9BHJkPrpvS9bFV8uxQSFu+
2m2ingyjv/RQJOq57FZNRtGP+PkzUrOH6nG8B2GZ9ZbXCQUgbLjVza3G939TZyHa
jUmOyKlZ9rZPv0x+D91ql34/JfXTHbQjp9vCnMP0m6dP6LbZ+WZdFpPfjId+Va4f
avXELf5ovZcXNMvy96JIyAUNbaEqofTtS4muWg8ifYKQTgxlQboNrw70c7j2I6aU
tGamq91WimBWZDF0
-----END CERTIFICATE-----`

func TestDecodePEMCertificate(t *testing.T) {
	content := loadScript(t, "Decode PEM Certificate")

	result := execScript(t, "Decode PEM Certificate", content, testCertPEM)
	if !result.Success {
		t.Fatalf("script failed: %s", result.ErrorMessage)
	}

	var out string
	switch result.MutationKind {
	case engine.MutationReplaceSelect:
		out = result.NewText
	case engine.MutationReplaceDoc:
		out = result.NewFullText
	default:
		t.Fatalf("unexpected mutation kind: %v", result.MutationKind)
	}

	checks := []struct {
		label   string
		contain string
	}{
		{"subject CN", "CN=test.example.com"},
		{"subject O", "O=Test Org"},
		{"subject C", "C=DE"},
		{"issuer", "Issuer:"},
		{"validity", "Validity:"},
		{"status valid", "VALID"},
		{"RSA key algorithm", "RSA"},
		{"RSA key size", "2048 bit"},
		{"SAN DNS entry", "DNS: test.example.com"},
		{"SAN www entry", "DNS: www.test.example.com"},
		{"SHA256 fingerprint", "SHA256:"},
		{"SHA1 fingerprint", "SHA1:"},
		{"MD5 fingerprint", "MD5:"},
		{"extensions section", "X.509v3 Extensions:"},
	}

	for _, c := range checks {
		if !strings.Contains(out, c.contain) {
			t.Errorf("output missing %s: expected to contain %q\nFull output:\n%s", c.label, c.contain, out)
		}
	}
}

// testCertPEMEC is a static self-signed EC P-256 certificate with known fields,
// generated for test purposes only. It is valid until 2036.
//
//	Subject: C=DE, ST=Berlin, L=Berlin, O=Test Org, OU=Test Unit, CN=test.example.com
//	SANs:    DNS:test.example.com, DNS:www.test.example.com
//	Key:     EC P-256
const testCertPEMEC = `-----BEGIN CERTIFICATE-----
MIICbTCCAhKgAwIBAgIUAx7Iy1VmGuNRKkV/kqCBgqfdllIwCgYIKoZIzj0EAwIw
cTELMAkGA1UEBhMCREUxDzANBgNVBAgMBkJlcmxpbjEPMA0GA1UEBwwGQmVybGlu
MREwDwYDVQQKDAhUZXN0IE9yZzESMBAGA1UECwwJVGVzdCBVbml0MRkwFwYDVQQD
DBB0ZXN0LmV4YW1wbGUuY29tMB4XDTI2MDIyMTE0MzMzMVoXDTM2MDIxOTE0MzMz
MVowcTELMAkGA1UEBhMCREUxDzANBgNVBAgMBkJlcmxpbjEPMA0GA1UEBwwGQmVy
bGluMREwDwYDVQQKDAhUZXN0IE9yZzESMBAGA1UECwwJVGVzdCBVbml0MRkwFwYD
VQQDDBB0ZXN0LmV4YW1wbGUuY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
Ow4czbKO5bVHoeN3bLQIRz88w9hs7WOFmONdFWSVzNFkAAVtnPy25mgCwE58oCw9
pSUv+scbLZy8k2LwD3gM7aOBhzCBhDAdBgNVHQ4EFgQUXHwqPlLIKGL/qkFGAAfY
KnyqgsUwHwYDVR0jBBgwFoAUXHwqPlLIKGL/qkFGAAfYKnyqgsUwDwYDVR0TAQH/
BAUwAwEB/zAxBgNVHREEKjAoghB0ZXN0LmV4YW1wbGUuY29tghR3d3cudGVzdC5l
eGFtcGxlLmNvbTAKBggqhkjOPQQDAgNJADBGAiEAgZX8droZnonMhdhcZ6WeO63T
7AoTc3TuuXl30WWZJL0CIQCWGsNtcPJa/jKDbScL9h0adq/5BDl+PTE+Jywzn4pd
2g==
-----END CERTIFICATE-----`

func TestDecodePEMCertificate_EC(t *testing.T) {
	content := loadScript(t, "Decode PEM Certificate")

	result := execScript(t, "Decode PEM Certificate", content, testCertPEMEC)
	if !result.Success {
		t.Fatalf("script failed: %s", result.ErrorMessage)
	}

	var out string
	switch result.MutationKind {
	case engine.MutationReplaceSelect:
		out = result.NewText
	case engine.MutationReplaceDoc:
		out = result.NewFullText
	default:
		t.Fatalf("unexpected mutation kind: %v", result.MutationKind)
	}

	checks := []struct {
		label   string
		contain string
	}{
		{"subject CN", "CN=test.example.com"},
		{"subject C", "C=DE"},
		{"EC key algorithm", "ecPublicKey"},
		{"P-256 curve", "P-256"},
		{"SAN DNS entry", "DNS: test.example.com"},
		{"SHA256 fingerprint", "SHA256:"},
		{"extensions section", "X.509v3 Extensions:"},
	}

	for _, c := range checks {
		if !strings.Contains(out, c.contain) {
			t.Errorf("output missing %s: expected to contain %q\nFull output:\n%s", c.label, c.contain, out)
		}
	}
}

func TestDecodePEMCertificate_InvalidInput(t *testing.T) {
	content := loadScript(t, "Decode PEM Certificate")

	result := execScript(t, "Decode PEM Certificate", content, "not a certificate")
	// postError() sets Success=false by design; verify it's a graceful user-facing
	// error (non-empty message) rather than an unexpected engine crash.
	if result.Success {
		t.Fatalf("expected failure for invalid input, got success")
	}
	if result.TimedOut {
		t.Fatalf("script timed out on invalid input")
	}
	if result.ErrorMessage == "" {
		t.Errorf("expected a non-empty error message for invalid input")
	}
}
