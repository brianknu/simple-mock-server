package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Help     key.Binding
	New      key.Binding
	Edit     key.Binding
	Detail   key.Binding
	Delete   key.Binding
	Reload   key.Binding
	Clear    key.Binding
	Save     key.Binding
	Send     key.Binding
	Back     key.Binding
	Up       key.Binding
	Down     key.Binding
	Capture  key.Binding
	Toggle   key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new mock"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit mock"),
	),
	Detail: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "detail"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete mock"),
	),
	Reload: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reload mocks"),
	),
	Clear: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear log"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Send: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "send request"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
	Capture: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "toggle capture mode"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "enable/disable mock"),
	),
}
