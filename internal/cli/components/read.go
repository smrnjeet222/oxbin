package components

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeet/oxbin/internal/core"
	walrus "github.com/namihq/walrus-go"
)

type readStep int

const (
	enterBlobIDStep readStep = iota
	fetchingStep
	displayContentStep
	saveFileStep
	readErrorStep
)

type ReadModel struct {
	step         readStep
	textInput    textinput.Model
	saveInput    textinput.Model
	viewport     viewport.Model
	spinner      spinner.Model
	blobID       string
	content      string // Processed content for display
	rawContent   []byte // Raw content for saving
	errorMsg     string
	service      *core.Service
	metadata     *walrus.BlobMetadata
	fileMetadata *core.FileMetadata // File metadata from OxBin
	isOxBinFile  bool               // Whether this is an OxBin file with metadata
}

func NewReadModel(service *core.Service) ReadModel {
	ti := textinput.New()
	ti.Placeholder = "Enter Blob ID..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	si := textinput.New()
	si.Placeholder = "Enter file path to save..."
	si.CharLimit = 256
	si.Width = 50

	vp := viewport.New(80, 20)
	vp.Style = readViewportStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize Walrus client with hardcoded endpoints and timeout
	// Use the provided service instead of creating a walrus client

	return ReadModel{
		step:        enterBlobIDStep,
		textInput:   ti,
		saveInput:   si,
		viewport:    vp,
		spinner:     s,
		service:     service,
		isOxBinFile: false,
	}
}

func (m ReadModel) Init() tea.Cmd {
	return nil
}

