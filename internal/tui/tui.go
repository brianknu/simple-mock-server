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
	tabCount    // number of tabs in the tab bar
	tabMockForm // not in the tab bar; shown as overlay when creating/editing
)

var tabNames = []string{"Mocks", "Request Log"}

type pendingRequestMsg server.PendingRequest

type Model struct {
	srv       *server.Server
	activeTab int
	width     int
	height    int

	mockList   mockListModel
	reqLog     requestLogModel
	mockForm   mockFormModel
	mockPicker mockPickerModel

	showHelp    bool
	captureMode bool
	pendingReq  *server.PendingRequest
}

func NewModel(srv *server.Server) Model {
	return Model{
		srv:        srv,
		activeTab:  tabMockList,
		mockList:   newMockListModel(srv),
		reqLog:     newRequestLogModel(srv),
		mockForm:   newMockFormModel(srv),
		mockPicker: newMockPickerModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.reqLog.waitForEntry(),
		m.waitForSelection(),
	)
}

func (m Model) waitForSelection() tea.Cmd {
	return func() tea.Msg {
		req, ok := <-m.srv.SelectionCh
		if !ok {
			return nil
		}
		return selectionRequestMsg(req)
	}
}

func (m Model) waitForPending() tea.Cmd {
	return func() tea.Msg {
		pr, ok := <-m.srv.PendingCh
		if !ok {
			return nil
		}
		return pendingRequestMsg(pr)
	}
}

// inputActive returns true when a text input has focus or a detail overlay is
// open — single-char keys should not be treated as global commands in either case.
func (m Model) inputActive() bool {
	if m.activeTab == tabMockForm && m.mockForm.editing && m.mockForm.focusedField != fieldVerb {
		return true
	}
	if m.activeTab == tabMockList && m.mockList.detailActive() {
		return true
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// When mock picker is active, delegate everything to it except
	// selectionDoneMsg which is handled below (arrives after active=false).
	if m.mockPicker.active {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.mockPicker.width = msg.Width
			m.mockPicker.height = msg.Height - 4
			return m, nil
		case newLogEntryMsg:
			var cmd tea.Cmd
			m.reqLog, cmd = m.reqLog.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.mockPicker, cmd = m.mockPicker.Update(msg)
			return m, cmd
		}
	}

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

	case selectionRequestMsg:
		req := server.SelectionRequest(msg)
		m.mockPicker.activate(req, m.width, m.height-4)
		return m, nil

	case selectionDoneMsg:
		// Re-arm the selection listener. This always arrives after mockPicker.active
		// has already been set to false, so it must be handled here, not in the
		// picker-active block above.
		return m, m.waitForSelection()

	case tea.KeyMsg:
		// Esc always goes back to mock list from form tabs
		if key.Matches(msg, keys.Back) {
			if m.activeTab == tabMockForm {
				m.mockForm.editing = false
				m.activeTab = tabMockList
				// Cancel pending request if any
				if m.pendingReq != nil {
					m.pendingReq.ResponseCh <- server.PendingResponse{Cancel: true}
					m.pendingReq = nil
					m.srv.NextPending()
				}
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
			if key.Matches(msg, keys.Capture) && m.activeTab != tabMockForm {
				m.captureMode = !m.captureMode
				m.srv.SetCaptureMode(m.captureMode)
				if m.captureMode {
					return m, m.waitForPending()
				}
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

	case pendingRequestMsg:
		pr := server.PendingRequest(msg)
		m.pendingReq = &pr

		form := newMockFormModel(m.srv)
		form.width = m.width
		form.isPendingResponse = true
		form.pendingInfo = fmt.Sprintf("%s %s", pr.Method, pr.Path)
		// Set verb from request
		for i, v := range verbs {
			if v == pr.Method {
				form.verbIndex = i
				break
			}
		}
		form.pathsInput.SetValue(pr.Path)
		form.startEditing()
		m.mockForm = form
		m.activeTab = tabMockForm
		return m, m.mockForm.Init()

	case mockSavedMsg:
		m.mockList.mocks = m.srv.Mocks()
		m.mockList.message = string(msg)
		m.mockForm.editing = false
		m.activeTab = tabMockList

		if m.pendingReq != nil {
			resp := m.mockForm.buildResponse()
			m.pendingReq.ResponseCh <- resp
			m.pendingReq = nil
			m.srv.NextPending()
			return m, m.waitForPending()
		}
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

	// Mock picker overlay takes over the full screen
	if m.mockPicker.active {
		var b strings.Builder
		b.WriteString(m.tabBar())
		b.WriteString("\n\n")
		b.WriteString(m.mockPicker.View())
		b.WriteString("\n")
		status := fmt.Sprintf(" :%d | %d/%d mocks active | SELECTING MOCK | ? help", m.srv.Port, m.srv.MockCount(), m.srv.TotalMockCount())
		b.WriteString(statusBarStyle.Width(m.width).Render(status))
		return b.String()
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
	indicator := ""
	if m.captureMode {
		indicator = " | " + captureIndicatorStyle.Render("CAPTURE")
		if m.pendingReq != nil {
			indicator += fmt.Sprintf(" | awaiting: %s %s", m.pendingReq.Method, m.pendingReq.Path)
		}
	}
	status := fmt.Sprintf(" :%d | %d/%d mocks active%s | ? help", m.srv.Port, m.srv.MockCount(), m.srv.TotalMockCount(), indicator)
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
    d                   Show mock details
    space               Enable/disable mock
    x                   Delete selected mock
    r                   Reload mocks from disk

  Request Log Tab
    c                   Clear log

  Create/Edit Form (opens over Mocks tab)
    ↑/↓                 Cycle between fields
    ←/→                 Cycle HTTP verb (when on Method)
    ctrl+s              Save mock
    esc                 Cancel and go back

  Mock Selection
    When multiple mocks match the same path+verb,
    an interactive picker appears on each request.
    ↑/↓                 Navigate options
    enter                Select mock
    esc                  Cancel (returns 404)

  General
    i                   Toggle capture mode
    q / ctrl+c          Quit
`
	return titleStyle.Render("Help") + help + "\n\n" + helpStyle.Render("  Press ? to close")
}
