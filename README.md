# simple-mock-server

A local HTTP mock server with an interactive terminal UI. Define mock responses as JSON files, manage them live from the TUI, and capture real requests to create new mocks on the fly.

---

## Features

- **Terminal UI** — browse, create, edit, enable/disable and delete mocks without leaving the terminal
- **Multi-mock per route** — multiple mocks can share the same path + verb; an interactive picker lets you choose which one to serve on each request
- **Capture mode** — intercept unmapped requests and define a response live; the mock is saved to disk automatically. Also works on already-defined routes, letting you add variants without touching any file
- **Enable / disable** — toggle mocks on and off instantly without deleting them
- **Colored status codes** — 2xx green, 3xx yellow, 4xx orange, 5xx red
- **Request log** — live scrollable log of every incoming request
- **Headless mode** — run without the TUI for CI or scripted environments

---

## Installation

```bash
go install github.com/brianknu/simple-mock-server/cmd/api@latest
```

Or build from source:

```bash
git clone https://github.com/brianknu/simple-mock-server
cd simple-mock-server
go build -o simple-mock-server ./cmd/api
```

---

## Usage

```bash
# TUI mode (default)
simple-mock-server -p 8081 -d ./mocks

# Headless mode (logs to stdout, no TUI)
simple-mock-server -p 8081 -d ./mocks --no-tui
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p` | `8081` | Port to listen on |
| `-d` | — | Directory containing mock JSON files |
| `--no-tui` | `false` | Disable the terminal UI |

You can also set the mocks directory via environment variable:

```bash
export SIMPLE_MOCKS_LOCATION=./mocks
simple-mock-server -p 8081
```

---

## Mock format

Each `.json` file in the mocks directory defines one mock:

```json
{
    "paths": ["/api/users", "/api/v2/users"],
    "verb": "GET",
    "status": 200,
    "headers": {
        "Content-Type": "application/json"
    },
    "body": {
        "users": [{ "id": 1, "name": "alice" }]
    },
    "response_time": 150,
    "disabled": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `paths` | `[]string` | One or more URL paths this mock matches |
| `verb` | `string` | HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, …) |
| `status` | `int` | HTTP status code to return |
| `headers` | `object` | Response headers |
| `body` | `any` | JSON response body |
| `response_time` | `int` | Delay in milliseconds before responding |
| `disabled` | `bool` | When `true`, the mock is loaded but not served (omit to default to enabled) |

> Multiple JSON files can define mocks for the same path + verb. The TUI will ask you to pick one on each request.

---

## Keyboard shortcuts

### Mocks tab

| Key | Action |
|-----|--------|
| `n` | Create new mock |
| `e` | Edit selected mock |
| `d` | Show mock details |
| `space` | Enable / disable selected mock |
| `x` | Delete selected mock |
| `r` | Reload mocks from disk |
| `i` | Toggle capture mode |
| `↑` / `↓` | Navigate list |

### Create / Edit form

| Key | Action |
|-----|--------|
| `↑` / `↓` | Cycle between fields |
| `←` / `→` | Cycle HTTP verb (on Method field) |
| `ctrl+s` | Save mock |
| `esc` | Cancel |

### Request Log tab

| Key | Action |
|-----|--------|
| `c` | Clear log |

### Global

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Switch tabs |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

---

## Capture mode

Press `i` to toggle capture mode. While active:

- Requests to **unmapped routes** are intercepted — the TUI opens a pre-filled form where you define the response. The mock is saved to disk and the HTTP request is answered immediately.
- Requests to **already-mapped routes** show the mock picker with an extra **"+ Define new mock"** option. Selecting it opens the same capture form, creating a new variant alongside the existing mock.

---

## License

MIT
