# Feature Specification: goop — Linux Text Transformation Tool

**Feature Branch**: `001-boop-core`
**Created**: 2026-02-20
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Apply a Text Transformation (Priority: P1)

A Linux desktop user needs to quickly transform a piece of text — perhaps URL-encoding
a string, formatting JSON, or converting a timestamp. They open goop, paste their
text into the input area, browse or search the script list, click a script, and
immediately see the transformed text replace their input. They copy the result and
continue their work.

**Why this priority**: This is the entire value proposition of the tool. Without the
ability to select a script and transform text, nothing else matters. It must be the
first working slice.

**Independent Test**: Can be fully tested by launching the app, pasting a known string,
applying a bundled script (e.g., "URL Encode"), and verifying the exact expected output
appears in the text area.

**Acceptance Scenarios**:

1. **Given** the app is open with text in the input area, **When** the user clicks a
   script in the list, **Then** the text is replaced with the transformed output
   immediately with no page reload or additional action required.
2. **Given** a script that produces an error (e.g., invalid JSON passed to a JSON
   formatter), **When** the user applies that script, **Then** the original text is
   preserved unchanged and a clear error message is shown to the user.
3. **Given** the app is open with an empty input area, **When** the user applies any
   script, **Then** the app handles the empty input gracefully without crashing.

---

### User Story 2 - Search and Browse the Script Library (Priority: P2)

A user has 100+ scripts available and needs to find "Base64 Decode" quickly. They type
"base64" in the search/filter field and the script list narrows instantly to matching
entries. They click the script and it is applied.

**Why this priority**: The tool ships with all original Boop scripts (60+) plus the
user may add more. Without search, the list becomes unusable. This is the second most
critical feature for day-to-day usability.

**Independent Test**: Can be tested by launching the app with all bundled scripts
loaded, typing a partial script name into the search field, and verifying that only
matching scripts appear in the filtered list.

**Acceptance Scenarios**:

1. **Given** the script list is showing all available scripts, **When** the user types
   a partial name into the search field, **Then** the list updates in real time to show
   only scripts whose name or description contains the typed text (case-insensitive).
2. **Given** the user has filtered the list and then clears the search field, **Then**
   all available scripts are displayed again.
3. **Given** the user types a query that matches no scripts, **Then** a clear
   "no results" message is shown rather than a blank list.

---

### User Story 3 - Add a Custom Transformation Script (Priority: P3)

A developer wants to add a personal script that strips internal company log prefixes.
They place a `.js` file following the Boop script format into their user scripts
directory, relaunch the app, and the new script appears in the list alongside
built-in scripts and can be applied just like any built-in script.

**Why this priority**: Extensibility is a core differentiator. Without it, goop is
a read-only tool for the built-in set. This delivers power-user value without affecting
the P1/P2 stories.

**Independent Test**: Can be tested by placing a known test script into the user scripts
directory, closing and relaunching the app, and verifying the script appears in the
list and produces correct output when applied.

**Acceptance Scenarios**:

1. **Given** a user places a valid `.js` script file in the user scripts directory,
   **When** the application is opened, **Then**
   the custom script appears in the list with its declared name and description.
2. **Given** a user places an invalid or malformed `.js` file in the user scripts
   directory, **When** the application loads scripts, **Then** the invalid script is
   skipped with a warning logged, and the rest of the library still loads successfully.
3. **Given** a custom script has the same name as a built-in script, **When** the list
   is displayed, **Then** both scripts are shown with a visible indicator of which is
   built-in and which is user-provided.

---

### User Story 4 - Install and Run Without Setup (Priority: P4)

A user downloads the goop binary, marks it executable, and runs it. The application
launches and is fully functional without installing any additional runtimes, libraries,
or packages. They can also install it via a package manager or copy the binary to any
Linux machine.

**Why this priority**: Single-binary distribution is an explicit project goal and a key
quality-of-life feature for Linux users accustomed to tools that require complex
installation steps. It also ensures repeatability across versions.

