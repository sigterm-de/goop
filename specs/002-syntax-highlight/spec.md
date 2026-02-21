# Feature Specification: Editor Syntax Highlighting

**Feature Branch**: `002-syntax-highlight`
**Created**: 2026-02-21
**Status**: Draft
**Input**: User description: "in the editor, it would be great to support syntax highlighting. There are some scripts which format eg JSON and it would be great to add highlighting afterwards at least. Maybe we can guess the content of the window and apply a syntax if we are certain. there should be a setting to allow/disallow this detection/guessing."

---

## Effort Challenge

> **Is the effort too high?**
>
> **Short answer: No — this is a good fit for the project.**
>
> The editor already uses GtkSourceView, a component that ships with built-in, production-quality syntax highlighting for dozens of languages (JSON, XML, HTML, YAML, SQL, Markdown, and many more). Enabling highlighting for a given language is a single operation on the existing editor buffer — no additional rendering pipeline or third-party library is needed.
>
> The work that remains is:
>
> 1. **Content detection heuristics** — lightweight pattern matching to identify common structured formats (JSON, XML/HTML, YAML, CSV, SQL, Markdown). This is ~100–150 lines of straightforward logic, not a research problem.
> 2. **Trigger point** — running detection after a script executes, using the transformed output as input.
> 3. **Settings integration** — adding a single toggle to the existing Preferences dialog, which is already wired up and extensible.
>
> The main risk is false positives (wrong language applied). This is mitigated by using **high-confidence detection only**: if the content does not clearly match a known format, no highlighting is applied, and the editor reverts to plain text. The feature is opt-in via a preference and has no impact on the transform pipeline itself.
>
> **Conclusion**: Low effort, high visibility improvement. Recommended to proceed.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Highlighting after script run (Priority: P1)

A user runs a formatting script on arbitrary text — for example a "Format JSON" script. After the script succeeds, the output in the editor is valid, colourful JSON with tokens visually distinguished (strings, numbers, keys, punctuation). The user did not have to do anything to trigger this; it happened automatically.

**Why this priority**: This is the primary motivation for the feature and the use case that delivers immediate value with no user friction.

**Independent Test**: Run a JSON-formatting script against raw JSON input and verify that the editor buffer switches to JSON syntax highlighting without any additional user action.

**Acceptance Scenarios**:

1. **Given** the editor contains plain text and auto-detection is enabled, **When** the user runs a script that produces valid JSON output, **Then** the editor applies JSON syntax highlighting to the result.
2. **Given** the editor contains plain text and auto-detection is enabled, **When** the user runs a script that produces valid XML output, **Then** the editor applies XML/HTML syntax highlighting to the result.
3. **Given** the editor contains plain text and auto-detection is enabled, **When** the user runs a script that produces output that does not clearly match any known format, **Then** the editor displays plain text with no highlighting applied.
4. **Given** the editor currently shows highlighted JSON, **When** the user edits the content manually so it is no longer valid JSON, **Then** the highlighting is not automatically removed (it remains until the next script run or explicit clear).
5. **Given** the editor shows highlighted JSON, **When** the user runs a second script whose output is plain text (not a detectable format), **Then** JSON highlighting is cleared and the syntax zone of the status bar becomes empty.

---

### User Story 2 - Disabling auto-detection (Priority: P2)

A user who prefers a distraction-free, plain-text editing experience can turn off automatic syntax detection in Preferences. Once disabled, scripts run normally and the editor never applies or changes any highlighting — the feature is completely silent.

**Why this priority**: Some users will find unsolicited visual changes disruptive. The opt-out must be easy and persistent.

**Independent Test**: Disable auto-detection in Preferences, run a JSON-formatting script, and verify that the editor buffer shows no syntax highlighting.

**Acceptance Scenarios**:

1. **Given** auto-detection is disabled in Preferences, **When** a script produces JSON output, **Then** no syntax highlighting is applied.
2. **Given** auto-detection is disabled, **When** the user re-enables it and runs a script producing JSON output, **Then** highlighting is applied correctly.
3. **Given** the preference is saved and the application is restarted, **When** the user opens goop, **Then** the auto-detection toggle reflects the previously saved value.

---

### User Story 3 - Highlighting resets on editor clear (Priority: P3)

When the user clears the editor content (e.g., selects all and deletes, or the editor is reset), any active syntax highlighting is also cleared so the editor returns to a neutral plain-text state, ready for new input.

**Why this priority**: Without this, stale highlighting from a previous script run could persist and be confusing.

**Independent Test**: Apply a JSON-formatting script, verify highlighting is active, then clear the editor content and verify no highlighting remains.

**Acceptance Scenarios**:

1. **Given** the editor has JSON highlighting active, **When** the editor content is fully cleared (zero characters), **Then** syntax highlighting is removed.
2. **Given** the editor has JSON highlighting active, **When** the user pastes entirely new unrelated content, **Then** highlighting is not automatically re-evaluated until a script is next run.

---

### Edge Cases

