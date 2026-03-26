package tui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/server"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	fieldVerb = iota
	fieldPaths
	fieldStatus
	fieldResponseTime
	fieldHeaders
	fieldBody
	fieldCount
)

var verbs = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

type mockFormModel struct {
	srv       *server.Server
	editing   bool
	editIndex int // -1 for new mock

	verbIndex    int
	pathsInput   textinput.Model
	statusInput  textinput.Model
	rtimeInput   textinput.Model
	headersArea  textarea.Model
	bodyArea     textarea.Model

	focusedField int
	width        int
	message      string
	sourceFile   string

	isPendingResponse bool
	pendingInfo       string
}

type mockSavedMsg string

func newMockFormModel(srv *server.Server) mockFormModel {
	m := mockFormModel{
		srv:       srv,
		editIndex: -1,
	}
	m.pathsInput = textinput.New()
	m.pathsInput.Placeholder = "/api/endpoint, /api/other"
	m.pathsInput.CharLimit = 256

	m.statusInput = textinput.New()
	m.statusInput.Placeholder = "200"
	m.statusInput.CharLimit = 3

	m.rtimeInput = textinput.New()
	m.rtimeInput.Placeholder = "0"
	m.rtimeInput.CharLimit = 6

	m.headersArea = textarea.New()
	m.headersArea.Placeholder = "X-Custom: value"
	m.headersArea.SetValue("Content-Type: application/json")
	m.headersArea.SetHeight(3)

	m.bodyArea = textarea.New()
	m.bodyArea.Placeholder = `{"key": "value"}`
	m.bodyArea.SetHeight(6)

	return m
}

func newMockFormModelFromMock(srv *server.Server, mk mock.Mock, index int) mockFormModel {
	m := newMockFormModel(srv)
	m.editIndex = index
	m.sourceFile = mk.SourceFile

	for i, v := range verbs {
		if v == mk.Verb {
			m.verbIndex = i
			break
		}
	}

	m.pathsInput.SetValue(strings.Join(mk.Paths, ", "))
	m.statusInput.SetValue(strconv.Itoa(mk.Status))
	m.rtimeInput.SetValue(strconv.Itoa(mk.ResponseTime))

	var headerLines []string
	for k, v := range mk.Headers {
		headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, v))
	}
	m.headersArea.SetValue(strings.Join(headerLines, "\n"))

	if mk.Body != nil {
		bodyBytes, err := json.MarshalIndent(mk.Body, "", "  ")
		if err == nil {
			m.bodyArea.SetValue(string(bodyBytes))
		}
	}

	return m
}

func (m *mockFormModel) startEditing() {
	m.editing = true
	m.focusedField = fieldPaths
	m.pathsInput.Focus()
}

func (m mockFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m mockFormModel) Update(msg tea.Msg) (mockFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Enter starts editing a new mock if not already editing
		if msg.String() == "enter" && !m.editing {
			m.startEditing()
			return m, textinput.Blink
		}

		if !m.editing {
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Save):
			return m, m.save()

		case key.Matches(msg, keys.Down):
			m.advanceFocus()
			return m, nil

		case key.Matches(msg, keys.Up):
			m.retreatFocus()
			return m, nil

		case (msg.String() == "left" || msg.String() == "right") && m.focusedField == fieldVerb:
			if msg.String() == "right" {
				m.verbIndex = (m.verbIndex + 1) % len(verbs)
			} else {
				m.verbIndex = (m.verbIndex - 1 + len(verbs)) % len(verbs)
			}
			return m, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldPaths:
		m.pathsInput, cmd = m.pathsInput.Update(msg)
	case fieldStatus:
		m.statusInput, cmd = m.statusInput.Update(msg)
	case fieldResponseTime:
		m.rtimeInput, cmd = m.rtimeInput.Update(msg)
	case fieldHeaders:
		m.headersArea, cmd = m.headersArea.Update(msg)
	case fieldBody:
		m.bodyArea, cmd = m.bodyArea.Update(msg)
	}

	return m, cmd
}

func (m *mockFormModel) advanceFocus() {
	m.blurAll()
	m.focusedField = (m.focusedField + 1) % fieldCount
	m.focusCurrent()
}

func (m *mockFormModel) retreatFocus() {
	m.blurAll()
	m.focusedField = (m.focusedField - 1 + fieldCount) % fieldCount
	m.focusCurrent()
}

func (m *mockFormModel) blurAll() {
	m.pathsInput.Blur()
	m.statusInput.Blur()
	m.rtimeInput.Blur()
	m.headersArea.Blur()
	m.bodyArea.Blur()
}

