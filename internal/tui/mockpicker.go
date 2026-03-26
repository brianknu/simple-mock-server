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

type selectionRequestMsg server.SelectionRequest

type selectionDoneMsg struct {
	defineNew bool // true when user chose "+ Define new mock"
	path      string
	verb      string
}

type mockPickerModel struct {
	active      bool
	mocks       []mock.Mock
	path        string
	verb        string
	captureMode bool
	cursor      int
	width       int
	height      int

	responseCh chan int
}

func newMockPickerModel() mockPickerModel {
	return mockPickerModel{}
}

func (m *mockPickerModel) activate(req server.SelectionRequest, width, height int) {
	m.active = true
	m.mocks = req.Mocks
	m.path = req.Path
	m.verb = req.Verb
	m.captureMode = req.CaptureMode
	m.cursor = 0
	m.width = width
	m.height = height
	m.responseCh = req.ResponseCh
}

// totalItems returns the number of selectable rows (mocks + optional "define new").
func (m mockPickerModel) totalItems() int {
	if m.captureMode {
		return len(m.mocks) + 1
	}
	return len(m.mocks)
}

func (m mockPickerModel) Update(msg tea.Msg) (mockPickerModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < m.totalItems()-1 {
				m.cursor++
			}
		case msg.String() == "enter":
			isDefineNew := m.captureMode && m.cursor == len(m.mocks)
			if isDefineNew {
				m.responseCh <- -2
				m.active = false
				return m, func() tea.Msg {
					return selectionDoneMsg{defineNew: true, path: m.path, verb: m.verb}
				}
			}
			m.responseCh <- m.cursor
			m.active = false
			return m, func() tea.Msg { return selectionDoneMsg{} }
		case key.Matches(msg, keys.Back):
			m.responseCh <- -1
			m.active = false
			return m, func() tea.Msg { return selectionDoneMsg{} }
		}
	}

	return m, nil
}

func (m mockPickerModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	banner := fmt.Sprintf("SELECT MOCK: %s %s", m.verb, m.path)
	b.WriteString(liveRequestBannerStyle.Render(banner))
	b.WriteString("\n\n")

	header := fmt.Sprintf("  %-4s %-8s %-12s %-10s %s", "#", "STATUS", "DELAY(ms)", "FILE", "BODY")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 90)))
	b.WriteString("\n")

	for i, mk := range m.mocks {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		source := mk.SourceFile
		if len(source) > 20 {
			source = "..." + source[len(source)-17:]
		}

		bodyPreview := ""
		if mk.Body != nil {
			if bodyBytes, err := json.Marshal(mk.Body); err == nil {
				bodyPreview = string(bodyBytes)
				if len(bodyPreview) > 40 {
					bodyPreview = bodyPreview[:37] + "..."
				}
			}
		}

		line := fmt.Sprintf("%s%-4d %-8d %-12d %-10s %s", cursor, i+1, mk.Status, mk.ResponseTime, source, bodyPreview)

		if i == m.cursor {
			line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// "+ Define new mock" option in capture mode
	if m.captureMode {
		b.WriteString(strings.Repeat("─", min(m.width, 90)))
		b.WriteString("\n")
		defineIdx := len(m.mocks)
		cursor := "  "
		if m.cursor == defineIdx {
			cursor = "▸ "
		}
		line := cursor + captureIndicatorStyle.Render("+ Define new mock for this request")
		if m.cursor == defineIdx {
			line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑/↓: navigate  enter: select  esc: cancel (404)"))

	return b.String()
}