func (m ReadModel) Update(msg tea.Msg) (ReadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = msg.Height - 12

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			switch m.step {
			case enterBlobIDStep, displayContentStep, readErrorStep:
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: "menu"}
				}
			case saveFileStep:
				m.step = displayContentStep
				return m, nil
			}
		case "enter":
			switch m.step {
			case enterBlobIDStep:
				if m.textInput.Value() != "" {
					m.blobID = strings.TrimSpace(m.textInput.Value())
					m.step = fetchingStep
					return m, tea.Batch(
						m.spinner.Tick,
						m.fetchContent(),
					)
				}
			case displayContentStep:
				// No action on enter in display mode
			case saveFileStep:
				if m.saveInput.Value() != "" {
					return m, m.saveToFile()
				}
			case readErrorStep:
				// Reset for new attempt
				m.step = enterBlobIDStep
				m.blobID = ""
				m.content = ""
				m.rawContent = nil
				m.errorMsg = ""
				m.textInput.SetValue("")
				return m, nil
			}
		case "s":
			if m.step == displayContentStep {
				m.step = saveFileStep
				m.saveInput.Focus()
				// Pre-populate with original filename if available
				if m.isOxBinFile && m.fileMetadata != nil {
					m.saveInput.SetValue(m.fileMetadata.Filename)
				}
				return m, nil
			}
		case "r":
			if m.step == displayContentStep {
				// Refresh content
				m.step = fetchingStep
				return m, tea.Batch(
					m.spinner.Tick,
					m.fetchContent(),
				)
			}
		}

	case fetchCompleteMsg:
		m.step = displayContentStep
		m.rawContent = msg.rawContent
		m.metadata = msg.metadata
		m.fileMetadata = msg.fileMetadata
		m.isOxBinFile = msg.isOxBinFile

		// Process content for display - viewport will handle scrolling
		m.content = processContentForDisplay(msg.rawContent, msg.fileMetadata)
		m.viewport.SetContent(m.content)
		return m, nil

	case fetchErrorMsg:
		m.step = readErrorStep
		m.errorMsg = string(msg)
		return m, nil

	case saveCompleteMsg:
		m.step = displayContentStep
		return m, nil

	case saveErrorMsg:
		// Stay in save step but show error
		return m, nil

	case spinner.TickMsg:
		if m.step == fetchingStep {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Handle input updates
	switch m.step {
	case enterBlobIDStep:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	case displayContentStep:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case saveFileStep:
		var cmd tea.Cmd
		m.saveInput, cmd = m.saveInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ReadModel) View() string {
	return m.ViewWithSize(80, 24) // Default size
}

func (m ReadModel) ViewWithSize(width, height int) string {
	// Adjust container width based on screen size
	containerWidth := max(min(width-8, 80), 40)

	// Update text input widths based on container
	m.textInput.Width = containerWidth - 10
	m.saveInput.Width = containerWidth - 10
	if m.textInput.Width < 20 {
		m.textInput.Width = 20
	}
	if m.saveInput.Width < 20 {
		m.saveInput.Width = 20
	}

	// Update viewport size based on current step
	m.viewport.Width = containerWidth - 4

	// Calculate viewport height dynamically based on step and content
	// Account for main layout: header (1) + spacing (1) + footer (1) + spacing (1) = 4 lines
	// Account for container style: border (2) + padding (2) = 4 lines
	availableHeight := height - 4 - 4 // main layout + container overhead

	switch m.step {
	case displayContentStep:
		// Calculate metadata lines for accurate height
		metadataLines := m.calculateMetadataLines(containerWidth)
		// Title (1) + spacing (1) + metadata + spacing (1) + viewport spacing (1) + help (1)
		usedHeight := 1 + 1 + metadataLines + 1 + 1 + 1
		m.viewport.Height = max(availableHeight-usedHeight, 3)
	default:
		// For other steps, use a conservative estimate
		m.viewport.Height = max(availableHeight-8, 3)
	}

	switch m.step {
	case enterBlobIDStep:
		return m.renderEnterBlobID(containerWidth)
	case fetchingStep:
		return m.renderFetching(containerWidth)
	case displayContentStep:
		return m.renderDisplayContent(containerWidth)
	case saveFileStep:
		return m.renderSaveFile(containerWidth)
	case readErrorStep:
		return m.renderReadError(containerWidth)
	}
	return ""
}

func (m ReadModel) renderEnterBlobID(containerWidth int) string {
	s := readTitleStyle.Render("📥 Read File from Walrus") + "\n\n"
	s += readSubtitleStyle.Render("Enter the Blob ID:") + "\n"
	s += m.textInput.View() + "\n\n"
	s += readHelpStyle.Render("Press Enter to fetch file, Esc to go back")

	return readContainerStyle.Width(containerWidth).Render(s)
}

func (m ReadModel) renderFetching(containerWidth int) string {
	s := readTitleStyle.Render("📥 Fetching from Walrus") + "\n\n"
	s += fmt.Sprintf("%s Retrieving blob %s...\n\n", m.spinner.View(), m.blobID)
	s += readHelpStyle.Render("Please wait while your file is being retrieved...")

	return readContainerStyle.Width(containerWidth).Render(s)
}

func (m ReadModel) renderDisplayContent(containerWidth int) string {
	s := readTitleStyle.Render("📄 File Content") + "\n\n"

	// Available width for content (subtract emoji and label space)
	contentWidth := containerWidth - 15

	// Show file metadata if this is an OxBin file
	if m.isOxBinFile && m.fileMetadata != nil {
		// Filename with wrapping
		filenameLines := wrapText(m.fileMetadata.Filename, contentWidth)
		s += "📁 Filename: " + readInfoStyle.Render(filenameLines[0]) + "\n"
		for i := 1; i < len(filenameLines); i++ {
			s += "           " + readInfoStyle.Render(filenameLines[i]) + "\n"
		}

		s += fmt.Sprintf("📏 Size: %s\n", readInfoStyle.Render(formatFileSize(m.fileMetadata.Size)))
		s += fmt.Sprintf("🏷️  Type: %s\n", readInfoStyle.Render(m.fileMetadata.ContentType))
		s += fmt.Sprintf("📅 Uploaded: %s\n", readInfoStyle.Render(m.fileMetadata.UploadTime.Format("2006-01-02 15:04:05")))

		// Blob ID with wrapping
		blobIDLines := wrapText(m.blobID, contentWidth)
		s += "🆔 Blob ID: " + readBlobIDStyle.Render(blobIDLines[0]) + "\n"
		for i := 1; i < len(blobIDLines); i++ {
			s += "           " + readBlobIDStyle.Render(blobIDLines[i]) + "\n"
		}
		s += "\n"
	} else {
		// Show basic metadata for non-OxBin files
		// Blob ID with wrapping
		blobIDLines := wrapText(m.blobID, contentWidth)
		s += "🆔 Blob ID: " + readBlobIDStyle.Render(blobIDLines[0]) + "\n"
		for i := 1; i < len(blobIDLines); i++ {
			s += "           " + readBlobIDStyle.Render(blobIDLines[i]) + "\n"
		}

		if m.metadata != nil {
			s += fmt.Sprintf("📏 Size: %s\n", readInfoStyle.Render(formatFileSize(m.metadata.ContentLength)))
			if m.metadata.ContentType != "" {
				s += fmt.Sprintf("🏷️  Type: %s\n", readInfoStyle.Render(m.metadata.ContentType))
			}
		}
		s += "\n"
	}

	// Content viewport
	s += m.viewport.View() + "\n\n"

	s += readHelpStyle.Render("↑/↓ to scroll • 's' to save • 'r' to refresh • Esc to go back")

	return readContainerStyle.Width(containerWidth).Render(s)
}

func (m ReadModel) renderSaveFile(containerWidth int) string {
	s := readTitleStyle.Render("💾 Save File") + "\n\n"

	// Show suggested filename if available
	if m.isOxBinFile && m.fileMetadata != nil {
		contentWidth := containerWidth - 20 // Account for label and padding
		filenameLines := wrapText(m.fileMetadata.Filename, contentWidth)
		s += "💡 Suggested filename: " + readInfoStyle.Render(filenameLines[0]) + "\n"
		for i := 1; i < len(filenameLines); i++ {
			s += "                    " + readInfoStyle.Render(filenameLines[i]) + "\n"
		}
		s += "\n"
	}

	s += readSubtitleStyle.Render("Enter file path to save:") + "\n"
	s += m.saveInput.View() + "\n\n"
	s += readHelpStyle.Render("Press Enter to save, Esc to cancel")

	return readContainerStyle.Width(containerWidth).Render(s)
}

func (m ReadModel) renderReadError(containerWidth int) string {
	s := readErrorStyle.Render("❌ Failed to Read File") + "\n\n"
	s += fmt.Sprintf("Error: %s\n\n", readErrorMsgStyle.Render(m.errorMsg))
	s += readHelpStyle.Render("Press Enter to try again, Esc to go back to menu")

	return readContainerStyle.Width(containerWidth).Render(s)
}

// Wrap text to fit within specified width, preserving words when possible
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	remaining := text

	for len(remaining) > width {
		// Try to find a good break point (space, slash, dash)
		breakPoint := width
		for i := width - 1; i >= width/2; i-- {
			if remaining[i] == ' ' || remaining[i] == '/' || remaining[i] == '-' || remaining[i] == '_' {
				breakPoint = i
				break
			}
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = remaining[breakPoint:]

		// Skip leading spaces in the next line
		for len(remaining) > 0 && remaining[0] == ' ' {
			remaining = remaining[1:]
		}
	}

	if len(remaining) > 0 {
		lines = append(lines, remaining)
	}

	return lines
}

// Check if content is binary (contains non-printable characters)
func isBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check first 512 bytes for binary content detection
	sampleSize := min(len(data), 512)
	sample := data[:sampleSize]

	// Count non-printable characters
	nonPrintable := 0
	for _, b := range sample {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		}
	}

	// If more than 30% are non-printable, consider it binary
	return float64(nonPrintable)/float64(sampleSize) > 0.3
}

