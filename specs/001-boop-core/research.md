# Research: goop — Linux Text Transformation Tool

**Date**: 2026-02-20
**Branch**: `001-boop-core`

---

## 1. GTK4 Go Bindings (gotk4)

**Decision**: Use `github.com/diamondburned/gotk4` for GTK4 bindings and
`github.com/diamondburned/gotk4-sourceview` for GtkSourceView.

**Key imports**:
```
github.com/diamondburned/gotk4/pkg/gtk/v4          # widgets
github.com/diamondburned/gotk4/pkg/glib/v2          # main loop, IdleAdd
github.com/diamondburned/gotk4/pkg/gio/v2           # Application base
github.com/diamondburned/gotk4-sourceview/pkg/gtksource/v5  # editor
```

**Rationale**: gotk4 is the only maintained, generated-from-GIR GTK4 binding for Go.
It produces idiomatic Go with typed signal connectors (`ConnectClicked(func(){...})`).
GtkSourceView ships as a separate module (`gotk4-sourceview`) — it is an additional
system library dependency (`libgtksourceview-5`).

**Critical finding — FR-008 conflict**: gotk4 requires CGo and produces a binary
that dynamically links against `libgtk-4.so`, `libgtksourceview-5.so`, and transitive
libraries. The original spec's FR-008 ("no external runtime dependencies") is
incompatible with GTK4 itself. The user's own technical description qualifies this:
"no runtime dependencies *beyond GTK4 and GtkSourceView system libraries*". FR-008 is
amended accordingly in the plan (see Complexity Tracking).

**Thread safety**: All GTK API calls MUST occur on the main thread. Use
`glib.IdleAdd(func() bool { ...; return false })` to marshal results from goroutines
back to the UI thread.

**Script picker**: Use a `gtk.Overlay` + `gtk.Revealer` for the in-window script panel
(no secondary window required). `gtk.Popover` is an alternative for smaller panels.

**GTK4 minimum version**: 4.6 minimum; 4.10+ recommended; 4.12 ships on Arch/Manjaro.

**System packages (Arch/Manjaro)**: `gtk4`, `gtksourceview5`

**Alternatives considered**:
- Fyne: pure-Go, static binary, but OpenGL-rendered — does not integrate with system
  GTK themes or look native on GNOME/KDE.
- Gio (gioui.org): pure-Go, GPU-rendered, static binary — same theme integration issue.
- Decision: gotk4 accepted with the system library dependency trade-off, which is
  standard practice for Linux GTK apps.

---

## 2. JavaScript Engine (goja)

**Decision**: Use `github.com/dop251/goja` for JavaScript execution and
`github.com/dop251/goja_nodejs/require` for the CommonJS `require()` system.

**Rationale**: goja is a pure-Go ES5.1+ES2015 interpreter with no CGo. It is
production-proven (used in k6, Gitea). It is the only viable pure-Go option that
supports ES6+ features Boop scripts require. The sandbox is clean-room: no `fs`,
`net`, `process`, or `window` globals exist by default — only what you explicitly add.

**Supported ES features**: `let`/`const`, arrow functions, template literals,
destructuring, spread, `async`/`await`, `class`, `Map`/`Set`, optional chaining (`?.`),
nullish coalescing (`??`), `Proxy`/`Reflect`. ESM `import`/`export` is NOT supported —
scripts must use function scope or CommonJS `require()`.

**State object pattern**:
```go
type ScriptState struct {
    FullText  string `json:"fullText"`
    Text      string `json:"text"`
    Selection Selection `json:"selection"`
}
vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
vm.Set("state", vm.ToValue(&state))
```
This maps `state.fullText`, `state.text`, `state.selection`, `state.insert()`, and
`state.postError()` as camelCase JS properties.

**Sandboxing**: Goja has no system access by default. Explicitly poison dangerous
globals: `require`, `process`, `global`, `Buffer`, `setTimeout`, `setInterval`,
`fetch`. Then reinstall a safe `require` via `goja_nodejs/require`.

