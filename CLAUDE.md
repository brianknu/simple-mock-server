# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -v ./...

# Run tests
go test -v ./...

# Run the server with TUI (default)
go run cmd/api/main.go -d /path/to/mocks -p 8081

# Run the server in headless mode (no TUI, logs to stdout)
go run cmd/api/main.go -d /path/to/mocks -p 8081 --no-tui
```

## Architecture

**simple-mock-server** is a mock HTTP server with a terminal UI (TUI) for creating, editing, and monitoring mocks in real time.

### Entry point and modes

`cmd/api/main.go` parses flags (`-p` port, `-d` directory, `--no-tui`) and runs in one of two modes:
- **TUI mode** (default): creates a `server.Server`, starts HTTP in a goroutine, runs bubbletea on the main goroutine. The TUI and HTTP server share state through the `Server` struct.
- **Headless mode** (`--no-tui`): loads mocks once, registers routes on a `DynamicMux`, serves HTTP directly. Requests log to stdout via the standard logger. No selector or capture — first matching mock is used.

### Core packages

- `internal/utils/` — resolves mock directory: `-d` flag takes precedence over `SIMPLE_MOCKS_LOCATION` env var.
- `internal/mock/` — `Mock` struct + `LoadMocksFromFS(dir)` reads all `.json` files. `SaveMockToFS(dir, mock)` writes a mock back to disk, generating a unique filename if the base name already exists (`GET_api_users.json` → `GET_api_users_2.json`). Each mock tracks its `SourceFile`.
- `internal/router/` — `DynamicMux` is a `sync.RWMutex`-protected handler map. `RegisterMocks(mux, mocks, RouteConfig)` groups mocks by path; each handler filters by verb, then: if multiple matches or capture mode is on it calls `RouteConfig.Selector`; selector returning `-2` calls `RouteConfig.Fallback` (capture flow); disabled mocks are skipped entirely. `RouteConfig.CaptureCheck` lets the server signal whether capture mode is active.
- `internal/server/` — `Server` owns mocks (behind `sync.RWMutex`), the `DynamicMux`, the `http.Server`, `RequestLog`, `PendingCh`, and `SelectionCh`. Key methods: `LoadAndRegister`, `Start`, `Stop`, `AddMock`, `UpdateMock`, `DeleteMock`, `ToggleMock`, `ReloadMocks`. `handleCapture` is a reusable method that blocks the HTTP handler until the TUI responds via `PendingCh`. `selectMock` is the `MockSelector` — it sends a `SelectionRequest` (with `CaptureMode` flag) to `SelectionCh` and blocks for the response. `RequestLog` is a ring buffer (capacity 1000) with channel-based pub/sub.

### TUI package (`internal/tui/`)

Built with **bubbletea** (Elm architecture), **bubbles** (text inputs, textareas), and **lipgloss** (styling). Color palette is Dracula-inspired (hex colors).

**Files:**
- `tui.go` — top-level `Model` with two navigable tabs (Mocks, Request Log). The Create/Edit form and mock picker are overlays, not tabs — they are activated by setting `activeTab = tabMockForm` or `mockPicker.active = true`. Handles global keybindings and all inter-component message routing.
- `keys.go` — all keybinding definitions. `inputActive()` gates single-char keys; it now also returns true when the mock list detail overlay is open.
- `styles.go` — lipgloss style constants. `methodStyle(verb)` returns per-HTTP-method colors. `statusStyle(code)` returns color by HTTP status class (2xx green, 3xx yellow, 4xx orange, 5xx red). **Critical:** always pad columns with plain strings (`fmt.Sprintf("%-Ns", plain)`) *before* applying lipgloss styles, then concatenate. Never pass already-styled strings into `fmt.Sprintf` format directives — ANSI escape codes inflate byte counts and break column alignment.
- `mocklist.go` — table view of loaded mocks. `d` opens a detail overlay (method, paths, status, delay, file, headers, body). `space` toggles mock enabled/disabled. `x` deletes. Detail overlay blocks global keys via `inputActive()`.
- `mockform.go` — create/edit form. Opens as an overlay over the Mocks tab (not a separate tab). `esc` always returns to Mocks tab.
- `mockpicker.go` — interactive overlay shown when multiple mocks match a request, or when capture mode is on and an existing route is hit. Shows existing mocks + optional "+ Define new mock" row (when `CaptureMode` is true). Returns `-2` for "define new" which triggers the capture flow. **Important:** `selectionDoneMsg` always arrives *after* `mockPicker.active` has been set to false, so it must be handled in the main switch of `tui.go`, not inside the picker-active block.
- `requestlog.go` — live request log with cursor navigation. `enter` opens a detail overlay with full request body (pretty-printed if JSON). `y` copies the body to clipboard via `atotto/clipboard`. Request bodies are always logged (no opt-in flag).

**Concurrency model:**
- Main goroutine → bubbletea (owns stdin/stdout)
- Background goroutine → `http.Server.ListenAndServe()`
- Per-request goroutine → handler may block on `SelectionCh` (mock picker) or `PendingCh` (capture form), then writes to `RequestLog`
- bubbletea cmds → `waitForEntry()`, `waitForPending()`, `waitForSelection()` each block on their respective channel and return a message when data arrives. They must be re-dispatched after each message is consumed.

**Key TUI design decisions:**
- `waitForSelection()` is re-dispatched from `selectionDoneMsg` in the main switch (not the picker-active block) because bubbletea updates model state before delivering the cmd's message.
- `waitForPending()` is started when capture mode is toggled on and re-dispatched from `mockSavedMsg`. No extra dispatch needed from `selectionDoneMsg`.
- Column alignment: pad with plain strings first, then style, then concatenate — never use `fmt.Sprintf("%-Ns", styledString)`.
- Tab switching blurs all inputs ("browse mode"). Press `Enter` to engage form inputs.

### Mock JSON format

```json
{
    "paths": ["/api/endpoint"],
    "verb": "POST",
    "body": { "key": "value" },
    "headers": { "Content-Type": "application/json" },
    "status": 200,
    "response_time": 100,
    "disabled": false
}
```

`response_time` is a delay in milliseconds. `disabled: true` skips the mock during route registration without deleting it. Request bodies are always captured. `print_request_body` no longer exists.

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework (Elm architecture)
- `github.com/charmbracelet/bubbles` — TUI components (textinput, textarea)
- `github.com/charmbracelet/lipgloss` — terminal styling/layout
- `github.com/atotto/clipboard` — clipboard access for copying request bodies
