# ADR 001: Capture Mode — Interactive Mock Creation from Live Requests

## Status

Accepted

## Date

2026-03-25

## Context

The simple-mock-server serves predefined mock responses from JSON files. When a request arrives that doesn't match any mock, the server returns a 404. Users building mocks iteratively — for example, while developing a frontend against a not-yet-implemented backend — must manually create each mock file or use the TUI form, which requires knowing the exact endpoints in advance.

We wanted a "learning mode" where the server captures unmatched requests and lets the user define mock responses on the fly, directly from the TUI, so that the mock library grows organically as the application under development makes requests.

## Decision

We introduce **Capture Mode**, toggled with the `i` key in the TUI. When enabled, unmatched HTTP requests no longer receive a 404. Instead:

1. The HTTP handler captures the request (method, path, headers, body) and **blocks**, waiting for the user to define a response.
2. The TUI receives a notification, switches to the Create/Edit form pre-filled with the request's method and path, and shows a "LIVE REQUEST" banner.
3. The user fills in the response (status code, headers, body) and saves with `ctrl+s`.
4. The mock is saved to disk as a `.json` file and registered on the mux.
5. The blocked HTTP handler receives the user-defined response and sends it to the original caller.
6. The server continues listening for the next unmatched request.

### Channel-based synchronization

The core design challenge is coordinating the HTTP handler goroutine (which must wait) with the TUI running on the main goroutine. We use a **channel pair**:

```
HTTP handler goroutine                          TUI (main goroutine)
──────────────────────                          ────────────────────
Creates PendingRequest with ResponseCh(cap 1)
  → sends to server.PendingCh ──────────────→  waitForPending() reads from PendingCh
  blocks on <-ResponseCh                        Shows pre-filled form
                                                User saves mock
                                  ←──────────── Sends PendingResponse to ResponseCh
  Writes HTTP response
```

- `PendingRequest.ResponseCh` is a buffered channel (capacity 1) so the TUI's send never blocks even if the HTTP client disconnected.
- `server.PendingCh` is buffered (capacity 1). Overflow requests go to `pendingQueue` and are delivered one at a time after each completion.

### Pluggable fallback handler on DynamicMux

Rather than coupling the router to capture mode, we added a `NotFoundHandler http.HandlerFunc` field to `DynamicMux`. When set, it is called instead of the default `http.NotFound`. This keeps the router generic and testable.

### Request serialization

Only one unmatched request is presented to the user at a time. Additional requests queue in `pendingQueue` and their HTTP handler goroutines block on their individual `ResponseCh` channels. After the user handles one, `NextPending()` delivers the next. This avoids overwhelming the user and keeps the TUI interaction sequential.

## Consequences

### Positive

- Users can build a mock library incrementally by simply running their application against the mock server — no upfront endpoint knowledge required.
- The mock is saved to disk immediately, so it persists across server restarts.
- The original HTTP caller receives the user-defined response, enabling real-time development feedback.
- The `NotFoundHandler` pattern is reusable for other fallback behaviors in the future.

### Negative

- The HTTP caller blocks until the user responds (or the client times out). This is by design but means slow user interaction leads to client timeouts. The mock is still saved even if the client disconnects.
- Only available in TUI mode — headless mode does not support capture mode since there is no interactive UI.
- Serialized request handling means concurrent unmatched requests queue up. In practice this is acceptable since mock creation is a development-time activity.

### Risks

- If the user forgets capture mode is on, unmatched requests will hang instead of returning 404. The status bar indicator ("CAPTURE") mitigates this.
- Pressing `esc` on the form cancels the pending request (returns 404) and discards the mock. This is intentional — the user must save to create the mock.
