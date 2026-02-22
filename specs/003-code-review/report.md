# goop Code Review Report

Generated: 2026-02-22
Reviewer: Claude Sonnet 4.6
Scope: Go best practices · GTK4 best practices · UI/UX · Security of external script execution

---

## Findings

### Go Best Practices

#### G-01 · HIGH · Data race on `timedOut` — `executor.go:91–108`

`timedOut` is written from the `time.AfterFunc` goroutine and the context goroutine, then read
from the Execute goroutine with no synchronization. `vm.Interrupt()` uses an internal atomic, but
that does not create a happens-before guarantee for the caller's own bool. The Go race detector
will flag this. Replace with `atomic.Bool`.

```go
// current — racy
timedOut := false
timer := time.AfterFunc(timeout, func() {
    timedOut = true        // written in timer goroutine
    vm.Interrupt(errTimeout)
})
// ... later
return e.runError(runErr, timedOut, ...)  // read in execute goroutine
```

#### G-02 · LOW · Custom `itoa` / `formatPt` — `settings.go:35–54`

`itoa` reinvents `strconv.Itoa`, and `formatPt` reimplements `strconv.FormatFloat` with ad-hoc
rounding. Use stdlib:
- `strconv.Itoa(n)`
- `strconv.FormatFloat(v, 'f', -1, 64)` (drops trailing zeros automatically in Go 1.22+)

#### G-03 · LOW · `Reset()` calls `library.All()` twice — `scriptpicker.go:314–318`

`setScripts` already overwrites `sp.allScripts` internally. The second assignment in `Reset` is
redundant dead work.

```go
sp.setScripts(sp.library.All())   // sets sp.allScripts inside setScripts
sp.allScripts = sp.library.All()  // redundant second call
```

#### G-04 · INFO · Log file never closed — `logging.go`

`logFile` is opened in `InitLogger` but there is no `Close()` function. The OS reclaims it on
exit, but the package should export a `Close() error` for graceful shutdown and parallel
testability.

#### G-05 · MEDIUM · No size limit on user script files — `loader.go:111`

`os.ReadFile(absPath)` has no bound. A file of several hundred megabytes placed in
`~/.local/share/goop/scripts/` would be read entirely into RAM. Add a stat-before-read guard
(e.g., skip files larger than 1 MB).

#### G-06 · INFO · `context.Background()` makes cancellation a dead path — `scriptpicker.go:346`

The scriptpicker always passes `context.Background()` to `Execute`. The context-cancellation
goroutine in `executor.go:99–108` is never triggered in the running application. Either pass a
real cancelable context tied to the window lifecycle, or remove the ctx goroutine from the
executor and rely solely on the `time.AfterFunc` timer.

#### G-07 · INFO · Package-level mutable globals in `css.go` — lines 15–20

`fontCSSProvider` and `gnomeInterfaceSettings` are package-level vars mutated at runtime. This
couples initialization order implicitly and makes parallel testing impossible. Moving them into a
struct owned by `ApplicationWindow` would be cleaner.

#### G-08 · MEDIUM · `btoa` diverges from the browser API — `executor.go:229–235`

The browser's `btoa()` encodes a "binary string" (Latin-1, one byte per character). The current
implementation converts Go's UTF-8 string directly to bytes, so `btoa("café")` returns different
output than Chromium/Boop. Existing Boop scripts that call `btoa` on non-ASCII text will silently
produce wrong base64. The spec-compliant behaviour is to throw `InvalidCharacterError` on
characters > U+00FF, letting callers encode to UTF-8 first via `encodeURIComponent`.

#### G-09 · LOW · Preferences not sanitized on load — `preferences.go:59–63`

If the JSON is valid but contains a blank `ScriptPickerShortcut`, it is stored and later passed to
`SetAccelsForAction(…, []string{""})`, which silently no-ops, leaving the picker unreachable by
keyboard. Add a `sanitize()` step after unmarshal that falls back to the default for
empty/unparseable accelerator strings.

---

### GTK4 Best Practices

#### T-01 · LOW · Custom undo bypasses `GtkSourceBuffer`'s undo stack — `editor.go:47–110`

**Resolved.** The custom single-level snapshot (`SaveUndoSnapshot` / `Undo`) has been removed
entirely. `Editor.Undo()` now delegates to `gtk.TextBuffer.CanUndo()` / `.Undo()`, giving
unlimited multi-level undo. `SetMaxUndoLevels(0)` is set in `NewEditor()` to remove the 200-step
cap. The Ctrl+Z intercept in `window.go` was removed so GtkSourceView's own shortcut controller
handles undo natively. `buffer.ConnectUndo()` drives a status bar hint after each undo step.