**Independent Test**: Can be tested by copying the binary to a clean Linux environment
with no development tools installed, running it, and confirming the app opens and
scripts execute correctly.

**Acceptance Scenarios**:

1. **Given** a user has downloaded only the goop binary, **When** they execute it,
   **Then** the application opens and all built-in scripts are available with no
   additional setup.
2. **Given** a new release is published, **When** the user replaces the old binary with
   the new one, **Then** the application works immediately with no migration or
   reconfiguration step.

---

### Edge Cases

- What happens when a script takes too long to execute? (e.g., an infinite loop in user
  script) — the system MUST time out and report the issue without hanging the UI.
- How does the system handle very large input text (e.g., >1 MB)? — should warn the user
  but still attempt the transformation.
- What happens if the user scripts directory does not exist or has insufficient
  permissions? — the app MUST start normally using only built-in scripts.
- What happens if two user scripts share the same file name? — the system MUST load both
  but distinguish them.
- How are Unicode, emoji, and multi-byte characters handled? — transformations MUST
  preserve encoding correctness.
- What if the user pastes text and immediately closes the app? — no data is persisted
  between sessions (no implicit save).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST provide a text input area where users can type, paste, and
  edit arbitrary plain text.
- **FR-002**: The tool MUST display a browsable list of all available transformation
  scripts with each script's name and a short description visible.
- **FR-003**: Users MUST be able to apply a script to the current text by selecting it,
  with the result replacing the input text immediately.
- **FR-004**: The tool MUST ship with the complete set of scripts from the original Boop
  project, embedded in the binary so no external files are required.
- **FR-005**: The tool MUST load additional user-provided scripts from a standard
  per-user configuration directory (defaulting to the platform-conventional location,
  e.g., `~/.config/goop/scripts/` on Linux).
- **FR-006**: Users MUST be able to search and filter the script list by name or keyword
  with results updating as they type.
- **FR-007**: Transformation scripts MUST be authored in JavaScript and executed in a
  runtime that supports ES6+ features and is capable of running the complete set of
  original Boop scripts without modification.
- **FR-015**: The script execution environment MUST be sandboxed: scripts MUST have
  access only to the input text and standard JavaScript built-ins. Scripts MUST NOT be
  able to access the file system, network, environment variables, or any other system
  resource. Any attempt to access prohibited APIs MUST fail silently or with a clear
  in-script error rather than crashing the application.
- **FR-019**: Script execution errors and script load warnings MUST be recorded in a log
  file written to the user configuration directory (e.g., `~/.config/goop/goop.log`).
  Each log entry MUST include a timestamp, the script name, and the error message. The
  log file MUST be appended to across sessions (not overwritten on each launch). The log
  file path MUST be shown in the UI error message so users can locate it.
- **FR-018**: While a script is executing, the text input area and script list MUST be
  disabled and a visible progress indicator MUST be displayed. The UI MUST return to
  its normal interactive state immediately when execution completes (success, error, or
  timeout). No cancel action is required in v1.
- **FR-017**: User scripts MUST be loaded once at application startup. The tool MUST NOT
  watch the scripts directory for changes at runtime. To pick up newly added or modified
  scripts, the user MUST restart the application.
- **FR-016**: After a transformation is successfully applied, the tool MUST allow the
  user to undo the transformation with a single action, restoring the text area to its
  exact pre-transformation content. Only one level of undo is required; applying a
  second transformation replaces the undo buffer with the new previous state.
- **FR-008**: The tool MUST be distributed as a single self-contained executable binary
  with no dependency on an external JavaScript runtime, GUI toolkit installation, or
  any other separately-installed package.
- **FR-009**: When a script execution fails or encounters a runtime error, the tool MUST
  preserve the user's original text unchanged and display a human-readable error message
  identifying what went wrong.
- **FR-010**: Script execution MUST time out after a reasonable period (default: 5
  seconds) and report the timeout to the user rather than hanging.
- **FR-011**: The tool MUST provide a mechanism to copy the current text area content to
  the system clipboard.
