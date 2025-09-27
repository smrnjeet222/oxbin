package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/smrnjeet222/oxbin/internal/cli/components"
	"github.com/smrnjeet222/oxbin/internal/core"
)

// Application states
type state int

const (
	menuState state = iota
	uploadState
	readState
	aboutState
)

// App represents the CLI application
type App struct {
	config  *core.Config
	service *core.Service
	state   state
	width   int
	height  int

	// Components
	menuModel   components.MenuModel
	uploadModel components.UploadModel
	readModel   components.ReadModel
	aboutModel  components.AboutModel
}

// NewApp creates a new CLI application
func NewApp(config *core.Config, service *core.Service) *App {
	return &App{
		config:      config,
		service:     service,
		state:       menuState,
		width:       config.CLI.DefaultWidth,
		height:      config.CLI.DefaultHeight,
		menuModel:   components.NewMenuModel(),
		uploadModel: components.NewUploadModel(service),
		readModel:   components.NewReadModel(service),
		aboutModel:  components.NewAboutModel(),
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	case components.StateChangeMsg:
		switch msg.NewState {
		case "menu":
			a.state = menuState
		case "upload":
			a.state = uploadState
		case "read":
			a.state = readState
		case "about":
			a.state = aboutState
		}
		return a, nil
	}

	// Update the current component
	var cmd tea.Cmd
	switch a.state {
	case menuState:
		a.menuModel, cmd = a.menuModel.Update(msg)
	case uploadState:
		a.uploadModel, cmd = a.uploadModel.Update(msg)
	case readState:
		a.readModel, cmd = a.readModel.Update(msg)
	case aboutState:
		a.aboutModel, cmd = a.aboutModel.Update(msg)
	}

	return a, cmd
}

// View implements tea.Model
func (a *App) View() string {
	// Make header responsive to screen width
	headerText := "🗂️ OxBin - Walrus Pastebin CLI"
	if a.width < 50 {
		headerText = "🗂️ OxBin"
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#35155d")).
		Background(lipgloss.Color("#dcd6f7")).
		Bold(true).
		Padding(0, 1).
		Width(a.width - 4)

	header := headerStyle.Render(headerText)

	// Get content from current component
	var content string
	switch a.state {
	case menuState:
		content = a.menuModel.ViewWithSize(a.width, a.height)
	case uploadState:
		content = a.uploadModel.ViewWithSize(a.width, a.height)
	case readState:
		content = a.readModel.ViewWithSize(a.width, a.height)
	case aboutState:
		content = a.aboutModel.ViewWithSize(a.width, a.height)
	}

	// Footer
	footerText := "Press 'q' or 'Ctrl+C' to quit"
	if a.width < 30 {
		footerText = "'q' to quit"
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8a8a8a")).
		Width(a.width - 4)

	footer := footerStyle.Render(footerText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}
