package components

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/smrnjeet222/oxbin/internal/core"
	"github.com/smrnjeet222/oxbin/internal/utils"
)

// Constants for backward compatibility - these will be removed when fully migrated to service
const (
	WALRUS_PUBLISHER_URL  = "https://publisher.walrus-testnet.walrus.space"
	WALRUS_AGGREGATOR_URL = "https://aggregator.walrus-testnet.walrus.space"
	BACKUP_PUBLISHER_URL  = "https://walrus-publisher-testnet.staking4all.org"
	BACKUP_AGGREGATOR_URL = "https://walrus-testnet-aggregator.staking4all.org"
)

type uploadStep int

const (
	selectFileStep uploadStep = iota
	confirmUploadStep
	uploadingStep
	uploadCompleteStep
	uploadActionsStep
	uploadErrorStep
)

// Use core types instead of duplicating them

type UploadModel struct {
	step            uploadStep
	textInput       textinput.Model
	spinner         spinner.Model
	selectedFile    string
	blobID          string
	publicURL       string
	walrusScanURL   string
	errorMsg        string
	service         *core.Service
	cursor          int      // For action selection
	terminalHeight  int      // Terminal height for responsive rendering
	currentDir      string   // Current directory for autocompletionha
	suggestions     []string // File path suggestions
	showSuggestions bool     // Whether to show suggestions
}

func NewUploadModel(service *core.Service) UploadModel {
	ti := textinput.New()
	ti.Placeholder = "Enter file path (use Tab for autocompletion)..."
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 50

	// Get current directory for autocompletion
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir, _ = os.UserHomeDir()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize Walrus client with timeout
	// Use the provided service instead of creating a walrus client

	return UploadModel{
		step:            selectFileStep,
		textInput:       ti,
		spinner:         s,
		service:         service,
		currentDir:      currentDir,
		suggestions:     []string{},
		showSuggestions: false,
	}
}

func (m UploadModel) Init() tea.Cmd {
	return textinput.Blink
}

// Generate file path suggestions based on current input
func (m *UploadModel) generateSuggestions() {
	input := m.textInput.Value()
	if input == "" {
		m.suggestions = []string{}
		m.showSuggestions = false
		return
	}

	var dir, prefix string
	if strings.Contains(input, "/") {
		dir = filepath.Dir(input)
		prefix = filepath.Base(input)
	} else {
		dir = m.currentDir
		prefix = input
	}

	// Handle relative paths
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(m.currentDir, dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		m.suggestions = []string{}
		m.showSuggestions = false
		return
	}

	var suggestions []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			if entry.IsDir() {
				suggestions = append(suggestions, name+"/")
			} else {
				suggestions = append(suggestions, name)
			}
		}
	}

	sort.Strings(suggestions)
	m.suggestions = suggestions
	m.showSuggestions = len(suggestions) > 0
}

// Complete the current input with the first suggestion
func (m *UploadModel) completeInput() {
	if len(m.suggestions) == 0 {
		return
	}

	input := m.textInput.Value()
	var dir, prefix string
	if strings.Contains(input, "/") {
		dir = filepath.Dir(input)
		prefix = filepath.Base(input)
	} else {
		dir = ""
		prefix = input
	}

	// Find the first suggestion that matches
	for _, suggestion := range m.suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(prefix)) {
			var newPath string
			if dir != "" {
				newPath = filepath.Join(dir, suggestion)
			} else {
				newPath = suggestion
			}
			m.textInput.SetValue(newPath)
			m.textInput.SetCursor(len(newPath))
			break
		}
	}

	// Update suggestions after completion
	m.generateSuggestions()
}

// Create metadata from file path and info
func createFileMetadata(filePath string, fileInfo os.FileInfo) core.FileMetadata {
	filename := filepath.Base(filePath)
	extension := filepath.Ext(filename)
	contentType := getContentType(extension)

	return core.FileMetadata{
		Filename:    filename,
		Extension:   extension,
		Size:        fileInfo.Size(),
		ContentType: contentType,
		UploadTime:  time.Now(),
		Version:     "1.0",
	}
}

