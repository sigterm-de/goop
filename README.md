# goop

A Linux-native re-implementation of [Boop](https://github.com/IvanMathy/Boop) — a
text transformation tool for developers. Paste text, pick a script, and transform it
in seconds.

Built with Go and GTK4. Single binary, no Electron, no runtime.

## Features

- 70+ built-in text transformation scripts (URL encode/decode, Base64, JSON format,
  hashing, case conversion, sorting, and more)
- Real-time fuzzy script search
- User-provided custom scripts in `~/.config/goop/scripts/`
- Full compatibility with the upstream Boop script ecosystem (`@boop/` modules)
- Undo support
- XDG-compliant paths (config, logs)

## System Requirements

goop is distributed as a single compiled binary. The only runtime dependencies are the GTK4 and GtkSourceView system libraries.

## Required Libraries

| Library | Package name (varies by distro) |
|---------|----------------------------------|
| GTK 4   | `gtk4` / `libgtk-4-1`           |
| GtkSourceView 5 | `gtksourceview5` / `libgtksourceview-5-0` |

## Notes

- The embedded Boop scripts require no external files or network access.
- User-provided scripts are loaded from `~/.config/goop/scripts/` and also
  require no additional dependencies beyond what the scripts themselves use.

## Installation

Download a release archive from the [releases page](https://codeberg.org/daniel-ciaglia/goop/releases), extract and place the `goop` binary somewhere on your `$PATH`:

```bash
unzip goop_<version>_linux_amd64.zip
sudo install -m755 goop .local/bin/
```

To integrate with your desktop launcher, copy the `.desktop` file:

```bash
sudo cp .desktop/goop.desktop .local/share/applications/
```

## Build from Source

**Prerequisites:** Go 1.24+, Clang, GTK4 and GtkSourceView5 development headers.

```bash
git clone https://codeberg.org/daniel-ciaglia/goop
cd goop
CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop
```

## Usage

1. Launch `goop`
2. Paste or type text into the editor area
3. Press `Ctrl+/` to open the script picker
4. Type to filter scripts by name or description
5. Click a script or press Enter to run it
6. Press `Ctrl+Z` to undo the last transformation
7. Press `Escape` to dismiss the script picker without running anything

## Custom Scripts

Place `.js` files in `~/.config/goop/scripts/`. Scripts must begin with a
metadata header:

```js
/**!
 * @name          My Script
 * @description   What this script does
 * @icon          star
 * @tags          custom,example
 */

function main(state) {
    // state.text        — selected text (read/write)
    // state.fullText    — entire document (read/write)
    // state.selection   — {start, end} character offsets (read-only)
    // state.insert(str) — insert string at cursor
    // state.postError(msg) — display an error message

    state.text = state.text.toUpperCase();
}
```

### State API Quick Reference

| Property / Method | Type | Description |
|---|---|---|
| `state.text` | `string` (r/w) | Selected text; if nothing selected, equals `fullText` |
| `state.fullText` | `string` (r/w) | Entire document content |
| `state.selection` | `{start, end}` (r) | Character offsets of current selection |
| `state.insert(str)` | method | Insert `str` at the current cursor position |
| `state.postError(msg)` | method | Display `msg` as an error in the status bar |

### Available `@boop/` Modules

Community-compatible modules available via `require()`:

| Module | Functions |
|---|---|
| `@boop/base64` | `encode(str)`, `decode(str)` |
| `@boop/yaml` | `parse(str)`, `stringify(obj)` |
| `@boop/plist` | `parse(str)`, `stringify(obj)`, `parseBinary(str)` |
| `@boop/hashes` | Hashes object (MD5, SHA-1, SHA-256, ...) |
| `@boop/he` | `encode(str)`, `decode(str)` (HTML entities) |
| `@boop/js-yaml` | full js-yaml API |
| `@boop/lodash.boop` | `camelCase`, `kebabCase`, `snakeCase`, `startCase`, `deburr`, `size` |
| `@boop/vkBeautify` | `xml`, `xmlmin`, `css`, `cssmin`, `sql`, `sqlmin` |
| `@boop/papaparse.js` | Papa.parse / Papa.unparse (CSV) |

## Community Scripts

The `Scripts/` directory contains community-contributed scripts from the upstream Boop
project. See [Scripts/README.md](Scripts/README.md) for details.

## License

See [LICENSE](LICENSE).
