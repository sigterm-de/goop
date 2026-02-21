# goop Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-20

## Active Technologies
- Go 1.22+ (`CGO_ENABLED=1`, `CC=clang`) (required by gotk4 GTK4 bindings) (001-boop-core)
- `AppPreferences` JSON file via XDG (existing mechanism); one new field `syntax_auto_detect bool` (002-syntax-highlight)

## Project Structure

```text
cmd/goop/main.go         Entry point
internal/app/               GTK Application + ApplicationWindow
internal/ui/                Editor, script picker panel, status bar
internal/engine/            Sandboxed goja JS VM + state object
internal/scripts/           Script loading, /**! header parsing, library + fuzzy search
internal/logging/           XDG log file setup
assets/scripts/             Bundled Boop scripts (go:embed)
tests/contract/             Script API + library contract tests
tests/integration/          End-to-end transform + loading tests
.desktop/goop.desktop    XDG desktop entry
```

## Commands

```bash
# Build — CC=clang required (GCC 15 has incompatibilities with GTK4 CGo wrappers)
CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop

# Test (all packages)
CGO_ENABLED=1 CC=clang go test -race -coverprofile=coverage.out ./...

# Contract tests only
go test -race ./tests/contract/...

# Integration tests only
go test -race ./tests/integration/...

# Lint
golangci-lint run ./...

# Format
gofmt -w . && goimports -w .

# Vet
CGO_ENABLED=1 CC=clang go vet ./...
```

## Code Style

- Go 1.22+ idiomatic style; `gofmt` + `goimports` enforced
- Error wrapping: always use `fmt.Errorf("context: %w", err)`
- All GTK API calls MUST be on the main thread; use `glib.IdleAdd()` from goroutines
- `CGO_ENABLED=1` required; cross-compilation needs native arm64 CI runner
- Table-driven tests (`[]struct{...}`) for all unit tests
- Coverage gate: 80% minimum

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/diamondburned/gotk4/pkg/gtk/v4` | GTK4 UI bindings (CGo) |
| `github.com/diamondburned/gotk4-sourceview/pkg/gtksource/v5` | GtkSourceView editor |
| `github.com/dop251/goja` | ES6+ JavaScript engine (pure Go) |
| `github.com/dop251/goja_nodejs/require` | CommonJS require() for @boop/ modules |
| `github.com/sahilm/fuzzy` | Fuzzy search for script picker |
| `github.com/adrg/xdg` | XDG Base Directory paths |
| `gopkg.in/yaml.v3` | @boop/yaml module implementation |
| `howett.net/plist` | @boop/plist module implementation |

## System Libraries Required

Arch/Manjaro: `sudo pacman -S gtk4 gtksourceview5 pkg-config`
Ubuntu 24.04:  `sudo apt install libgtk-4-dev libgtksourceview-5-dev pkg-config`

## Recent Changes
- 002-syntax-highlight: Added Go 1.22+ (`CGO_ENABLED=1`, `CC=clang`)
- 001-boop-core: Initial goop application (GTK4, goja, Boop script compatibility)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