- What happens when a script produces output that is valid JSON but also valid YAML (e.g., `"hello"`)? → The detector must pick JSON only when it is the more specific match (e.g., starts with `{` or `[`); ambiguous plain scalars receive no highlighting.
- What happens if the detected language has no corresponding syntax definition in the installed GtkSourceView data? → The editor silently falls back to plain text; no error is shown.
- What happens if a script fails or times out? → Highlighting is not changed; the editor content and highlighting state from before the run are preserved.
- What happens with very large outputs (e.g., a 10 MB JSON blob)? → Detection must complete without perceptible delay; if analysis exceeds a short time budget it is skipped and no highlighting is applied.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The editor MUST support rendering syntax-highlighted content for at minimum the following formats: JSON, XML, HTML, YAML, CSV, SQL, Markdown.
- **FR-002**: After every successful script execution, the system MUST re-run format detection on the resulting editor content (when auto-detection is enabled), regardless of any previously active highlighting. Each execution produces an independent detection result that may keep, change, or clear the current highlighting.
- **FR-003**: Detection MUST use a two-tier approach: (1) a cheap structural heuristic gates candidate formats (e.g. content starts with `{`/`[` for JSON; contains `<?xml` or `<html` for XML/HTML; begins with `---` for YAML); (2) a quick validation pass confirms the candidate (e.g. the content parses without errors as that format). Highlighting is applied only when BOTH tiers pass. Content that passes neither tier MUST result in plain text with no highlighting applied.
- **FR-004**: When detection is disabled in Preferences, the system MUST NOT examine content for format detection or change editor highlighting at any time.
- **FR-005**: The auto-detection preference MUST be persisted across application restarts.
- **FR-006**: When the editor content is fully cleared (empty buffer), the system MUST remove any active syntax highlighting.
- **FR-007**: Syntax highlighting MUST NOT interfere with existing editor behaviour: text editing, copy/paste, undo, script execution, and the transform pipeline must continue to function identically regardless of whether highlighting is active.
- **FR-008**: The detection and highlighting step MUST complete without perceptible delay after script execution; if analysis cannot finish within a short time budget it MUST be skipped silently.
- **FR-009**: The Preferences dialog MUST include a clearly labelled toggle for enabling or disabling automatic syntax detection.
- **FR-010**: The status bar MUST be composed of two independent zones: (1) a **notification zone** (left-aligned) that shows transient event messages (script success, errors, idle hint) and auto-reverts as before; (2) a **syntax zone** (right-aligned) that shows the name of the currently active syntax language when highlighting is active, and displays nothing (no text, no placeholder) when no highlighting is applied. The two zones operate independently — a notification message MUST NOT replace or obscure the syntax zone, and syntax state changes MUST NOT affect the notification zone.

### Key Entities

- **SyntaxLanguage**: A named highlighting definition (e.g. "JSON", "XML") provided by the system's source-view component, identified by a stable language ID string.
- **DetectionResult**: The outcome of analysing editor content — either a matched SyntaxLanguage with a confidence level, or a "no match" result.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After running a script that produces JSON, XML, HTML, or YAML output, syntax highlighting is visually applied and the detected language name appears in the right-aligned syntax zone of the status bar within the same UI refresh cycle — no additional user action required. A concurrent notification message in the left zone must not displace or hide the syntax indicator.
- **SC-002**: The false-positive rate (wrong language highlighted) must be zero for any content that does not unambiguously match a supported format.
- **SC-003**: The Preferences toggle for auto-detection is reachable in under 3 clicks from the main window and its label clearly communicates the feature's purpose.
- **SC-004**: All existing editor functions (undo, copy/paste, script execution, text editing) pass their current test suites with highlighting active or inactive.
- **SC-005**: Detection adds no perceptible delay to the script-run completion feedback (target: under 50 ms for content up to 1 MB).

---

## Assumptions

- GtkSourceView language definitions for the listed formats (JSON, XML, HTML, YAML, CSV, SQL, Markdown) are available on all supported platforms via the system's `gtksourceview5-data` package — no bundling of language files is needed.
- Detection is triggered only after a successful script execution, not on every keystroke or paste — this keeps the feature lightweight and avoids unexpected mid-editing flicker.
- Syntax highlighting state is ephemeral: it is never saved to disk and does not travel with the text content.
- The feature does not expose a way for scripts themselves to declare or set the output language; that remains a potential future enhancement.

---

## Clarifications

### Session 2026-02-21

- Q: Should the detected language be surfaced somewhere in the UI so the user knows why the editor looks different? → A: Show detected language name in the status bar while highlighting is active; cleared when highlighting is removed.
- Q: What constitutes "high confidence" detection — strict parsing, structural heuristics, or a combination? → A: Two-tier — use cheap structural heuristics as a first gate; only apply highlighting if a quick validation pass also succeeds.
- Q: When a script runs over already-highlighted content, should detection re-run, skip, or clear-then-detect? → A: Always re-run detection fresh after every successful script execution; result may keep, change, or clear existing highlighting.
- Directive: Split the status bar into two distinct zones — left zone for events and notifications (existing behaviour); right zone for the detected syntax type (persistent, independent of the notification lifecycle).
- Q: What does the syntax zone display when no language is active? → A: Completely empty — no text, no placeholder; the zone shows nothing when no highlighting is applied.