// Get content type based on file extension
func getContentType(extension string) string {
	contentTypes := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".json": "application/json",
		".js":   "text/javascript",
		".css":  "text/css",
		".html": "text/html",
		".htm":  "text/html",
		".xml":  "text/xml",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".toml": "text/plain",
		".go":   "text/plain",
		".py":   "text/plain",
		".rs":   "text/plain",
		".java": "text/plain",
		".c":    "text/plain",
		".cpp":  "text/plain",
		".h":    "text/plain",
		".sh":   "text/plain",
		".log":  "text/plain",
		".cfg":  "text/plain",
		".conf": "text/plain",
		".ini":  "text/plain",
		".csv":  "text/csv",
		".svg":  "image/svg+xml",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
	}

	if contentType, exists := contentTypes[strings.ToLower(extension)]; exists {
		return contentType
	}

	return "application/octet-stream"
}

// Wrap file content with metadata
func wrapFileWithMetadata(filePath string) ([]byte, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Create metadata
	metadata := createFileMetadata(filePath, fileInfo)

	// Encode content as base64
	encodedContent := base64.StdEncoding.EncodeToString(content)

	// Create wrapper
	wrapper := core.BlobWithMetadata{
		Metadata: metadata,
		Content:  encodedContent,
		Type:     "oxbin-file",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %v", err)
	}

	return jsonData, nil
}

