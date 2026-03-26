package tui

import (
	"fmt"
	"strings"

	"simple-mock-server/internal/server"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabMockList = iota
	tabRequestLog
	tabMockForm
	tabCount
)

var tabNames = []string{"Mocks", "Request Log", "Create/Edit"}

type Model struct {
	srv       *server.Server
	activeTab int
	width     int
	height    int

	mockList mockListModel
	reqLog   requestLogModel
	mockForm mockFormModel

	showHelp bool
}

func NewModel(srv *server.Server) Model {
	return Model{
		srv:       srv,
		activeTab: tabMockList,
		mockList:  newMockListModel(srv),
		reqLog:    newRequestLogModel(srv),
		mockForm:  newMockFormModel(srv),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.reqLog.waitForEntry(),
	)
}

// inputActive returns true when a text input has focus and single-char keys
// (q, ?, n, etc.) should be forwarded to the input instead of treated as commands.
func (m Model) inputActive() bool {
	if m.activeTab == tabMockForm && m.mockForm.editing && m.mockForm.focusedField != fieldVerb {
		return true
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mockList.width = msg.Width
		m.mockList.height = msg.Height - 4
		m.reqLog.width = msg.Width
		m.reqLog.height = msg.Height - 4
		m.mockForm.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Esc always goes back to mock list from form tabs
		if key.Matches(msg, keys.Back) {
			if m.activeTab == tabMockForm {
				m.mockForm.editing = false
				m.activeTab = tabMockList
				return m, nil
			}
		}

		// Global keys only when no text input is active
		if !m.inputActive() {
			if key.Matches(msg, keys.Quit) {
				return m, tea.Quit
			}
			if key.Matches(msg, keys.Help) {
				m.showHelp = !m.showHelp
				return m, nil
			}
			if key.Matches(msg, keys.Tab) {
				m.activeTab = (m.activeTab + 1) % tabCount
				m.onTabSwitch()
				return m, nil
			}
			if key.Matches(msg, keys.ShiftTab) {
				m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
				m.onTabSwitch()
				return m, nil
			}
		}

		// Tab-specific: new/edit mock triggers form switch
		if m.activeTab == tabMockList {
			if key.Matches(msg, keys.New) {
				m.mockForm = newMockFormModel(m.srv)
				m.mockForm.width = m.width
				m.mockForm.startEditing()
				m.activeTab = tabMockForm
				return m, m.mockForm.Init()
			}
			if key.Matches(msg, keys.Edit) && len(m.mockList.mocks) > 0 {
				m.mockForm = newMockFormModelFromMock(m.srv, m.mockList.mocks[m.mockList.cursor], m.mockList.cursor)
				m.mockForm.width = m.width
				m.mockForm.startEditing()
				m.activeTab = tabMockForm
				return m, m.mockForm.Init()
			}
		}

	case newLogEntryMsg:
		var cmd tea.Cmd
		m.reqLog, cmd = m.reqLog.Update(msg)
		return m, cmd

	case mockSavedMsg:
		m.mockList.mocks = m.srv.Mocks()
		m.mockList.message = string(msg)
		m.mockForm.editing = false
		m.activeTab = tabMockList
		return m, nil
	}

	// Delegate to active tab
	switch m.activeTab {
	case tabMockList:
		var cmd tea.Cmd
		m.mockList, cmd = m.mockList.Update(msg)
		cmds = append(cmds, cmd)
	case tabRequestLog:
		var cmd tea.Cmd
		m.reqLog, cmd = m.reqLog.Update(msg)
		cmds = append(cmds, cmd)
	case tabMockForm:
		var cmd tea.Cmd
		m.mockForm, cmd = m.mockForm.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// onTabSwitch resets focus state when switching tabs.
// Tabs start in "browse" mode — press Enter to engage inputs.
func (m *Model) onTabSwitch() {
	m.mockForm.editing = false
	m.mockForm.blurAll()
}

func (m Model) View() string {
	if m.showHelp {
		return m.helpView()
	}

	var b strings.Builder

	// Tab bar
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")

	// Active tab content
	switch m.activeTab {
	case tabMockList:
		b.WriteString(m.mockList.View())
	case tabRequestLog:
		b.WriteString(m.reqLog.View())
	case tabMockForm:
		b.WriteString(m.mockForm.View())
	}

	// Status bar
	b.WriteString("\n")
	status := fmt.Sprintf(" :%d | %d mocks loaded | ? help", m.srv.Port, m.srv.MockCount())
	b.WriteString(statusBarStyle.Width(m.width).Render(status))

	return b.String()
}

func (m Model) tabBar() string {
	var tabs []string
	for i, name := range tabNames {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) helpView() string {
	help := `
  Simple Mock Server - Keyboard Shortcuts

  Navigation
    tab / shift+tab    Switch between tabs
    ↑/↓                Navigate lists
    esc                 Back to Mocks tab
    ?                   Toggle this help

  Mocks Tab
    n                   Create new mock
    e                   Edit selected mock
    d                   Delete selected mock
    r                   Reload mocks from disk

  Request Log Tab
    c                   Clear log

  Create/Edit Tab
    ↑/↓                 Cycle between fields
    ←/→                 Cycle HTTP verb (when on Method)
    ctrl+s              Save mock
    esc                 Cancel and go back

  General
    q / ctrl+c          Quit
`
	return titleStyle.Render("Help") + help + "\n\n" + helpStyle.Render("  Press ? to close")
}
