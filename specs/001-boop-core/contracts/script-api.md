# Contract: JavaScript Script API

**Date**: 2026-02-20
**Branch**: `001-boop-core`
**Implements**: FR-007, FR-015, SC-002

This contract defines the complete JavaScript API available to transformation scripts.
Any script that runs correctly in the original Boop macOS app MUST run correctly in
goop without modification (SC-002). This contract is the acceptance test specification
for `tests/contract/script_api_test.go`.

---

## File Format

A valid Boop script file MUST meet all of the following:

1. Begin with a `/**!` header block (exactly `/**!` — no other variant).
2. Declare `@name` and `@description` in the header.
3. Define a top-level `function main(state) { ... }`.

### Header Format

```js
/**!
 * @name          <Display Name>
 * @description   <Short description shown in the picker>
 * @icon          <i class="fas fa-icon-name"></i>   <!-- optional -->
 * @tags          tag1,tag2,tag3                      <!-- optional, comma-separated -->
 * @bias          -0.5                                <!-- optional, float, default 0.0 -->
 */
```

Files not starting with `/**!` MUST be silently skipped (not loaded, not shown).
Files with `/**!` but missing `@name` or `@description` MUST be skipped and logged.

---

## The `state` Object

Passed as the sole argument to `main(state)`. The host constructs it from the current
editor content before calling `main`, then reads it back after `main` returns.

### Properties

#### `state.fullText` — `string` (read/write)

The complete text currently in the editor.

- **Read**: Returns the full document text at the moment `main(state)` was called.
- **Write**: Setting `state.fullText` instructs the host to replace the entire document
  content with the new value when `main` returns.

**Contract test** — given `editor = "hello world"`, no selection:
```js
function main(state) {
    // state.fullText === "hello world"
    state.fullText = state.fullText.toUpperCase();
    // After return: editor === "HELLO WORLD"
}
```

#### `state.text` — `string` (read/write)

The currently selected text. If no selection exists, identical to `state.fullText`.

- **Read (no selection)**: Returns full document text (same as `state.fullText`).
- **Read (with selection)**: Returns only the selected substring.
- **Write (no selection)**: Replaces the entire document (equivalent to writing `fullText`).
- **Write (with selection)**: Replaces only the selected range; the rest of the document
  is preserved.

**Contract test** — given `editor = "hello world"`, selection = `"world"`:
```js
function main(state) {
    // state.text === "world"
    // state.fullText === "hello world"
    state.text = state.text.toUpperCase();
    // After return: editor === "hello WORLD"
}
```

**Contract test** — canonical selection-aware pattern:
```js
function main(state) {
    state.text = doTransform(state.text);
    // Works correctly whether or not text is selected.
}
```

**Write priority**: If both `state.text` and `state.fullText` are written, `state.text`
write takes precedence (selection-scoped replacement is applied).

#### `state.selection` — `object` (read-only)

Describes the current selection or caret position.

- `state.selection.start` — `number`: 0-based character offset of selection start.
- `state.selection.end` — `number`: 0-based character offset of selection end.
- When there is no selection: `start === end` (collapsed caret).

**Contract test**:
```js
function main(state) {
    // state.selection.start and state.selection.end are both numbers >= 0
    // state.selection.start <= state.selection.end
    // when no selection: state.selection.start === state.selection.end
}
```

`state.selection` is read-only. Writes to it are silently ignored.

### Methods

#### `state.insert(text: string) → void`

Inserts `text` at the current cursor position, replacing any active selection.
After `insert()` returns, the cursor is positioned after the inserted text.

- Takes precedence over neither `state.text` nor `state.fullText` mutations if both
  are also set (the last write mechanism used determines the result).
- Primary use: insert generated content at cursor (e.g., UUID, Lorem Ipsum).

**Contract test**:
```js
function main(state) {
    state.insert("INSERTED");
    // editor content at cursor position gains "INSERTED"
}
```

#### `state.postError(message: string) → void`

Signals an error to the host application.

- The `message` is displayed in the UI status area (in plain English).
- The `message` (with timestamp and script name) is written to the log file.
- **All mutations** to `state.fullText`, `state.text`, and pending `insert()` calls
  are **discarded**. The editor content is restored to its exact pre-execution state.
- Execution continues after `postError()` (it is not `throw`); subsequent state
  mutations are still discarded.
- Calling `postError` multiple times: only the first call's message is used.

**Contract test**:
```js
function main(state) {
    state.fullText = "THIS MUST NOT APPEAR";
    state.postError("Something went wrong");
    state.fullText = "ALSO MUST NOT APPEAR";
    // After return: editor unchanged, error shown, both mutations discarded.
}
```

---

## Write Semantics Priority Table