**Timeout**:
```go
timer := time.AfterFunc(5*time.Second, func() { vm.Interrupt(ErrTimeout) })
defer timer.Stop()
```
Check for `*goja.InterruptedError` in the returned error.

**Module loader**: `require.NewRegistry(require.WithLoader(func(path string) ...))`.
The loader returns source bytes for `@boop/plist` and `@boop/yaml`; all other paths
return `require.ModuleFileDoesNotExistError`.

**Performance**: Sub-10ms for typical string transform scripts. 100ms–1s possible for
character-by-character loops over 1MB+ text. Acceptable for a text utility tool.
Create a new `goja.Runtime` per script invocation (takes ~50µs) for isolation.

**Alternatives considered**:
- otto: No longer actively maintained; ES5 only — fails on many Boop scripts.
- V8 (via v8go): CGo-based, adds CGo complexity on top of gotk4; 2× CGo dependency.
- QuickJS (via go-quickjs): CGo-based; same concern. Rejected to minimize CGo surface.

---

## 3. Boop Script API Contract

**Decision**: Implement the exact Boop script API as defined below. Full backward
compatibility with all upstream Boop scripts is a non-negotiable requirement (SC-002).

**Header block format**: Files MUST start with `/**!` (exactly — not `/**` or `/*!`).
```js
/**!
 * @name          URL Encode
 * @description   Percent-encode all special characters
 * @icon          <i class="fas fa-percentage"></i>
 * @tags          encode,url,percent
 * @bias          -0.5
 */
```
Fields: `@name` (required), `@description` (required), `@icon` (optional, FontAwesome
HTML), `@tags` (optional, comma-separated), `@bias` (optional, float sort weight).

**State object API**:
| Property/Method | Type | Behavior |
|---|---|---|
| `state.fullText` | string RW | Full editor content; writing replaces entire document |
| `state.text` | string RW | Selected text if any; full text if no selection; writing replaces selection or full document |
| `state.selection` | object RO | `{ start: number, end: number }` — 0-based character offsets |
| `state.insert(text)` | function | Insert text at cursor / replace selection |
| `state.postError(msg)` | function | Signal error to host; rollback all state mutations; show msg in UI |

**Entry point**: Scripts MUST define `function main(state) { ... }`. Host calls it.

**Built-in modules**:
- `@boop/plist` → `{ parse(str), stringify(obj), parseBinary(data) }`
  Implemented in Go via `howett.net/plist`.
- `@boop/yaml` → `{ parse(str), stringify(obj) }`
  Implemented in Go via `gopkg.in/yaml.v3`.
  Exposed as goja-callable Go functions registered on the module exports object.

**Globals available**: `JSON`, `Math`, `Date`, `Array`, `String`, `RegExp`,
`parseInt`, `parseFloat`, `encodeURIComponent`, `decodeURIComponent`, `btoa`, `atob`,
`Error`, `Promise` (sync only — no async event loop resolution).
`console.log` MAY be provided and directed to the log file.

**NOT available**: `fetch`, `XMLHttpRequest`, `fs`, `process`, `os`, `setTimeout`,
`setInterval`, `window`, `document`, any `require()` not `@boop/*`.

**Execution is synchronous**: Scripts MUST complete synchronously. `async`/`await`
syntax is permitted by the parser, but async execution (Promise resolution across
ticks) does not occur in a synchronous goja invocation.

**`btoa`/`atob`**: MUST be provided as globals — Boop's Base64 scripts use them and
they are not in the ES core spec. Implement via `encoding/base64` and expose as goja
globals.

**Bundled scripts**: ~60–65 scripts across categories: encoding/decoding, format/
prettify, text transformation, hashing, number conversion, generation/utilities. The
full list is sourced from `IvanMathy/Boop/Boop/Scripts/`.

**Alternatives considered**:
- Implementing only a subset of the API: rejected — SC-002 requires 100% compatibility.
- Using a different module system: rejected — `@boop/plist` and `@boop/yaml` are used
  by real community scripts and must work without modification.

---

## 4. Release Pipeline (goreleaser + go-semantic-release + Woodpecker CI)