#### T-02 · LOW · `ConnectChanged` allocates full text on every keystroke — `window.go:74–79`

```go
w.editor.View.Buffer().ConnectChanged(func() {
    if w.editor.GetFullText() == "" {  // allocates a string copy on every edit
```

Replace `GetFullText() == ""` with `e.buffer.CharCount() == 0`. No allocation, correct semantics.

#### T-03 · MEDIUM · Settings dialog has no Escape-to-close — `settings.go:209–233`

`keyCtrl` only handles key events when `capturing == true`. Pressing Escape when not in capture
mode does nothing. Add an unconditional `if keyval == gdk.KEY_Escape && !capturing { win.Close()
}` branch. This is standard GNOME HIG behaviour for modal windows.

#### T-04 · LOW · Settings dialog missing `SetDestroyWithParent` — `settings.go:132–136`

If the parent window closes before the settings dialog, the dialog becomes an orphaned top-level
window. Add `win.SetDestroyWithParent(true)`.

#### T-05 · LOW · Application ID uses non-standard format — `app.go:21`

```go
"org.codeberg.daniel_ciaglia.go_boop"
```

D-Bus application IDs must use only alphanumerics and hyphens in each component, and should match
the Codeberg organization. A cleaner ID: `org.codeberg.sigterm-de.goop`.

#### T-06 · LOW · `showFatalError` creates an unclosable window — `app.go:94–103`

The error window has no Close button and no keyboard shortcut. Users must force-quit the
application. Use `gtk.NewAlertDialog` (GTK 4.10+), or at minimum add a key controller with
Escape → `app.Quit()` and a Close button.

---

### UI/UX

#### U-01 · MEDIUM · No visual feedback during script execution

The editor goes read-only (`SetEditable(false)`) but there is no spinner, progress indicator, or
cursor change. Users running a slow script will see a frozen interface with no explanation.
A `gtk.Spinner` in the header bar or status bar, activated during execution, would fix this.

#### U-02 · MEDIUM · Error messages auto-dismiss after 5 seconds — `statusbar.go`

JavaScript exception messages can be long and technical. A 5-second window is often insufficient
for the user to read, understand, and act on an error before it reverts to the idle hint. Errors
should persist until the next user action, or at minimum have a longer timeout than info messages.

#### U-03 · LOW · "No matching scripts" is an activatable list row — `scriptpicker.go:173–178`

```go
noMatch := gtk.NewLabel("No matching scripts")
sp.listBox.Append(noMatch)
```

GTK wraps this in an implicit `ListBoxRow`, making it appear selectable and respond to Enter. The
bounds check prevents a crash but the visual affordance is wrong. Use a separate widget placed
outside the `ListBox`, or add a `ListBoxRow` with `row.SetActivatable(false)` and
`row.SetSelectable(false)`.

#### U-04 · LOW · Unicode check mark in status bar — `scriptpicker.go:375`

`"✓ " + result.ScriptName + " applied"` uses U+2713 which may not render on all system fonts.
The green CSS class already communicates success; omit the character or use a GTK symbolic icon.

#### U-05 · INFO · Single-level undo not communicated to the user

**Resolved** (as part of T-01). Undo is now unlimited and native. `buffer.ConnectUndo()` in
`window.go` shows "Undone" after each step and "Nothing more to undo" when the stack is empty.

#### U-06 · INFO · Editor locked without cursor-style change

When `SetEditable(false)` is set during execution the mouse cursor does not change. Setting a
`progress` cursor on the view during execution would improve perceived feedback.

---

### Security of External Script Execution

#### S-01 · MEDIUM · `eval()` not sandboxed — `executor.go:48–55`

The poisoned-globals list blocks network and timer APIs but leaves `eval()` intact. `eval()`
executes arbitrary strings with access to the current local scope, which allows code obfuscation
and could complicate future sandbox audits. Add `vm.Set("eval", goja.Undefined())`.

`new Function(code)()` is a related vector but is intentionally NOT poisoned: it creates closures
in the global scope only (no local variable access) and legitimate Boop-compatible libraries such
as `node-forge` rely on it internally. Poisoning `Function` breaks these scripts.

#### S-02 · MEDIUM · No memory limit for scripts — `executor.go`

The 5-second CPU timeout is enforced but a script can allocate unbounded memory:

