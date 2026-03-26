package tui

import (
	"fmt"
	"strings"

	"simple-mock-server/internal/server"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type requestLogModel struct {
	srv     *server.Server
	entries []server.LogEntry
	sub     chan server.LogEntry
	scroll  int
	width   int
	height  int
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

func (m requestLogModel) Update(msg tea.Msg) (requestLogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case newLogEntryMsg:
		m.entries = append(m.entries, server.LogEntry(msg))
		// Auto-scroll to bottom
		maxVisible := m.height - 4
		if maxVisible < 1 {
			maxVisible = 10
		}
		if len(m.entries) > maxVisible {
			m.scroll = len(m.entries) - maxVisible
		}
		return m, m.waitForEntry()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Clear):
			m.srv.ReqLog.Clear()
			m.entries = nil
			m.scroll = 0
		case key.Matches(msg, keys.Up):
			if m.scroll > 0 {
				m.scroll--
			}
		case key.Matches(msg, keys.Down):
			maxVisible := m.height - 4
			if maxVisible < 1 {
				maxVisible = 10
			}
			if m.scroll < len(m.entries)-maxVisible {
				m.scroll++
			}
		}
	}
	return m, nil
}

func (m requestLogModel) View() string {
	var b strings.Builder

	header := fmt.Sprintf("  %-12s %-8s %-35s %-8s %-10s", "TIME", "METHOD", "PATH", "STATUS", "DELAY(ms)")
	b.WriteString(helpStyle.Render(header))
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
		e := m.entries[i]
		ts := e.Timestamp.Format("15:04:05.000")
		verb := methodStyle(e.Method).Render(fmt.Sprintf("%-8s", e.Method))
		line := fmt.Sprintf("  %-12s %s %-35s %-8d %-10d", ts, verb, e.Path, e.Status, e.ResponseTime)
		b.WriteString(line)
		if e.RequestBody != "" {
			b.WriteString("\n    Body: " + truncate(e.RequestBody, 60))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("  %d requests | c: clear  ↑/↓: scroll", len(m.entries))))

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
