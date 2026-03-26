package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/server"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mockListModel struct {
	srv        *server.Server
	mocks      []mock.Mock
	cursor     int
	width      int
	height     int
	message    string
	showDetail bool
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
		if m.showDetail {
			// Any key closes the detail view
			m.showDetail = false
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.mocks)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Detail):
			if len(m.mocks) > 0 {
				m.showDetail = true
			}
		case key.Matches(msg, keys.Toggle):
			if len(m.mocks) > 0 {
				m.srv.ToggleMock(m.cursor)
				m.mocks = m.srv.Mocks()
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
	if m.showDetail && len(m.mocks) > 0 {
		return m.detailView(m.mocks[m.cursor])
	}
	return m.listView()
}

func (m mockListModel) listView() string {
	var b strings.Builder

	header := fmt.Sprintf("  %-3s %-8s %-40s %-8s %s", "", "METHOD", "PATHS", "STATUS", "DELAY(ms)")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8F2")).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 80)))
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

		// Truncate paths before padding so column width is stable
		pathStr := strings.Join(mk.Paths, ", ")
		if len(pathStr) > 38 {
			pathStr = pathStr[:35] + "..."
		}

		// Pad each column to its fixed width BEFORE styling.
		// fmt.Sprintf with %-Ns counts bytes, not visual width, so styled strings
		// must never be passed to %-Ns — only plain strings get padded here.
		dotRaw := "●"
		dotColor := lipgloss.Color("#50FA7B")
		if mk.Disabled {
			dotRaw = "○"
			dotColor = lipgloss.Color("#44475A")
		}
		dot := lipgloss.NewStyle().Foreground(dotColor).Render(dotRaw)

		verbPadded := fmt.Sprintf("%-8s", mk.Verb)
		pathPadded := fmt.Sprintf("%-40s", pathStr)
		statusPadded := fmt.Sprintf("%-8d", mk.Status)
		delayPadded := fmt.Sprintf("%-12d", mk.ResponseTime)

		var verb, paths, status string
		if mk.Disabled {
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
			verb = dim.Render(verbPadded)
			paths = dim.Render(pathPadded)
			status = dim.Render(statusPadded)
		} else {
			verb = methodStyle(mk.Verb).Render(verbPadded)
			paths = pathPadded
			status = statusStyle(mk.Status).Render(statusPadded)
		}

		line := cursor + dot + " " + verb + " " + paths + " " + status + " " + delayPadded

		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("#44475A")).Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n  " + m.message)
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  n: new  e: edit  d: detail  space: toggle  x: delete  r: reload  i: capture  ↑/↓: navigate"))

	return b.String()
}

func (m mockListModel) detailView(mk mock.Mock) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Mock Detail"))
	b.WriteString("\n\n")

	row := func(label, value string) string {
		l := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Bold(true).Render(fmt.Sprintf("  %-14s", label))
		return l + " " + value
	}

	stateStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true).Render("● enabled")
	if mk.Disabled {
		stateStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("○ disabled")
	}
	b.WriteString(row("State", stateStr))
	b.WriteString("\n")

	verb := methodStyle(mk.Verb).Render(mk.Verb)
	b.WriteString(row("Method", verb))
	b.WriteString("\n")

	b.WriteString(row("Paths", lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(strings.Join(mk.Paths, ", "))))
	b.WriteString("\n")

	b.WriteString(row("Status", statusStyle(mk.Status).Render(fmt.Sprintf("%d", mk.Status))))
	b.WriteString("\n")

	b.WriteString(row("Delay", lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(fmt.Sprintf("%d ms", mk.ResponseTime))))
	b.WriteString("\n")

	b.WriteString(row("File", lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Render(mk.SourceFile)))
	b.WriteString("\n")

	if len(mk.Headers) > 0 {
		b.WriteString(row("Headers", ""))
		b.WriteString("\n")
		for k, v := range mk.Headers {
			b.WriteString(fmt.Sprintf("    %s: %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Render(k),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(v),
			))
		}
	}

	if mk.Body != nil {
		bodyBytes, err := json.MarshalIndent(mk.Body, "    ", "  ")
		if err == nil {
			b.WriteString(row("Body", ""))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("    " + string(bodyBytes)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  any key to close"))

	return b.String()
}

type mocksUpdatedMsg struct{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// detailActive reports whether the detail overlay is open.
// Used by tui.go to suppress global keybindings.
func (m mockListModel) detailActive() bool {
	return m.showDetail
}

// selectedMock returns the currently selected mock.
func (m mockListModel) selectedMock() *mock.Mock {
	if len(m.mocks) == 0 {
		return nil
	}
	mk := m.mocks[m.cursor]
	return &mk
}