```javascript
function main(state) {
    let s = "x";
    while (true) s += s;  // ~30 iterations → exhausts RAM
}
```

goja provides no built-in memory limit. The only mitigation is the OS OOM killer. Consider a
goroutine that periodically checks `runtime.ReadMemStats` and interrupts the VM if heap growth
exceeds a threshold.

#### S-03 · LOW · `state.selection` is mutable from JavaScript — `executor.go:191–193`

```go
selObj := vm.NewObject()
selObj.Set("start", state.SelectionInfo.Start)
selObj.Set("end", state.SelectionInfo.End)
```

This creates a plain mutable JS object. A script can write `state.selection.start = -999`.
The mutation is never read back into `ExecutionResult` so there is no functional impact, but it
may surprise script authors who expect `state.selection` to be read-only. Expose it with
`DefineDataProperty` and writable = `FLAG_FALSE`.

#### S-04 · LOW · Log file location is XDG_CONFIG_HOME — `logging.go:55`

```go
xdg.ConfigFile(filepath.Join(appName, appName+".log"))
```

Per the XDG spec, `$XDG_CONFIG_HOME` is for user-editable configuration files, not
runtime-generated logs. Logs belong in `$XDG_STATE_HOME` (`~/.local/state/goop/goop.log`).
`github.com/adrg/xdg` exposes `xdg.StateFile()` for exactly this purpose.

#### S-05 · INFO · Script source content held in memory from startup

All user script source code is loaded into `[]Script.Content` at startup and retained for the
process lifetime. If a user accidentally copies a file with sensitive content into the scripts
directory it will sit in process memory. This reinforces the case for the size limit (G-05).

---

## Summary

| ID   | Category | Severity | Location                    | Title                                      |
|------|----------|----------|-----------------------------|---------------------------------------------|
| G-01 | Go       | HIGH     | `executor.go:91`            | Data race on `timedOut`                    |
| S-02 | Security | MEDIUM   | `executor.go`               | No memory limit for scripts                |
| G-08 | Go       | MEDIUM   | `executor.go:229`           | `btoa` Latin-1 divergence from browser API |
| G-05 | Go       | MEDIUM   | `loader.go:111`             | No user script size limit                  |
| S-01 | Security | MEDIUM   | `executor.go:48`            | `eval`/`Function` not poisoned             |
| T-03 | GTK4     | MEDIUM   | `settings.go:209`           | No Escape-to-close on settings dialog      |
| U-01 | UX       | MEDIUM   | `scriptpicker.go:329`       | No busy indicator during execution         |
| U-02 | UX       | MEDIUM   | `statusbar.go`              | Errors auto-dismiss too fast               |
| G-09 | Go       | LOW      | `preferences.go:59`         | Preferences not sanitized on load          |
| T-01 | GTK4     | LOW      | `editor.go:47`              | Undo bypasses `NotUndoableAction`          |
| T-02 | GTK4     | LOW      | `window.go:75`              | `GetFullText()` allocated on every keystroke |
| T-04 | GTK4     | LOW      | `settings.go:132`           | Missing `SetDestroyWithParent`             |
| T-05 | GTK4     | LOW      | `app.go:21`                 | Non-standard app ID format                 |
| T-06 | GTK4     | LOW      | `app.go:94`                 | Fatal error window not closable            |
| G-02 | Go       | LOW      | `settings.go:35`            | Custom `itoa`/`formatPt` vs stdlib         |
| G-03 | Go       | LOW      | `scriptpicker.go:317`       | Double `library.All()` in `Reset`          |
| S-03 | Security | LOW      | `executor.go:191`           | `state.selection` mutable from JS          |
| S-04 | Security | LOW      | `logging.go:55`             | Log in XDG_CONFIG instead of XDG_STATE    |
| U-03 | UX       | LOW      | `scriptpicker.go:173`       | "No match" label is activatable list row   |
| U-04 | UX       | LOW      | `scriptpicker.go:375`       | Unicode check mark may not render          |
| G-04 | Go       | INFO     | `logging.go`                | Log file never explicitly closed           |
| G-06 | Go       | INFO     | `scriptpicker.go:346`       | `context.Background()` cancellation dead path |
| G-07 | Go       | INFO     | `css.go:15`                 | Package-level mutable globals              |
| U-05 | UX       | INFO     | `editor.go`                 | Silent double-Undo failure                 |
| U-06 | UX       | INFO     | `scriptpicker.go:329`       | No cursor-style change when editor locked  |
| S-05 | Security | INFO     | `loader.go`                 | Script source held in memory from startup  |