**Decision**: `go-semantic-release` + `provider-gitea` plugin for Codeberg;
`goreleaser` v2 for building and packaging; Woodpecker CI for automation.

**Rationale**: go-semantic-release ships as a single Go binary (no Node.js in CI),
has a first-class Forgejo/Gitea provider, and integrates cleanly with goreleaser.
Woodpecker CI is the native CI system for Codeberg.

**Commit format**: Conventional Commits — `feat:`, `fix:`, `perf:`, `BREAKING CHANGE`.
MINOR bump on `feat:`, PATCH on `fix:`/`perf:`, MAJOR on `BREAKING CHANGE` or `!`.

**Archive format**: `.zip` per project requirement. Named:
`goop_{version}_{os}_{arch}.zip`

**Cross-compilation constraint**: CGo + GTK4 system libs make cross-compilation
impractical. Two strategies:
1. Native arm64 CI runner in Woodpecker (recommended) — each arch builds natively.
2. QEMU emulation in Woodpecker (fallback) — 5–10× slower but no extra runner needed.

**Release flow**:
1. Push to `main` → Woodpecker runs lint + test.
2. go-semantic-release analyzes commits, creates semver tag, creates Forgejo release.
3. Tag event triggers goreleaser on amd64 + arm64 runners.
4. goreleaser builds binaries (CGo, native arch), packages as zip, uploads to release.
5. `checksums.txt` generated for both arches.

**GTK4 dependency in releases**: Release description MUST document that GTK4 and
GtkSourceView must be present on the target system. Install instructions for Arch,
Debian/Ubuntu, and Fedora are included in the release body template.

**Alternatives considered**:
- `semantic-release` npm package: requires Node.js in CI; Forgejo adapter less mature.
- GitHub Actions: not available on Codeberg; Woodpecker is the native choice.
- AppImage: bundles GTK4 inside a squashfs; resolves FR-008 literally; deferred to v1+.

---

## 5. Fuzzy Search (sahilm/fuzzy) and XDG Paths (adrg/xdg)

### Fuzzy Search

**Decision**: `github.com/sahilm/fuzzy` v0.1.5.

**Rationale**: Trivially fast at 100–200 items (sub-1ms). API is minimal: `fuzzy.FindFrom(query, source)` returns `[]Match{Str, Index, MatchedIndexes, Score}`. Implement `Source` interface on `[]Script` for direct struct-level matching. Results are pre-sorted by score. No debouncing or goroutines needed — run synchronously in the GTK `ConnectChanged` callback.

**Integration**: On each `searchEntry.ConnectChanged` event, call `fuzzy.FindFrom`,
map result indices back to `[]Script`, and replace the displayed model slice. Use
`gtk.FilterListModel` + `gtk.CustomFilter` or simpler slice replacement depending on
gotk4 model implementation complexity.

**Alternatives considered**: `lithammer/fuzzysearch`, manual substring match. Both
inferior to sahilm/fuzzy's scoring and MatchedIndexes (useful for highlighting matches).

### XDG Paths

**Decision**: `github.com/adrg/xdg` v0.4.0.

**Rationale**: Most complete, spec-correct XDG implementation in Go. Active maintenance.
Auto-creates directories via `xdg.ConfigFile()` helper.

**Path mapping**:
| Resource | XDG call | Default path |
|---|---|---|
| Config dir | `filepath.Join(xdg.ConfigHome, "goop")` | `~/.config/goop/` |
| User scripts | `filepath.Join(xdg.ConfigHome, "goop", "scripts")` | `~/.config/goop/scripts/` |
| Log file | `xdg.ConfigFile("goop/goop.log")` | `~/.config/goop/goop.log` |

Note: Strictly, log files belong in `XDG_STATE_HOME` (`~/.local/state`), but FR-019
explicitly specifies `~/.config/goop/goop.log`, so `ConfigFile` is used.

**Alternatives considered**: `kirsle/configdir` (unmaintained), manual `os.Getenv`
reading (error-prone, misses edge cases like custom `XDG_CONFIG_HOME`).
