# goop

A Linux-native re-implementation of [Boop](https://github.com/IvanMathy/Boop) — a
text transformation tool for developers. Paste text, pick a script, and transform it
in seconds.

Built with Go and GTK4. Single binary, no Electron, no runtime.

## Heritage

- created by [Ivan Mathy](https://github.com/IvanMathy/) as [Boop](https://github.com/IvanMathy/Boop) in 2019 (macOS, Swift)
- re-implemented by [Zoey Sheffield](https://github.com/zoeyfyi) as [Boop-GTK](https://github.com/zoeyfyi/Boop-GTK) in 2020 (*nix, Rust)
- re-imagined by [Daniel Ciaglia](https://www.linkedin.com/in/danielciaglia/) as [goop](https://codeberg.org/sigterm-de/goop) in 2026 (*nix, Golang)
  - implemented by [Claude Code](https://github.com/anthropics/claude-code) using [GitHub/spec-kit](https://github.com/github/spec-kit)

![](goop.webp)

---

## Features

- 70+ built-in text transformation scripts (URL encode/decode, Base64, JSON format,
  hashing, case conversion, sorting, and more)
- Real-time fuzzy script search
- User-provided custom scripts in `~/.config/goop/scripts/`
- Full compatibility with the upstream Boop script ecosystem (`@boop/` modules)
- Undo support
- XDG-compliant paths (config, logs)

## Required Libraries

| Library | Package name (varies by distro) |
|---------|----------------------------------|
| GTK 4   | `gtk4` / `libgtk-4-1`           |
| GtkSourceView 5 | `gtksourceview5` / `libgtksourceview-5-0` |

## Notes

- The embedded Boop scripts require no external files or network access.
- User-provided scripts are loaded from `~/.config/goop/scripts/` and also
  require no additional dependencies beyond what the scripts themselves use.

## Build from Source

**Prerequisites:** Go 1.26+, Clang, GTK4 and GtkSourceView5 development headers.

Make sure to set `CC=clang` as GCC 15 has issues (at least for me)

```bash
git clone https://codeberg.org/sigterm-de/goop
cd goop
CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop
```

Use the provided [Taskfile](https://taskfile.dev/)

```shell
$> task

task: [default] task --list
task: Available tasks for this project:
* build:               Compile the binary
* check:               Run fmt:check, fix, vet, lint and test (full pre-commit gate)
 [...]
* test:coverage:       Run tests and print per-function coverage (gate ≥80%)
```

## Usage

1. Launch `goop`
2. Paste or type text into the editor area
3. Press `Ctrl+/` to open the script picker
4. Type to filter scripts by name or description
5. Click a script or press Enter to run it
6. Press `Ctrl+Z` to undo the last transformation
7. Press `Escape` to dismiss the script picker without running anything

# Custom Scripts

Place `.js` files in `~/.config/goop/scripts/`. See
[writing-scripts.md](writing-scripts.md) for the full guide, including
the state API, available `@boop/` modules, and how CommonJS `require()` works in
goop (an improvement over original Boop, which had no module system at all).

## Community Scripts

The `Scripts/` directory contains community-contributed scripts from the upstream Boop
project. See [Scripts/README.md](Scripts/README.md) for details.
## License

See [LICENSE](LICENSE).