func (m *mockFormModel) focusCurrent() {
	switch m.focusedField {
	case fieldPaths:
		m.pathsInput.Focus()
	case fieldStatus:
		m.statusInput.Focus()
	case fieldResponseTime:
		m.rtimeInput.Focus()
	case fieldHeaders:
		m.headersArea.Focus()
	case fieldBody:
		m.bodyArea.Focus()
	}
}

func (m mockFormModel) save() tea.Cmd {
	return func() tea.Msg {
		// Parse paths
		pathStr := m.pathsInput.Value()
		if pathStr == "" {
			return mockSavedMsg("Error: paths required")
		}
		var paths []string
		for _, p := range strings.Split(pathStr, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}

		// Parse status
		status, err := strconv.Atoi(m.statusInput.Value())
		if err != nil {
			status = 200
		}

		// Parse response time
		rtime, _ := strconv.Atoi(m.rtimeInput.Value())

		// Parse headers
		headers := make(map[string]string)
		for _, line := range strings.Split(m.headersArea.Value(), "\n") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Parse body
		var body any
		bodyStr := strings.TrimSpace(m.bodyArea.Value())
		if bodyStr != "" {
			if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
				return mockSavedMsg(fmt.Sprintf("Error: invalid JSON body: %s", err))
			}
		}

		newMock := mock.Mock{
			Paths:        paths,
			Verb:         verbs[m.verbIndex],
			Body:         body,
			Headers:      headers,
			Status:       status,
			ResponseTime: rtime,
			SourceFile:   m.sourceFile,
		}

		// Save to disk
		filename, err := mock.SaveMockToFS(m.srv.MocksDir, newMock)
		if err != nil {
			return mockSavedMsg(fmt.Sprintf("Error saving: %s", err))
		}
		newMock.SourceFile = filename

		// Update server state
		if m.editIndex >= 0 {
			m.srv.UpdateMock(m.editIndex, newMock)
		} else {
			m.srv.AddMock(newMock)
		}

		action := "Created"
		if m.editIndex >= 0 {
			action = "Updated"
		}
		return mockSavedMsg(fmt.Sprintf("%s mock → %s", action, filename))
	}
}

func (m mockFormModel) buildResponse() server.PendingResponse {
	status, _ := strconv.Atoi(m.statusInput.Value())
	if status == 0 {
		status = 200
	}

	headers := make(map[string]string)
	for _, line := range strings.Split(m.headersArea.Value(), "\n") {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	bodyStr := strings.TrimSpace(m.bodyArea.Value())
	var body []byte
	if bodyStr != "" {
		body = []byte(bodyStr)
	}

	return server.PendingResponse{
		Status:  status,
		Headers: headers,
		Body:    body,
	}
}

func (m mockFormModel) View() string {
	if !m.editing {
		return "  Press 'n' on the Mocks tab to create a new mock, or 'e' to edit one.\n  You can also press Enter here to start a new mock."
	}

	var b strings.Builder

	if m.isPendingResponse {
		b.WriteString(liveRequestBannerStyle.Render("LIVE REQUEST: " + m.pendingInfo + " is waiting for your response"))
		b.WriteString("\n\n")
	}

	title := "Create New Mock"
	if m.editIndex >= 0 {
		title = "Edit Mock"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Verb selector
	focus := " "
	if m.focusedField == fieldVerb {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Method: ◀ %s ▶\n", focus, methodStyle(verbs[m.verbIndex]).Render(verbs[m.verbIndex])))

	// Paths
	focus = " "
	if m.focusedField == fieldPaths {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Paths:  %s\n", focus, m.pathsInput.View()))

	// Status
	focus = " "
	if m.focusedField == fieldStatus {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Status: %s\n", focus, m.statusInput.View()))

	// Response time
	focus = " "
	if m.focusedField == fieldResponseTime {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Delay:  %s\n", focus, m.rtimeInput.View()))

	// Headers
	focus = " "
	if m.focusedField == fieldHeaders {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Headers (Key: Value per line):\n", focus))
	b.WriteString("    " + m.headersArea.View())
	b.WriteString("\n")

	// Body
	focus = " "
	if m.focusedField == fieldBody {
		focus = "▸"
	}
	b.WriteString(fmt.Sprintf("  %s Body (JSON):\n", focus))
	b.WriteString("    " + m.bodyArea.View())
	b.WriteString("\n")

	if m.message != "" {
		b.WriteString("\n  " + m.message)
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑/↓: cycle fields  ←/→: cycle verb  ctrl+s: save  esc: cancel"))

	return b.String()
}
