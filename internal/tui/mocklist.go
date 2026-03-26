package tui

import (
	"fmt"
	"strings"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/server"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mockListModel struct {
	srv      *server.Server
	mocks    []mock.Mock
	cursor   int
	width    int
	height   int
	message  string
}

func newMockListModel(srv *server.Server) mockListModel {
	return mockListModel{
		srv:   srv,
		mocks: srv.Mocks(),
	}
}

func (m mockListModel) Update(msg tea.Msg) (mockListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.mocks)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Delete):
			if len(m.mocks) > 0 {
				m.srv.DeleteMock(m.cursor)
				m.mocks = m.srv.Mocks()
				if m.cursor >= len(m.mocks) && m.cursor > 0 {
					m.cursor--
				}
				m.message = "Mock deleted"
			}
		case key.Matches(msg, keys.Reload):
			if err := m.srv.ReloadMocks(); err != nil {
				m.message = fmt.Sprintf("Reload error: %s", err)
			} else {
				m.mocks = m.srv.Mocks()
				m.message = fmt.Sprintf("Reloaded %d mocks", len(m.mocks))
				if m.cursor >= len(m.mocks) && m.cursor > 0 {
					m.cursor = len(m.mocks) - 1
				}
			}
		}
	case mocksUpdatedMsg:
		m.mocks = m.srv.Mocks()
		m.message = ""
	}
	return m, nil
}

func (m mockListModel) View() string {
	var b strings.Builder

	header := fmt.Sprintf("  %-8s %-35s %-8s %-12s %s", "METHOD", "PATHS", "STATUS", "DELAY(ms)", "FILE")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 90)))
	b.WriteString("\n")

	if len(m.mocks) == 0 {
		b.WriteString("\n  No mocks loaded. Press 'n' to create one or 'r' to reload from disk.\n")
	}

	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 10
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.mocks) && i < start+maxVisible; i++ {
		mk := m.mocks[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		paths := strings.Join(mk.Paths, ", ")
		if len(paths) > 33 {
			paths = paths[:30] + "..."
		}

		source := mk.SourceFile
		if len(source) > 20 {
			source = "..." + source[len(source)-17:]
		}

		verb := methodStyle(mk.Verb).Render(fmt.Sprintf("%-8s", mk.Verb))
		line := fmt.Sprintf("%s%s %-35s %-8d %-12d %s", cursor, verb, paths, mk.Status, mk.ResponseTime, source)

		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n  " + m.message)
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  n: new  e: edit  d: delete  r: reload  ↑/↓: navigate"))

	return b.String()
}

type mocksUpdatedMsg struct{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