// Check if content is valid UTF-8 text
func isValidUTF8(data []byte) bool {
	return utf8.Valid(data)
}

// Process content for display in viewport
func processContentForDisplay(rawContent []byte, fileMetadata *core.FileMetadata) string {
	// First check if it's binary content
	if isBinaryContent(rawContent) {
		return generateBinaryPreview(rawContent, fileMetadata)
	}

	// Convert to string and check if it's valid UTF-8
	content := string(rawContent)
	if !isValidUTF8(rawContent) {
		return generateBinaryPreview(rawContent, fileMetadata)
	}

	// For text content, return full content - viewport will handle scrolling
	return content
}

// Generate a preview for binary files
func generateBinaryPreview(data []byte, fileMetadata *core.FileMetadata) string {
	var preview strings.Builder

	// File type information
	if fileMetadata != nil {
		preview.WriteString(fmt.Sprintf("📄 Binary File: %s\n", fileMetadata.ContentType))
		preview.WriteString(fmt.Sprintf("📏 Size: %s\n\n", formatFileSize(fileMetadata.Size)))
	} else {
		preview.WriteString("📄 Binary File\n")
		preview.WriteString(fmt.Sprintf("📏 Size: %s\n\n", formatFileSize(int64(len(data)))))
	}

	// Show hex dump - more bytes for better preview, but organized
	preview.WriteString("🔍 Hex Preview:\n")
	preview.WriteString("=" + strings.Repeat("=", 50) + "\n")

	// Show more bytes for better preview (up to 1KB)
	dumpSize := min(len(data), 1024)
	for i := 0; i < dumpSize; i += 16 {
		end := min(i+16, dumpSize)
		line := data[i:end]

		// Offset
		preview.WriteString(fmt.Sprintf("%08x  ", i))

		// Hex bytes
		for j := 0; j < 16; j++ {
			if j < len(line) {
				preview.WriteString(fmt.Sprintf("%02x ", line[j]))
			} else {
				preview.WriteString("   ")
			}
			if j == 7 {
				preview.WriteString(" ")
			}
		}

		// ASCII representation
		preview.WriteString(" |")
		for _, b := range line {
			if b >= 32 && b <= 126 {
				preview.WriteString(string(b))
			} else {
				preview.WriteString(".")
			}
		}
		preview.WriteString("|\n")
	}

	if len(data) > 1024 {
		preview.WriteString(fmt.Sprintf("\n... and %d more bytes (use 's' to save complete file)\n", len(data)-1024))
	}

	preview.WriteString("\n💡 This is a binary file. Use 's' to save it locally to view properly.")

	return preview.String()
}

