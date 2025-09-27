package components

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AboutModel struct{}

func NewAboutModel() AboutModel {
	return AboutModel{}
}

func (m AboutModel) Init() tea.Cmd {
	return nil
}

func (m AboutModel) Update(msg tea.Msg) (AboutModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", " ":
			return m, func() tea.Msg {
				return StateChangeMsg{NewState: "menu"}
			}
		}
	}
	return m, nil
}

func (m AboutModel) View() string {
	return m.ViewWithSize(80, 24) // Default size
}

func (m AboutModel) ViewWithSize(width, height int) string {
	// Adjust container width based on screen size
	containerWidth := max(min(width-8, 80), 50)

	publisherURL := os.Getenv("WALRUS_PUBLISHER_URL")
	aggregatorURL := os.Getenv("WALRUS_AGGREGATOR_URL")

	s := aboutTitleStyle.Render("ℹ️  About OxBin") + "\n\n"

	s += aboutSectionStyle.Render("What is OxBin?") + "\n"
	if width < 70 {
		s += aboutTextStyle.Render("OxBin is a decentralized pastebin built on Walrus Protocol.") + "\n"
		s += aboutTextStyle.Render("Store and retrieve files using Sui blockchain.") + "\n\n"
	} else {
		s += aboutTextStyle.Render("OxBin is a decentralized pastebin service built on top of the Walrus Protocol.") + "\n"
		s += aboutTextStyle.Render("It allows you to store and retrieve files in a decentralized manner using") + "\n"
		s += aboutTextStyle.Render("the Sui blockchain infrastructure.") + "\n\n"
	}

	s += aboutSectionStyle.Render("Features:") + "\n"
	s += aboutFeatureStyle.Render("• 📤 Upload files to Walrus storage") + "\n"
	s += aboutFeatureStyle.Render("• 📥 Retrieve files using Blob IDs") + "\n"
	s += aboutFeatureStyle.Render("• 🔒 Decentralized and secure storage") + "\n"
	if width >= 60 {
		s += aboutFeatureStyle.Render("• 🎨 Beautiful terminal UI with Bubble Tea") + "\n"
	} else {
		s += aboutFeatureStyle.Render("• 🎨 Beautiful terminal UI") + "\n"
	}
	s += aboutFeatureStyle.Render("• 💾 Save retrieved files locally") + "\n\n"

	s += aboutSectionStyle.Render("Current Configuration:") + "\n"

	// Truncate URLs for small screens
	pubURL := publisherURL
	aggURL := aggregatorURL
	if width < 70 {
		if len(pubURL) > 35 {
			pubURL = pubURL[:32] + "..."
		}
		if len(aggURL) > 35 {
			aggURL = aggURL[:32] + "..."
		}
	}

	s += aboutConfigStyle.Render("Publisher:  ") + aboutURLStyle.Render(pubURL) + "\n"
	s += aboutConfigStyle.Render("Aggregator: ") + aboutURLStyle.Render(aggURL) + "\n\n"

	if width >= 60 {
		s += aboutSectionStyle.Render("About Walrus Protocol:") + "\n"
		s += aboutTextStyle.Render("Walrus is a decentralized storage and data availability protocol") + "\n"
		s += aboutTextStyle.Render("built on Sui. It provides cost-effective storage for large data blobs") + "\n"
		s += aboutTextStyle.Render("with high availability and censorship resistance.") + "\n\n"

		s += aboutSectionStyle.Render("Links:") + "\n"
		s += aboutLinkStyle.Render("• Walrus Protocol: https://walrus.site") + "\n"
		s += aboutLinkStyle.Render("• Walrus Go SDK: https://github.com/namihq/walrus-go") + "\n"
		s += aboutLinkStyle.Render("• Bubble Tea: https://github.com/charmbracelet/bubbletea") + "\n\n"
	}

	s += aboutHelpStyle.Render("Press any key to go back to menu")

	return aboutContainerStyle.Width(containerWidth).Render(s)
}

// About styles
var (
	aboutContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#aa96da")).
				Padding(1, 2).
				MarginLeft(2)

	aboutTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#35155d")).
			Background(lipgloss.Color("#ffffd2")).
			Bold(true).
			Padding(0, 1)

	aboutSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fcbad3")).
				Bold(true).
				Underline(true)

	aboutTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f4eeff"))

	aboutFeatureStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a8d8ea")).
				PaddingLeft(2)

	aboutConfigStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fcbad3")).
				Bold(true).
				PaddingLeft(2)

	aboutURLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#aa96da")).
			Underline(true)

	aboutLinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95e1d3")).
			PaddingLeft(2)

	aboutHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a")).
			Italic(true)
)
