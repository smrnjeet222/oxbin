package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StateChangeMsg is sent when a component wants to change the app state
type StateChangeMsg struct {
	NewState string
}

type MenuModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func NewMenuModel() MenuModel {
	return MenuModel{
		choices: []string{
			"📤 Upload File",
			"📖 Read File",
			"ℹ️ About",
		},
		selected: make(map[int]struct{}),
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			switch m.cursor {
			case 0: // Upload File
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: "upload"}
				}
			case 1: // Read File
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: "read"}
				}
			case 2: // About
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: "about"}
				}
			}
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	return m.ViewWithSize(80, 24) // Default size
}

func (m MenuModel) ViewWithSize(width, height int) string {
	// Adjust container width based on screen size
	containerWidth := max(min(width-8, 70), 30)

	s := menuTitleStyle.Render("Welcome to OxBin") + "\n\n"

	// Responsive description
	desc := "A decentralized pastebin powered by Walrus Protocol"
	if width < 60 {
		desc = "Decentralized pastebin on Walrus"
	}
	s += menuDescStyle.Render(desc) + "\n\n"
	s += menuSubtitleStyle.Render("Choose an option:") + "\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = menuCursorStyle.Render("▶")
			choice = menuSelectedStyle.Render(choice)
		} else {
			choice = menuItemStyle.Render(choice)
		}

		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	// Responsive help text
	helpText := "Use ↑/↓ or j/k to navigate, Enter to select"
	if width < 50 {
		helpText = "↑/↓ to navigate, Enter to select"
	}
	s += "\n" + menuHelpStyle.Render(helpText)

	return menuContainerStyle.Width(containerWidth).Render(s)
}

// Menu styles
var (
	menuContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#aa96da")).
				Padding(1, 2).
				MarginLeft(2)

	menuTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#35155d")).
			Background(lipgloss.Color("#dcd6f7")).
			Bold(true).
			Padding(0, 1)

	menuDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a")).
			Italic(true)

	menuSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f4eeff")).
				Bold(true)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f4eeff")).
			PaddingLeft(1)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fcbad3")).
				Bold(true).
				PaddingLeft(1)

	menuCursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9b572"))

	menuHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a")).
			Italic(true)
)