After `main(state)` returns, the host applies exactly one of the following outcomes,
evaluated in priority order:

| Priority | Condition | Action |
|----------|-----------|--------|
| 1 (highest) | `postError()` was called | Discard all mutations; show error; log |
| 2 | `state.text` was written | Replace selection range (or full doc if no selection) |
| 3 | `state.fullText` was written | Replace full document content |
| 4 | `state.insert()` was called | Insert text at cursor |
| 5 (lowest) | None of the above | No change to editor content |

---

## Available Globals

Scripts run in a sandboxed environment. The following globals are available:

### Standard JavaScript built-ins (always available)
`Array`, `Boolean`, `Date`, `Error`, `Function`, `JSON`, `Map`, `Math`, `Number`,
`Object`, `Promise`, `Proxy`, `Reflect`, `RegExp`, `Set`, `String`, `Symbol`,
`TypeError`, `WeakMap`, `WeakSet`, `parseInt`, `parseFloat`, `isNaN`, `isFinite`,
`encodeURIComponent`, `decodeURIComponent`, `encodeURI`, `decodeURI`,
`eval` (available but strongly discouraged)

### Additional globals provided by goop

| Global | Type | Description |
|--------|------|-------------|
| `btoa(data: string)` | function | Base64 encode (matches browser `btoa`) |
| `atob(data: string)` | function | Base64 decode (matches browser `atob`) |
| `console.log(...args)` | function | Writes to the log file at INFO level; NOT shown in UI |
| `require(path: string)` | function | Load `@boop/*` modules only (see Modules section) |
| `state` | object | The ScriptState object (see above) |

### Prohibited globals

The following globals MUST NOT be available. Accessing them MUST return `undefined`
or throw a `ReferenceError`:

`fetch`, `XMLHttpRequest`, `WebSocket`, `fs`, `process`, `os`, `path`,
`child_process`, `Buffer`, `global`, `window`, `document`, `navigator`,
`setTimeout`, `setInterval`, `clearTimeout`, `clearInterval`

---

## Module System (`require()`)

The `require()` function is available but MUST only resolve `@boop/` namespaced paths.
All other paths MUST throw `Error: Cannot find module '<path>'`.

### `@boop/plist`

```js
var plist = require('@boop/plist');
```

Exports:
- `plist.parse(str: string) → object` — Parse an XML plist string into a JS object.
- `plist.stringify(obj: object) → string` — Serialize a JS object to XML plist format.
- `plist.parseBinary(data: string) → object` — Parse a binary plist.

**Contract test**:
```js
var plist = require('@boop/plist');
var obj = plist.parse('<plist><dict><key>foo</key><string>bar</string></dict></plist>');
// obj.foo === "bar"
var xml = plist.stringify({ foo: "bar" });
// xml is a valid XML plist string containing "foo" and "bar"
```

### `@boop/yaml`

```js
var yaml = require('@boop/yaml');
```

Exports:
- `yaml.parse(str: string) → object` — Parse a YAML string into a JS object.
- `yaml.stringify(obj: object) → string` — Serialize a JS object to YAML.

**Contract test**:
```js
var yaml = require('@boop/yaml');
var obj = yaml.parse('name: Alice\nage: 30');
// obj.name === "Alice", obj.age === 30
var str = yaml.stringify({ name: "Alice", age: 30 });
// str contains "name: Alice" and "age: 30"
```

---

## Execution Constraints

| Constraint | Value | Behavior on violation |
|------------|-------|----------------------|
| Execution timeout | 5 seconds | VM interrupted; treated as `postError("Script execution timed out")` |
| Synchronous only | — | `async`/`await` and Promises are syntactically valid but resolve synchronously only; no async ticks occur |
| No side effects | — | No file system, network, or environment variable access |
| ES6+ required | ES2017 minimum | Arrow functions, destructuring, template literals, `class`, `Map`, `Set` all supported |
| ESM `import` | Not supported | Scripts MUST use `require()` or function-scope patterns; `import` statements throw a SyntaxError |

---

## Error Handling Contract

Unhandled JavaScript exceptions (not caught inside `main`) MUST be treated exactly as
if the script called `state.postError(exceptionMessage)`:
- The editor content is preserved unchanged.
- The exception message is shown in the UI and logged.
- The host does not crash.

**Contract test — unhandled throw**:
```js
function main(state) {
    state.fullText = "MUST NOT APPEAR";
    throw new Error("Unhandled error");
}
// After execution: editor unchanged, "Unhandled error" shown in UI and logged.
```

**Contract test — invalid module require**:
```js
function main(state) {
    var x = require('fs'); // MUST throw "Cannot find module 'fs'"
}
// After execution: editor unchanged, error shown in UI.
```
