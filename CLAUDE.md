# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -v ./...

# Run tests
go test -v ./...

# Run a single test
go test -v ./internal/router/...

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
- **Headless mode** (`--no-tui`): loads mocks once, registers routes on a `DynamicMux`, serves HTTP directly. Requests log to stdout via the standard logger.

### Core packages

- `internal/utils/` — resolves mock directory: `-d` flag takes precedence over `SIMPLE_MOCKS_LOCATION` env var.
- `internal/mock/` — `Mock` struct + `LoadMocksFromFS(dir)` reads all `.json` files. `SaveMockToFS(dir, mock)` writes a mock back to disk (used by the TUI create/edit form). Each mock tracks its `SourceFile`.
- `internal/router/` — `DynamicMux` is a `sync.RWMutex`-protected handler map that supports runtime route add/remove/clear. `RegisterMocks(mux, mocks, logger)` registers one handler per path per mock. The `RequestLogger` interface decouples request logging from the router (nil falls back to `log.Printf`).
- `internal/server/` — `Server` struct owns mocks (behind `sync.RWMutex`), the `DynamicMux`, the `http.Server`, and the `RequestLog`. Methods: `LoadAndRegister`, `Start`, `Stop`, `AddMock`, `UpdateMock`, `DeleteMock`, `ReloadMocks`, `Mocks`, `MockCount`. `RequestLog` is a ring buffer (capacity 1000) with channel-based pub/sub for the TUI. Non-blocking sends prevent slow TUI from blocking HTTP handlers.

### TUI package (`internal/tui/`)

Built with **bubbletea** (Elm architecture), **bubbles** (text inputs, textareas), and **lipgloss** (styling).

**Structure:**
- `tui.go` — top-level `Model` with tab management (Mocks, Request Log, Create/Edit). Handles global keybindings, delegates to sub-models per tab. `newLogEntryMsg` is handled at the top level so log entries are captured regardless of active tab.
- `keys.go` — all keybinding definitions. Single-char keys (`q`, `?`, `n`, `e`, `d`, `r`, `c`) only work when no text input is focused (`inputActive()` check). `j`/`k` were removed from Up/Down to avoid conflicts with text input.
- `styles.go` — lipgloss style constants, including per-HTTP-method color coding.
- `mocklist.go` — table view of loaded mocks with cursor navigation.
- `mockform.go` — create/edit form with verb selector (left/right arrows), text inputs for paths/status/delay, textareas for headers and JSON body. New mocks default to `Content-Type: application/json` header. Uses `↑`/`↓` to cycle fields. `esc` cancels and returns to Mocks tab.
- `requestlog.go` — live scrollable request log. Subscribes to `server.RequestLog` channel via `waitForEntry()` tea.Cmd pattern.

**Concurrency model:**
- Main goroutine → bubbletea (owns stdin/stdout)
- Background goroutine → `http.Server.ListenAndServe()`
- Per-request goroutine → handler writes to `RequestLog` channel
- bubbletea cmd → reads from channel, dispatches `newLogEntryMsg`

**Key TUI design decisions:**
- Tab switching (`tab`/`shift+tab`) puts tabs in "browse mode" — text inputs are blurred. Press `Enter` to engage form inputs. This prevents getting stuck in a tab.
- `inputActive()` gates whether single-char keys are treated as commands or forwarded to text inputs.
- `esc` from Create/Edit always returns to Mocks tab regardless of form state.

### Mock JSON format

```json
{
    "paths": ["/api/endpoint"],
    "verb": "POST",
    "body": { "key": "value" },
    "headers": { "Content-Type": "application/json" },
    "status": 200,
    "print_request_body": true,
    "response_time": 100
}
```

`print_request_body: true` logs the incoming request body. `response_time` is a delay in milliseconds before responding.

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework (Elm architecture)
- `github.com/charmbracelet/bubbles` — TUI components (textinput, textarea)
- `github.com/charmbracelet/lipgloss` — terminal styling/layout
