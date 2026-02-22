# Writing Scripts for goop

Scripts are plain `.js` files placed in `~/.local/share/goop/scripts/`. goop picks them
up automatically — no restart required. \
Please note, there is a 5MB size limit in place per valid script file.

---

## Anatomy of a script

Every script must start with a metadata header, followed by a `main(state)` function:

```js
/**!
 * @name          My Script
 * @description   What this script does
 * @icon          star
 * @tags          custom,example
 */

function main(state) {
    state.text = state.text.toUpperCase();
}
```

### Header fields

| Field | Required | Description |
|---|---|---|
| `@name` | Yes | Display name shown in the script picker |
| `@description` | Yes | Short description shown below the name |
| `@icon` | No | SF Symbol name (cosmetic only on Linux) |
| `@tags` | No | Comma-separated search tags |

---

## The `state` object

`main` receives a single `state` argument that exposes the editor content:

| Property / Method | Type | Description |
|---|---|---|
| `state.text` | `string` (r/w) | Selected text; equals `fullText` when nothing is selected |
| `state.fullText` | `string` (r/w) | Entire document content |
| `state.selection` | `{start, end}` (r) | Character offsets of the current selection |
| `state.insert(str)` | method | Insert `str` at the current cursor position |
| `state.postError(msg)` | method | Display `msg` as an error in the status bar |
| `state.postInfo(msg)` | method | Display `msg` as an informational message in the status bar |

### Mutation rules

goop infers what changed by inspecting which fields were written:

| What you wrote | Effect |
|---|---|
| `state.text = ...` | Replaces the selection (or full text if no selection) |
| `state.fullText = ...` | Replaces the entire document |
| `state.insert(str)` | Inserts at cursor; does not replace existing content |
| Nothing | No change applied |

If both `state.text` and `state.fullText` are written, `fullText` wins.

---

## Module support

> **tl;dr:** `require('@boop/...')` works. Arbitrary npm packages and ES6
> `import` do not.

### What changed vs. original Boop