- **FR-012**: Releases MUST follow a semantic versioning scheme with an automated
  changelog, enabling users and package maintainers to understand the scope of each
  update.
- **FR-013**: The tool MUST visually distinguish built-in scripts from user-provided
  scripts in the script list.
- **FR-014**: The tool MUST function as a standard application window launched from a
  terminal, `.desktop` launcher, or application menu. It MUST NOT require a background
  service, system tray daemon, or global keyboard shortcut registration. Desktop
  environment integration is limited to providing a proper `.desktop` file and
  application icon so the tool appears in app launchers like any other application.

### Key Entities

- **Script**: A JavaScript file with structured metadata (name, description, optional
  tags/category) and a transformation function. Has a source attribute (built-in vs
  user-provided) and an execution status (ready, executing, errored).
- **Script Library**: The full collection of scripts available at runtime, composed of
  built-in scripts (embedded in the binary) and user scripts (loaded from the config
  directory). Supports search/filter operations.
- **Transformation**: A single application of a Script to an input text, yielding an
  output text or an error. Captures input, output, script identity, success/failure,
  and any error message.
- **User Configuration**: The set of user-controlled settings, including the custom
  scripts directory path and any user preferences (e.g., window size, last-used script).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can open the app, paste text, select a script, and view the
  transformed result in under 10 seconds from a cold start.
- **SC-002**: 100% of the original Boop project scripts execute correctly without any
  source modification.
- **SC-003**: Users can locate any built-in script by name by typing 3 or fewer
  characters in the search field.
- **SC-004**: The application is ready for user interaction within 2 seconds of launch
  on standard desktop hardware.
- **SC-005**: A custom script placed in the user scripts directory is available in the
  app without reinstalling or reconfiguring anything.
- **SC-006**: Script execution errors are communicated to the user in plain language
  within 1 second of the failure occurring, with the original text intact.
- **SC-007**: The released binary runs on any modern Linux distribution without
  installing additional packages or runtimes.
- **SC-008**: The project produces a new versioned release with an automated changelog
  for every merged change set, following semantic versioning conventions.

## Clarifications

### Session 2026-02-20

- Q: Should scripts run sandboxed (no network/file system/env access) or with full system access? → A: Sandboxed — scripts have access only to input text and standard JS built-ins; all system APIs are prohibited.
- Q: After a successful transformation, can the user undo it? → A: Single-level undo — one "Undo" action restores the pre-transformation text; applying a second transformation replaces the undo buffer.
- Q: How does the app pick up newly added custom scripts — manual refresh, hot-reload, or restart? → A: Restart only — user scripts are loaded once at startup; the app is restarted to pick up new or modified scripts.
- Q: What does the UI do while a script is executing? → A: Input area and script list are disabled with a visible progress indicator; no cancel button required in v1.
- Q: Where do script errors and load warnings get logged? → A: Shown in the UI and written to a log file in the user config directory (e.g., `~/.config/goop/goop.log`); log path shown in UI error messages.

## Assumptions

- Boop script format is treated as the authoritative contract: any script that runs in
  the original Boop macOS app MUST also run in goop without modification.
- The user scripts directory defaults to `~/.config/goop/scripts/` and is created
  automatically on first launch if it does not exist.
- Built-in scripts are embedded directly into the binary at build time; no separate data
  files are distributed alongside the binary.
- The app targets X11 and Wayland display servers; behavior on other display servers is
  undefined but the app MUST NOT crash.
- Clipboard integration relies on the desktop environment's standard clipboard
  mechanism; no clipboard daemon is required.
- The semantic release process uses conventional commit messages (`feat:`, `fix:`, etc.)
  to determine version bumps and generate changelogs automatically.
- Script execution timeout defaults to 5 seconds and is not user-configurable in v1.
- The log file is written to `~/.config/goop/goop.log` and is appended to across sessions; no log rotation is required in v1.
- No text entered in the app is persisted to disk between sessions.