// Wrap text to fit within specified width, preserving words when possible
func wrapTextUpload(text string, width int) []string {
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

func (m UploadModel) Update(msg tea.Msg) (UploadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.step == selectFileStep || m.step == uploadCompleteStep || m.step == uploadActionsStep || m.step == uploadErrorStep {
				return m, func() tea.Msg {
					return StateChangeMsg{NewState: "menu"}
				}
			}
		case "enter":
			switch m.step {
			case selectFileStep:
				if m.textInput.Value() != "" {
					m.selectedFile = m.textInput.Value()
					m.step = confirmUploadStep
					return m, nil
				}
			case confirmUploadStep:
				m.step = uploadingStep
				return m, tea.Batch(
					m.spinner.Tick,
					m.uploadFile(),
				)
			case uploadCompleteStep:
				// Move to actions step
				m.step = uploadActionsStep
				m.cursor = 0
				return m, nil
			case uploadActionsStep:
				return m, m.handleAction()
			case uploadErrorStep:
				// Reset for new upload
				m.step = selectFileStep
				m.selectedFile = ""
				m.blobID = ""
				m.publicURL = ""
				m.walrusScanURL = ""
				m.errorMsg = ""
				m.cursor = 0
				m.suggestions = []string{}
				m.showSuggestions = false
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, nil
			}
		case "up", "k":
			if m.step == uploadActionsStep && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.step == uploadActionsStep && m.cursor < 4 { // 5 options (0-4)
				m.cursor++
			}
		case "tab":
			if m.step == selectFileStep {
				m.completeInput()
				return m, nil
			}
		case "n":
			if m.step == confirmUploadStep {
				m.step = selectFileStep
				m.selectedFile = ""
				return m, nil
			}
		}

	case uploadCompleteMsg:
		m.step = uploadCompleteStep
		m.blobID = string(msg)

		// Generate public URL using hosted WebUI
		m.publicURL = utils.GetWebUIURL(m.blobID)

		// Determine network for WalrusScan
		network := "testnet"
		if strings.Contains(WALRUS_AGGREGATOR_URL, "mainnet") {
			network = "mainnet"
		}
		m.walrusScanURL = fmt.Sprintf("https://walruscan.com/%s/blob/%s", network, m.blobID)

		// Copy Blob ID to clipboard
		if err := clipboard.WriteAll(m.blobID); err == nil {
			// Successfully copied to clipboard
		}

		return m, nil

	case uploadErrorMsg:
		m.step = uploadErrorStep
		m.errorMsg = string(msg)
		return m, nil

	case uploadResetMsg:
		// Reset for new upload
		m.step = selectFileStep
		m.selectedFile = ""
		m.blobID = ""
		m.publicURL = ""
		m.walrusScanURL = ""
		m.errorMsg = ""
		m.cursor = 0
		m.suggestions = []string{}
		m.showSuggestions = false
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, nil

	case clipboardSuccessMsg:
		// Stay in actions step, maybe show a brief success message
		return m, nil

	case clipboardErrorMsg:
		// Stay in actions step, could show error but for now just continue
		return m, nil

	case spinner.TickMsg:
		if m.step == uploadingStep {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Handle textinput updates and generate suggestions
	if m.step == selectFileStep {
		var cmd tea.Cmd
		oldValue := m.textInput.Value()
		m.textInput, cmd = m.textInput.Update(msg)

		// Generate suggestions if the input changed
		if m.textInput.Value() != oldValue {
			m.generateSuggestions()
		}

		return m, cmd
	}

	return m, nil
}

func (m UploadModel) View() string {
	return m.ViewWithSize(80, 24) // Default size
}

func (m UploadModel) ViewWithSize(width, height int) string {
	// Adjust container width based on screen size
	containerWidth := max(min(width-8, 80), 40)

	// Update text input width based on container
	m.textInput.Width = max(containerWidth-10, 20)

	// Store height for use in render functions
	m.terminalHeight = height

	switch m.step {
	case selectFileStep:
		return m.renderSelectFile(containerWidth)
	case confirmUploadStep:
		return m.renderConfirmUpload(containerWidth)
	case uploadingStep:
		return m.renderUploading(containerWidth)
	case uploadCompleteStep:
		return m.renderUploadComplete(containerWidth)
	case uploadActionsStep:
		return m.renderUploadActions(containerWidth)
	case uploadErrorStep:
		return m.renderUploadError(containerWidth)
	}
	return ""
}

func (m UploadModel) renderSelectFile(containerWidth int) string {
	s := uploadTitleStyle.Render("📤 Upload File to Walrus") + "\n\n"

	s += uploadSubtitleStyle.Render("Enter file path:") + "\n"
	s += m.textInput.View() + "\n"

	// Show current directory
	currentDir := m.currentDir
	if len(currentDir) > 50 && containerWidth < 80 {
		currentDir = "..." + currentDir[len(currentDir)-47:]
	}
	s += uploadInfoStyle.Render(fmt.Sprintf("Current directory: %s", currentDir)) + "\n"

	// Show suggestions if available
	if m.showSuggestions && len(m.suggestions) > 0 {
		s += "\n" + uploadSubtitleStyle.Render("Suggestions:") + "\n"

		// Calculate how many suggestions we can show based on terminal height
		// Account for: main layout(4) + container style(4) + title(1) + spacing(1) + subtitle(1) + input(1) +
		// current_dir(1) + spacing(1) + suggestions_title(1) + help(1) + spacing(1) = 13 lines
		availableHeight := m.terminalHeight - 4 - 4 - 13 // main layout + container + other content
		maxSuggestionsByHeight := max(availableHeight, 1)
		maxSuggestions := min(len(m.suggestions), min(5, maxSuggestionsByHeight))

		for i := range maxSuggestions {
			suggestion := m.suggestions[i]
			if len(suggestion) > containerWidth-10 {
				suggestion = suggestion[:containerWidth-13] + "..."
			}
			s += uploadInfoStyle.Render(fmt.Sprintf("  %s", suggestion)) + "\n"
		}

		if len(m.suggestions) > maxSuggestions {
			s += uploadInfoStyle.Render(fmt.Sprintf("  ... and %d more", len(m.suggestions)-maxSuggestions)) + "\n"
		}
	}

	s += "\n" + uploadHelpStyle.Render("Press Tab for autocompletion, Enter to confirm, Esc to go back")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

func (m UploadModel) renderConfirmUpload(containerWidth int) string {
	s := uploadTitleStyle.Render("📤 Confirm Upload") + "\n\n"

	// Handle long filenames with wrapping
	filename := filepath.Base(m.selectedFile)
	contentWidth := containerWidth - 10 // Account for label and padding
	filenameLines := wrapTextUpload(filename, contentWidth)
	s += "File: " + uploadFileStyle.Render(filenameLines[0]) + "\n"
	for i := 1; i < len(filenameLines); i++ {
		s += "      " + uploadFileStyle.Render(filenameLines[i]) + "\n"
	}

	// Handle long paths with wrapping
	pathLines := wrapTextUpload(m.selectedFile, contentWidth)
	s += "Path: " + uploadPathStyle.Render(pathLines[0]) + "\n"
	for i := 1; i < len(pathLines); i++ {
		s += "      " + uploadPathStyle.Render(pathLines[i]) + "\n"
	}
	s += "\n"

	// Show file info if possible
	if info, err := os.Stat(m.selectedFile); err == nil {
		s += fmt.Sprintf("Size: %s\n", uploadInfoStyle.Render(formatFileSize(info.Size())))
		s += fmt.Sprintf("Modified: %s\n\n", uploadInfoStyle.Render(info.ModTime().Format("2006-01-02 15:04:05")))
	}

	s += uploadQuestionStyle.Render("Upload this file to Walrus? (y/N)") + "\n\n"
	s += uploadHelpStyle.Render("Press Enter to upload, 'n' to cancel, Esc to go back")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

func (m UploadModel) renderUploading(containerWidth int) string {
	s := uploadTitleStyle.Render("📤 Uploading to Walrus") + "\n\n"
	s += fmt.Sprintf("%s Uploading %s...\n\n", m.spinner.View(), filepath.Base(m.selectedFile))

	// Show current endpoints for debugging (truncate if needed)
	publisherURL := WALRUS_PUBLISHER_URL
	aggregatorURL := WALRUS_AGGREGATOR_URL

	if containerWidth < 60 {
		// Truncate URLs for small screens
		if len(publisherURL) > 35 {
			publisherURL = publisherURL[:32] + "..."
		}
		if len(aggregatorURL) > 35 {
			aggregatorURL = aggregatorURL[:32] + "..."
		}
	}

	s += uploadInfoStyle.Render(fmt.Sprintf("Publisher: %s", publisherURL)) + "\n"
	s += uploadInfoStyle.Render(fmt.Sprintf("Aggregator: %s", aggregatorURL)) + "\n\n"

	s += uploadHelpStyle.Render("Please wait while your file is being stored on Walrus...")
	s += "\n" + uploadHelpStyle.Render("This may take up to 60 seconds...")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

func (m UploadModel) renderUploadComplete(containerWidth int) string {
	s := uploadSuccessStyle.Render("✅ Upload Successful!") + "\n\n"

	// Handle long filenames with wrapping
	filename := filepath.Base(m.selectedFile)
	contentWidth := containerWidth - 10 // Account for label and padding
	filenameLines := wrapTextUpload(filename, contentWidth)
	s += "File: " + uploadFileStyle.Render(filenameLines[0]) + "\n"
	for i := 1; i < len(filenameLines); i++ {
		s += "      " + uploadFileStyle.Render(filenameLines[i]) + "\n"
	}

	// Handle long blob IDs with wrapping
	blobIDLines := wrapTextUpload(m.blobID, contentWidth)
	s += "Blob ID: " + uploadBlobIDStyle.Render(blobIDLines[0]) + "\n"
	for i := 1; i < len(blobIDLines); i++ {
		s += "         " + uploadBlobIDStyle.Render(blobIDLines[i]) + "\n"
	}

	s += uploadInfoStyle.Render("📋 Blob ID copied to clipboard!") + "\n\n"
	s += uploadInfoStyle.Render("💡 Save the Blob ID to retrieve your file later!") + "\n\n"
	s += uploadHelpStyle.Render("Press Enter for more options, Esc to go back to menu")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

func (m UploadModel) renderUploadActions(containerWidth int) string {
	s := uploadTitleStyle.Render("🎯 What would you like to do?") + "\n\n"

	actions := []string{
		"🌐 Open in browser",
		"🔍 View on WalrusScan",
		"📋 Copy Blob ID",
		"📤 Upload another file",
		"🏠 Back to main menu",
	}

	for i, action := range actions {
		cursor := " "
		if m.cursor == i {
			cursor = menuCursorStyle.Render("▶")
			action = menuSelectedStyle.Render(action)
		} else {
			action = menuItemStyle.Render(action)
		}
		s += fmt.Sprintf("%s %s\n", cursor, action)
	}

	s += "\n" + uploadHelpStyle.Render("Use ↑/↓ or j/k to navigate, Enter to select")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

func (m UploadModel) renderUploadError(containerWidth int) string {
	s := uploadErrorStyle.Render("❌ Upload Failed") + "\n\n"

	// Wrap error message for small screens
	errorMsg := m.errorMsg
	if containerWidth < 60 && len(errorMsg) > 50 {
		// Simple word wrapping for error messages
		words := strings.Fields(errorMsg)
		var lines []string
		var currentLine string

		for _, word := range words {
			if len(currentLine)+len(word)+1 <= 45 {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
		errorMsg = strings.Join(lines, "\n")
	}

	s += fmt.Sprintf("Error: %s\n\n", uploadErrorMsgStyle.Render(errorMsg))
	s += uploadHelpStyle.Render("Press Enter to try again, Esc to go back to menu")

	return uploadContainerStyle.Width(containerWidth).Render(s)
}

// Handle action selection
func (m UploadModel) handleAction() tea.Cmd {
	switch m.cursor {
	case 0: // Open in browser
		return func() tea.Msg {
			openURL(m.publicURL)
			return nil
		}
	case 1: // View on WalrusScan
		return func() tea.Msg {
			openURL(m.walrusScanURL)
			return nil
		}
	case 2: // Copy Blob ID
		return func() tea.Msg {
			if clipErr := clipboard.WriteAll(m.blobID); clipErr == nil {
				return clipboardSuccessMsg{}
			} else {
				return clipboardErrorMsg{error: clipErr}
			}
		}
	case 3: // Upload another file
		return func() tea.Msg {
			// Reset for new upload
			return uploadResetMsg{}
		}
	case 4: // Back to main menu
		return func() tea.Msg {
			return StateChangeMsg{NewState: "menu"}
		}
	}
	return nil
}

// Open URL in default browser
func openURL(url string) error {
	return utils.OpenBrowser(url)
}

// Upload file command
func (m UploadModel) uploadFile() tea.Cmd {
	return func() tea.Msg {
		// Check if file exists
		if _, err := os.Stat(m.selectedFile); os.IsNotExist(err) {
			return uploadErrorMsg("File does not exist")
		}

		// Check file size (optional - add reasonable limits)
		fileInfo, err := os.Stat(m.selectedFile)
		if err != nil {
			return uploadErrorMsg(fmt.Sprintf("Cannot read file info: %v", err))
		}

		// Limit file size to 10MB for demo purposes
		if fileInfo.Size() > 10*1024*1024 {
			return uploadErrorMsg("File too large (max 10MB)")
		}

		// Create context with timeout for the upload operation
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Channel to receive the result
		resultChan := make(chan uploadResult, 1)

		// Run upload in goroutine with timeout
		go func() {
			// Use service to upload file
			ctx := context.Background()
			response, err := m.service.UploadFile(ctx, m.selectedFile)
			var blobID string
			if response != nil {
				blobID = response.BlobID
			}
			resultChan <- uploadResult{
				blobID: blobID,
				err:    err,
			}
		}()

		// Wait for result or timeout
		select {
		case result := <-resultChan:
			if result.err != nil {
				return uploadErrorMsg(fmt.Sprintf("Failed to upload file: %v", result.err))
			}

			// Check if blob ID is empty
			if result.blobID == "" {
				return uploadErrorMsg("Received empty blob ID from service")
			}

			return uploadCompleteMsg(result.blobID)

		case <-ctx.Done():
			return uploadErrorMsg("Upload timed out after 60 seconds. Please check your network connection and try again.")
		}
	}
}

// Helper struct for upload result
type uploadResult struct {
	blobID string
	err    error
}

// Messages
type uploadCompleteMsg string
type uploadErrorMsg string
type uploadResetMsg struct{}
type clipboardSuccessMsg struct{}
type clipboardErrorMsg struct {
	error error
}

// Helper function to format file size
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// Upload styles
var (
	uploadContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#aa96da")).
				Padding(1, 2).
				MarginLeft(2)

	uploadTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#35155d")).
				Background(lipgloss.Color("#fcbad3")).
				Bold(true).
				Padding(0, 1)

	uploadSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f4eeff")).
				Bold(true)

	uploadFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a8d8ea")).
			Bold(true)

	uploadPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a"))

	uploadInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a"))

	uploadQuestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fcbad3")).
				Bold(true)

	uploadSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#95e1d3")).
				Bold(true)

	uploadBlobIDStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#35155d")).
				Background(lipgloss.Color("#f4eeff")).
				Padding(0, 1).
				Bold(true)

	uploadErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9b572")).
				Bold(true)

	uploadErrorMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9b572"))

	uploadHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8a8a")).
			Italic(true)
)