The original Boop used JavaScriptCore with no module system — `@boop/` libraries
were injected as plain globals. goop runs on [Goja](https://github.com/dop251/goja)
with [goja_nodejs](https://github.com/dop251/goja_nodejs), which adds a real
CommonJS `require()` implementation.

### What is supported

| Feature | Supported | Notes |
|---|---|---|
| `require('@boop/...')` | Yes | All built-in `@boop/` modules |
| CommonJS inside a script | Yes | `const x = require('@boop/yaml')` etc. |
| ES6 `import` / `export` | No | Goja does not implement ES modules |
| Arbitrary npm packages | No | Non-`@boop/` paths are hard-blocked by the engine |
| Network access | No | `fetch`, `XMLHttpRequest`, `WebSocket` are removed |
| `setTimeout` / `setInterval` | No | Removed; scripts must be synchronous |
| `process` / `Buffer` | No | Removed |

### Available `@boop/` modules

Import them with CommonJS `require()`:

```js
const yaml = require('@boop/yaml');
const plist = require('@boop/plist');
```

| Module | Exports |
|---|---|
| `@boop/base64` | `encode(str)`, `decode(str)` |
| `@boop/yaml` | `parse(str)`, `stringify(obj)` |
| `@boop/plist` | `parse(str)`, `stringify(obj)`, `parseBinary(str)` |
| `@boop/hashes` | `Hashes` object — MD5, SHA-1, SHA-256, SHA-512, … |
| `@boop/he` | `encode(str)`, `decode(str)` — HTML entities |
| `@boop/js-yaml` | Full js-yaml API |
| `@boop/lodash.boop` | `camelCase`, `kebabCase`, `snakeCase`, `startCase`, `deburr`, `size` |
| `@boop/vkBeautify` | `xml`, `xmlmin`, `css`, `cssmin`, `sql`, `sqlmin` |
| `@boop/papaparse.js` | `Papa.parse`, `Papa.unparse` — CSV |
| `@boop/node-forge` | Cryptography suite — ASN.1, PKI, message digests, ciphers, HMAC |

#### `@boop/node-forge` in depth

[node-forge](https://github.com/digitalbazaar/forge) is a full-featured
cryptography library bundled as a single browserified file. It exposes its API
on the `forge` object returned by `require`.

```js
const forge = require('@boop/node-forge');
```

**Key namespaces**

| Namespace | Purpose |
|---|---|
| `forge.asn1` | Low-level ASN.1 parsing and serialization |
| `forge.pki` | PKI helpers — certificate parsing, OID map |
| `forge.md` | Message digest algorithms (MD5, SHA-1, SHA-256, SHA-384, SHA-512) |
| `forge.hmac` | HMAC construction |
| `forge.cipher` | Symmetric ciphers (AES-CBC/CTR/GCM, 3DES) |
| `forge.util` | Utility functions — base64, hex, binary string conversions |
| `forge.random` | Pseudorandom byte generation |

**Important: use `forge.util.decode64()` for binary data, not `atob()`**

goop's built-in `atob()` returns a UTF-8 string, which corrupts bytes > 127
when passed to `forge.asn1.fromDer()`. Always use `forge.util.decode64()` when
decoding base64-encoded binary structures (DER, raw keys, etc.):

```js
// Good
const der = forge.util.decode64(base64PEM);
const cert = forge.asn1.fromDer(der);

// Bad — corrupts non-ASCII bytes
const der = atob(base64PEM);
```

### Additional globals

| Global | Description |
|---|---|
| `btoa(str)` | Base64 encode (browser-compatible) |
| `atob(str)` | Base64 decode (browser-compatible) |
| `console.log(...)` | Writes to the goop log file (XDG cache dir) |

---

## Examples

### Simple: reverse selected text

```js
/**!
 * @name        Reverse Text
 * @description Reverses the selected text character by character
 * @tags        reverse,text
 */

function main(state) {
    state.text = state.text.split('').reverse().join('');
}
```

### Using a `@boop/` module

```js
/**!
 * @name        Format YAML
 * @description Pretty-prints YAML
 * @tags        yaml,format
 */

const yaml = require('@boop/yaml');

function main(state) {
    try {
        const parsed = yaml.parse(state.text);
        state.text = yaml.stringify(parsed);
    } catch (e) {
        state.postError('Invalid YAML: ' + e.message);
    }
}
```

### Reporting results without mutating text

```js
/**!
 * @name        Count Words
 * @description Shows word count in the status bar
 * @tags        count,words,stats
 */

function main(state) {
    const words = state.text.trim().split(/\s+/).filter(Boolean).length;
    state.postInfo(words + ' words');
}
```

### Working with the full document

```js
/**!
 * @name        Sort Lines
 * @description Sorts all lines in the document alphabetically
 * @tags        sort,lines
 */

function main(state) {
    state.fullText = state.fullText
        .split('\n')
        .sort()
        .join('\n');
}
```

### Hashing with `@boop/node-forge`

```js
/**!
 * @name        SHA-256 Hash
 * @description Computes the SHA-256 hash of the selected text
 * @tags        hash,sha256,crypto
 */

const forge = require('@boop/node-forge');

function main(state) {
    const digest = forge.md.sha256.create();
    digest.update(state.text);
    state.text = digest.digest().toHex();
}
```

---

## Compatibility with Boop scripts

Most scripts from the [Boop script ecosystem](https://github.com/IvanMathy/Boop)
work without modification. The main incompatibility is scripts that rely on
JavaScriptCore-specific APIs not available in Goja (rare in practice).

Scripts from the upstream community [Scripts/](/Scripts/) directory are
bundled with goop and work out of the box.

Check also upstream's [script documentation](https://github.com/IvanMathy/Boop/tree/main/Boop/Documentation).