// Calculate how many lines the metadata will take, accounting for text wrapping
func (m ReadModel) calculateMetadataLines(containerWidth int) int {
	// Available width for content (subtract emoji and label space)
	contentWidth := containerWidth - 15 // Account for emoji, label, and padding

	if m.isOxBinFile && m.fileMetadata != nil {
		lines := 0
		// Filename
		lines += len(wrapText(m.fileMetadata.Filename, contentWidth))
		// Size (usually short)
		lines += 1
		// Type (usually short)
		lines += 1
		// Upload date (fixed format)
		lines += 1
		// Blob ID (can be long)
		lines += len(wrapText(m.blobID, contentWidth))
		// Spacing
		lines += 1
		return lines
	} else {
		lines := 0
		// Blob ID (can be long)
		lines += len(wrapText(m.blobID, contentWidth))
		if m.metadata != nil {
			// Size (usually short)
			lines += 1
			if m.metadata.ContentType != "" {
				// Content type (usually short)
				lines += 1
			}
		}
		// Spacing
		lines += 1
		return lines
	}
}

// Unwrap file content and metadata
func unwrapFileWithMetadata(data []byte) ([]byte, *core.FileMetadata, bool, error) {
	// Try to parse as OxBin file with metadata
	var wrapper core.BlobWithMetadata
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Not a JSON file or not our format, return as plain content
		return data, nil, false, nil
	}

	// Check if it's our format
	if wrapper.Type != "oxbin-file" {
		// Not our format, return as plain content
		return data, nil, false, nil
	}

	// Decode the base64 content
	content, err := base64.StdEncoding.DecodeString(wrapper.Content)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to decode content: %v", err)
	}

	return content, &wrapper.Metadata, true, nil
}

// Fetch content command
func (m ReadModel) fetchContent() tea.Cmd {
	return func() tea.Msg {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Channel to receive the result
		resultChan := make(chan fetchResult, 1)

		// Run fetch in goroutine with timeout
		go func() {
			// Use service to read file
			ctx := context.Background()
			response, err := m.service.ReadFile(ctx, m.blobID)
			if err != nil {
				resultChan <- fetchResult{err: fmt.Errorf("failed to read file: %v", err)}
				return
			}

			// Check if the service call was successful
			if !response.Success {
				resultChan <- fetchResult{err: fmt.Errorf("service error: %s", response.Error)}
				return
			}

			// The service already unwraps the metadata for us
			resultChan <- fetchResult{
				rawContent:   response.RawContent,
				metadata:     nil, // Will be nil for now
				fileMetadata: response.FileMetadata,
				isOxBinFile:  response.IsOxBinFile,
				err:          nil,
			}
		}()

		// Wait for result or timeout
		select {
		case result := <-resultChan:
			if result.err != nil {
				return fetchErrorMsg(result.err.Error())
			}
			return fetchCompleteMsg{
				rawContent:   result.rawContent,
				metadata:     result.metadata,
				fileMetadata: result.fileMetadata,
				isOxBinFile:  result.isOxBinFile,
			}
		case <-ctx.Done():
			return fetchErrorMsg("Fetch timed out after 30 seconds. Please check your network connection and try again.")
		}
	}
}

// Helper struct for fetch result
type fetchResult struct {
	rawContent   []byte
	metadata     *walrus.BlobMetadata
	fileMetadata *core.FileMetadata
	isOxBinFile  bool
	err          error
}

// Save to file command
func (m ReadModel) saveToFile() tea.Cmd {
	return func() tea.Msg {
		filePath := strings.TrimSpace(m.saveInput.Value())

		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return saveErrorMsg(fmt.Sprintf("Failed to create directory: %v", err))
		}

		// Write content to file
		if err := os.WriteFile(filePath, m.rawContent, 0644); err != nil {
			return saveErrorMsg(fmt.Sprintf("Failed to save file: %v", err))
		}

		return saveCompleteMsg(filePath)
	}
}

// Messages
type fetchCompleteMsg struct {
	rawContent   []byte
	metadata     *walrus.BlobMetadata
	fileMetadata *core.FileMetadata
	isOxBinFile  bool
}
type fetchErrorMsg string
type saveCompleteMsg string
type saveErrorMsg string

// Read styles
var (
	readContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#aa96da")).
				Padding(1, 2).
				MarginLeft(2)

	readTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#35155d")).
			Background(lipgloss.Color("#95e1d3")).
			Bold(true).
			Padding(0, 1)

	readSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f4eeff")).
				Bold(true)

	readBlobIDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#35155d")).
			Background(lipgloss.Color("#f4eeff")).
			Padding(0, 1).
			Bold(true)

	readInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a"))

	readViewportStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#8a8a8a")).
				Padding(1)

	readErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9b572")).
			Bold(true)

	readErrorMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9b572"))

	readHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a")).
			Italic(true)
)
