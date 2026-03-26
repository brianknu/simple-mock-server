package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"simple-mock-server/internal/server"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type requestLogModel struct {
	srv        *server.Server
	entries    []server.LogEntry
	sub        chan server.LogEntry
	cursor     int
	scroll     int
	width      int
	height     int
	showDetail bool
	copyMsg    string // transient feedback after copying
}

func newRequestLogModel(srv *server.Server) requestLogModel {
	return requestLogModel{
		srv:     srv,
		entries: srv.ReqLog.Entries(),
		sub:     srv.ReqLog.Subscribe(),
	}
}

type newLogEntryMsg server.LogEntry

func (m requestLogModel) waitForEntry() tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-m.sub
		if !ok {
			return nil
		}
		return newLogEntryMsg(entry)
	}
}

// entryAt returns the entry for visual index i (0 = newest).
func (m requestLogModel) entryAt(i int) server.LogEntry {
	return m.entries[len(m.entries)-1-i]
}

func (m requestLogModel) Update(msg tea.Msg) (requestLogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case newLogEntryMsg:
		m.entries = append(m.entries, server.LogEntry(msg))
		// New entries appear at the top; keep cursor at 0 so the user
		// always sees the latest request without manual scrolling.
		m.cursor = 0
		m.scroll = 0
		return m, m.waitForEntry()

	case tea.KeyMsg:
		if m.showDetail {
			switch {
			case key.Matches(msg, keys.Back):
				m.showDetail = false
				m.copyMsg = ""
			case key.Matches(msg, keys.Copy):
				if len(m.entries) > 0 {
					body := m.entryAt(m.cursor).RequestBody
					if body == "" {
						m.copyMsg = "nothing to copy — no request body was logged"
					} else if err := clipboard.WriteAll(body); err != nil {
						m.copyMsg = "copy failed: " + err.Error()
					} else {
						m.copyMsg = "copied to clipboard!"
					}
				}
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Clear):
			m.srv.ReqLog.Clear()
			m.entries = nil
			m.cursor = 0
			m.scroll = 0
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.clampScroll()
			}
		case msg.String() == "enter":
			if len(m.entries) > 0 {
				m.showDetail = true
				m.copyMsg = ""
			}
		}
	}
	return m, nil
}

// clampScroll keeps the cursor visible in the viewport.
func (m *requestLogModel) clampScroll() {
	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 10
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+maxVisible {
		m.scroll = m.cursor - maxVisible + 1
	}
}

func (m requestLogModel) View() string {
	if m.showDetail && len(m.entries) > 0 {
		return m.detailView(m.entryAt(m.cursor))
	}
	return m.listView()
}

func (m requestLogModel) listView() string {
	var b strings.Builder

	// Mirror the data row layout: "  "(2) + cur(2) + ts(12) + " " + verb(8) + ...
	header := "  " + "  " + fmt.Sprintf("%-12s", "TIME") + " " + fmt.Sprintf("%-8s", "METHOD") + " " + fmt.Sprintf("%-35s", "PATH") + " " + fmt.Sprintf("%-8s", "STATUS") + " " + "DELAY(ms)"
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2")).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 90)))
	b.WriteString("\n")

	if len(m.entries) == 0 {
		b.WriteString("\n  No requests yet. Send a request to see it here.\n")
	}

	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 10
	}

	end := m.scroll + maxVisible
	if end > len(m.entries) {
		end = len(m.entries)
	}

	for i := m.scroll; i < end; i++ {
		e := m.entryAt(i)
		cur := "  "
		if i == m.cursor {
			cur = "▸ "
		}

		ts := e.Timestamp.Format("15:04:05.000")
		verbPadded := fmt.Sprintf("%-8s", e.Method)
		verb := methodStyle(e.Method).Render(verbPadded)
		pathPadded := fmt.Sprintf("%-35s", truncate(e.Path, 35))
		statusPadded := fmt.Sprintf("%-8d", e.Status)
		status := statusStyle(e.Status).Render(statusPadded)
		delayPadded := fmt.Sprintf("%-10d", e.ResponseTime)

		line := fmt.Sprintf("  %s%-12s ", cur, ts) + verb + " " + pathPadded + " " + status + " " + delayPadded

		if e.RequestBody != "" {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(" ⬡ body")
		}

		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("#44475A")).Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("  %d requests | enter: inspect  c: clear  ↑/↓: navigate", len(m.entries))))

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m requestLogModel) detailView(e server.LogEntry) string {
	var b strings.Builder

	banner := fmt.Sprintf("REQUEST DETAIL: %s %s", e.Method, e.Path)
	b.WriteString(liveRequestBannerStyle.Render(banner))
	b.WriteString("\n\n")

	row := func(label, value string) string {
		l := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Bold(true).Render(fmt.Sprintf("  %-14s", label))
		return l + " " + value
	}

	b.WriteString(row("Time", lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(e.Timestamp.Format("2006-01-02 15:04:05.000"))))
	b.WriteString("\n")
	b.WriteString(row("Method", methodStyle(e.Method).Render(e.Method)))
	b.WriteString("\n")
	b.WriteString(row("Path", lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(e.Path)))
	b.WriteString("\n")
	b.WriteString(row("Status", statusStyle(e.Status).Render(fmt.Sprintf("%d", e.Status))))
	b.WriteString("\n")
	b.WriteString(row("Delay", lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(fmt.Sprintf("%d ms", e.ResponseTime))))
	b.WriteString("\n")

	b.WriteString("\n")
	bodyLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Bold(true).Render("  Request Body")
	b.WriteString(bodyLabel)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 60)))
	b.WriteString("\n")

	if e.RequestBody == "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Italic(true).Render("  (no body)"))
	} else {
		body := e.RequestBody
		// Pretty-print if valid JSON
		var js any
		if err := json.Unmarshal([]byte(body), &js); err == nil {
			if pretty, err := json.MarshalIndent(js, "", "  "); err == nil {
				body = string(pretty)
			}
		}
		// Render each line individually — lipgloss misaligns multiline strings
		// when passed as a single Render call.
		lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
		for _, line := range strings.Split(body, "\n") {
			b.WriteString(lineStyle.Render("  " + line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")

	hint := "  esc: back"
	if e.RequestBody != "" {
		if m.copyMsg != "" {
			hint += "  |  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true).Render(m.copyMsg)
		} else {
			hint += "  |  " + helpStyle.Render("y: copy body to clipboard")
		}
	}
	b.WriteString(helpStyle.Render(hint))

	return b.String()
}
