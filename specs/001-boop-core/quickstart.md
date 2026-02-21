# Quickstart: goop Development Guide

**Date**: 2026-02-20
**Branch**: `001-boop-core`

---

## System Prerequisites

Install the required system libraries (Arch/Manjaro):

```bash
sudo pacman -S --needed \
    gtk4 \
    gtksourceview5 \
    pkg-config \
    go \
    golangci-lint \
    git
```

Debian/Ubuntu 24.04:

```bash
sudo apt install -y \
    libgtk-4-dev \
    libgtksourceview-5-dev \
    pkg-config \
    golang-go
```

Fedora 39+:

```bash
sudo dnf install -y \
    gtk4-devel \
    gtksourceview5-devel \
    pkg-config \
    golang
```

---

## Build

```bash
# Clone
git clone https://codeberg.org/sigterm-de/goop.git
cd goop

# Install Go dependencies (CGo required)
go mod download

# Build (CGo must be enabled for gotk4)
CGO_ENABLED=1 go build -o goop ./cmd/goop

# Run
./goop
```

The binary dynamically links against `libgtk-4` and `libgtksourceview-5`. These must
be present on the system at runtime (see System Prerequisites above).

---

## Development Workflow

### Run tests

```bash
# Unit tests (all packages) with race detector
go test -race ./...

# Contract tests only
go test -race ./tests/contract/...

# Integration tests only
go test -race ./tests/integration/...

# With coverage report
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

Coverage gate: the CI pipeline fails if coverage drops below 80%.

### Lint

```bash
golangci-lint run ./...
```

All lint errors must be resolved before merging. Configuration is in `.golangci.yml`.

### Format

```bash
gofmt -w .
goimports -w .
```

Unformatted code will be rejected by the CI pipeline.

---

## Project Layout at a Glance

```
cmd/goop/main.go         Entry point — calls app.Run()
internal/app/               GTK Application + ApplicationWindow
internal/ui/                Editor widget, script picker panel, status bar
internal/engine/            JavaScript VM: sandboxed goja + state object
internal/scripts/           Script loading, /**! header parsing, fuzzy library
internal/logging/           XDG log file setup

assets/scripts/             Bundled Boop scripts (embedded via go:embed)
tests/contract/             Script API and library interface contract tests
tests/integration/          End-to-end transformation and loading tests
.desktop/goop.desktop    XDG desktop entry
```

---

## Adding a Built-in Script

1. Place the `.js` file in `assets/scripts/`. It MUST begin with a `/**!` header.
2. Run `go build ./cmd/goop` — the script is embedded at compile time.
3. Run `go test ./tests/contract/...` to verify the script loads without errors.

```js
/**!
 * @name          My Transform
 * @description   Does something useful
 * @tags          example,transform
 */

function main(state) {
    state.text = state.text.toUpperCase();
}
```

---

## Adding a User Script (at Runtime)

1. Place the `.js` file in `~/.config/goop/scripts/`.
2. Restart goop.
3. The script appears in the picker list, labelled as user-provided.

The scripts directory is created automatically on first launch if it does not exist.

---

## Script API Quick Reference

Scripts receive a `state` object. The common patterns:

```js
// Transform entire document:
function main(state) {
    state.fullText = doSomething(state.fullText);
}

// Transform selection (or full text if nothing selected):
function main(state) {
    state.text = doSomething(state.text);
}

// Signal an error (preserves original text):
function main(state) {
    try {
        state.text = JSON.stringify(JSON.parse(state.fullText), null, 2);
    } catch (e) {
        state.postError("Invalid JSON: " + e.message);
    }
}

// Use a bundled module:
var yaml = require('@boop/yaml');
function main(state) {
    state.fullText = yaml.stringify(JSON.parse(state.fullText));
}
```

See `specs/001-boop-core/contracts/script-api.md` for the complete API contract.

---

## CI Pipeline (Woodpecker)

```
push to main:
  lint → go test -race → semantic-release (creates tag if feat/fix commits found)

tag created:
  goreleaser (amd64 native runner) → build binary → package as zip → upload to release
  goreleaser (arm64 native runner) → build binary → package as zip → upload to release
```

Required CI secrets (set in Woodpecker repository settings):
- `CODEBERG_TOKEN` — Codeberg API token with `repo` scope

Required CI environment variables (set in pipeline YAML):
- `GITEA_API=https://codeberg.org/api/v1`

---

## Release Process

goop uses semantic releases driven by conventional commit messages:

| Commit prefix | Version bump |
|---------------|-------------|
| `feat: ...` | MINOR (0.x.0) |
| `fix: ...` | PATCH (0.0.x) |
| `BREAKING CHANGE` in footer | MAJOR (x.0.0) |
| `chore:`, `docs:`, `ci:`, `test:` | No release |

Example commit messages (include task ID in body, not subject):

```
feat(ui): add undo button to main toolbar

Implements FR-016 (single-level undo after transformation).
Closes T042.
```

```
fix(engine): preserve original text on postError when fullText pre-mutated

Closes T055.
```

---

## Runtime Behaviour Validation

After building, verify core flows manually:

```bash
./goop &

# 1. Paste text, select "URL Encode" → text should be percent-encoded
# 2. Apply "JSON Prettify" to invalid JSON → error shown, text preserved
# 3. Place a test script in ~/.config/goop/scripts/, restart → script appears
# 4. Type "base" in search → list filters to base64/baseline scripts
# 5. Apply any script → undo with Ctrl+Z restores previous text
```

Log file location: `~/.config/goop/goop.log`

---

## System Requirements for End Users

The released binary requires these system libraries. Include this in release notes:

| Distro | Install command |
|--------|-----------------|
| Arch/Manjaro | `sudo pacman -S gtk4 gtksourceview5` |
| Debian/Ubuntu 24.04 | `sudo apt install libgtk-4-1 libgtksourceview-5-0` |
| Fedora 39+ | `sudo dnf install gtk4 gtksourceview5` |
