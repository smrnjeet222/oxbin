package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/smrnjeet222/oxbin/internal/cli"
	"github.com/smrnjeet222/oxbin/internal/core"
)

func main() {
	// Load configuration
	config := core.DefaultConfig()

	// Create core service
	service, err := core.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create CLI application
	app := cli.NewApp(config, service)

	// Start Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
