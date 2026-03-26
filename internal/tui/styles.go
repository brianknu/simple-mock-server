package tui

import "github.com/charmbracelet/lipgloss"

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#FF79C6")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4")).
				Padding(0, 2)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#282A36")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1).
			Width(80)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#BD93F9")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	methodGetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)

	methodPostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true)

	methodPutStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD")).
			Bold(true)

	methodDeleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5555")).
				Bold(true)

	methodPatchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F1FA8C")).
				Bold(true)

	methodDefaultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BD93F9")).
				Bold(true)

	captureIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF79C6")).
				Bold(true)

	liveRequestBannerStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF79C6")).
				Foreground(lipgloss.Color("#282A36")).
				Bold(true).
				Padding(0, 1)
)

func statusStyle(code int) lipgloss.Style {
	switch {
	case code >= 500:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	case code >= 400:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)
	case code >= 300:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")).Bold(true)
	case code >= 200:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	}
}

func methodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return methodGetStyle
	case "POST":
		return methodPostStyle
	case "PUT":
		return methodPutStyle
	case "DELETE":
		return methodDeleteStyle
	case "PATCH":
		return methodPatchStyle
	default:
		return methodDefaultStyle
	}
}
