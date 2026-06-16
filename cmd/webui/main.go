package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smrnjeet222/oxbin/internal/core"
	"github.com/smrnjeet222/oxbin/internal/utils"
	"github.com/smrnjeet222/oxbin/internal/webui/templates"
)

type blobService interface {
	ReadFile(ctx context.Context, blobID string) (*core.ReadResponse, error)
	UploadBytes(ctx context.Context, filename string, content []byte) (*core.UploadResponse, error)
	MaxFileSize() int64
}

type WebServer struct {
	service blobService
	port    string
}

type PageData struct {
	Title       string
	BlobID      string
	Content     string
	RawContent  []byte
	Metadata    *core.FileMetadata
	IsOxBinFile bool
	IsBinary    bool
	Error       string
	ContentType string
	Size        int64
}

type WebUploadResponse struct {
	Success       bool   `json:"success"`
	BlobID        string `json:"blob_id,omitempty"`
	Filename      string `json:"filename,omitempty"`
	Size          int64  `json:"size,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	UploadedAt    string `json:"uploaded_at,omitempty"`
	ViewerURL     string `json:"viewer_url,omitempty"`
	DownloadURL   string `json:"download_url,omitempty"`
	WalrusScanURL string `json:"walrus_scan_url,omitempty"`
	PublicURL     string `json:"public_url,omitempty"`
	Error         string `json:"error,omitempty"`
}

func main() {
	// Initialize service with default config (this loads .env file)
	config := core.DefaultConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	service, err := core.NewService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	server := &WebServer{
		service: service,
		port:    port,
	}

	// Setup routes
	http.HandleFunc("/", server.handleHome)
	http.HandleFunc("/history", server.handleHistory)
	http.HandleFunc("/upload", server.handleUpload)
	http.HandleFunc("/api/upload", server.handleUpload)
	http.HandleFunc("/blob/", server.handleBlob)
	http.HandleFunc("/api/blob/", server.handleAPIBlob)
	http.HandleFunc("/download/", server.handleDownload)

	// Serve static assets
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets/"))))

	log.Printf("🌐 OxBin Web UI starting on port %s", port)
	log.Printf("📍 Access at: http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (ws *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		blobID := strings.TrimSpace(r.FormValue("blob_id"))
		if blobID != "" {
			http.Redirect(w, r, "/blob/"+blobID, http.StatusSeeOther)
			return
		}
	}

	// Render the index page using Templ
	component := templates.Index(utils.FormatFileSize(ws.service.MaxFileSize()))
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (ws *WebServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	component := templates.History()
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (ws *WebServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxFileSize := ws.service.MaxFileSize()
	wantsJSON := r.URL.Path == "/api/upload" ||
		strings.Contains(r.Header.Get("Accept"), "application/json") ||
		r.Header.Get("X-Requested-With") == "XMLHttpRequest"

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+1024*1024)
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		ws.writeUploadError(w, wantsJSON, http.StatusRequestEntityTooLarge, fmt.Sprintf("Upload could not be read. Maximum file size is %s.", utils.FormatFileSize(maxFileSize)))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		ws.writeUploadError(w, wantsJSON, http.StatusBadRequest, "Choose a file before uploading.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		ws.writeUploadError(w, wantsJSON, http.StatusBadRequest, "Failed to read uploaded file.")
		return
	}

	if int64(len(content)) > maxFileSize {
		ws.writeUploadError(w, wantsJSON, http.StatusRequestEntityTooLarge, fmt.Sprintf("File is too large. Maximum size is %s.", utils.FormatFileSize(maxFileSize)))
		return
	}

	if len(content) == 0 {
		ws.writeUploadError(w, wantsJSON, http.StatusBadRequest, "Empty files are not supported yet.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	response, err := ws.service.UploadBytes(ctx, header.Filename, content)
	if err != nil {
		ws.writeUploadError(w, wantsJSON, http.StatusInternalServerError, fmt.Sprintf("Upload failed: %v", err))
		return
	}

	if response == nil || !response.Success {
		message := "Upload failed. Please try again."
		if response != nil && response.Error != "" {
			message = response.Error
		}
		ws.writeUploadError(w, wantsJSON, http.StatusBadGateway, message)
		return
	}

	viewerURL := "/blob/" + response.BlobID
	if wantsJSON {
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(content)
		}

		ws.writeJSON(w, http.StatusOK, WebUploadResponse{
			Success:       true,
			BlobID:        response.BlobID,
			Filename:      header.Filename,
			Size:          int64(len(content)),
			ContentType:   contentType,
			UploadedAt:    time.Now().UTC().Format(time.RFC3339),
			ViewerURL:     viewerURL,
			DownloadURL:   "/download/" + response.BlobID,
			WalrusScanURL: response.WalrusScanURL,
			PublicURL:     response.PublicURL,
		})
		return
	}

	http.Redirect(w, r, viewerURL, http.StatusSeeOther)
}

func (ws *WebServer) writeUploadError(w http.ResponseWriter, wantsJSON bool, status int, message string) {
	if wantsJSON {
		ws.writeJSON(w, status, WebUploadResponse{
			Success: false,
			Error:   message,
		})
		return
	}

	http.Error(w, message, status)
}

func (ws *WebServer) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("JSON response error: %v", err)
	}
}

func (ws *WebServer) handleBlob(w http.ResponseWriter, r *http.Request) {
	blobID := strings.TrimPrefix(r.URL.Path, "/blob/")
	if blobID == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := ws.fetchBlobData(blobID)

	// Convert to the template's BlobData struct
	blobData := templates.BlobData{
		BlobID:      data.BlobID,
		Content:     data.Content,
		RawContent:  data.RawContent,
		Metadata:    data.Metadata,
		IsOxBinFile: data.IsOxBinFile,
		IsBinary:    data.IsBinary,
		IsMedia:     isMediaType(data.ContentType),
		Language:    detectLanguage(data.ContentType, data.Metadata),
		Error:       data.Error,
		ContentType: data.ContentType,
		Size:        data.Size,
	}

	// Render the blob page using Templ
	component := templates.Blob(blobData)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (ws *WebServer) handleAPIBlob(w http.ResponseWriter, r *http.Request) {
	blobID := strings.TrimPrefix(r.URL.Path, "/api/blob/")
	if blobID == "" {
		http.Error(w, "Blob ID required", http.StatusBadRequest)
		return
	}

	data := ws.fetchBlobData(blobID)

	if data.Error != "" {
		http.Error(w, data.Error, http.StatusInternalServerError)
		return
	}

	// Always serve the raw content directly with appropriate content type
	w.Header().Set("Content-Type", data.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data.RawContent)))

	// Add filename header if available from OxBin metadata
	if data.IsOxBinFile && data.Metadata != nil && data.Metadata.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", data.Metadata.Filename))
	}

	w.Write(data.RawContent)
}

func (ws *WebServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	blobID := strings.TrimPrefix(r.URL.Path, "/download/")
	if blobID == "" {
		http.Error(w, "Blob ID required", http.StatusBadRequest)
		return
	}

	data := ws.fetchBlobData(blobID)
	if data.Error != "" {
		http.Error(w, data.Error, http.StatusInternalServerError)
		return
	}

	// Set appropriate headers for download
	filename := "blob_" + blobID
	if data.IsOxBinFile && data.Metadata != nil && data.Metadata.Filename != "" {
		filename = data.Metadata.Filename
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data.RawContent)))

	w.Write(data.RawContent)
}

func (ws *WebServer) fetchBlobData(blobID string) PageData {
	data := PageData{
		Title:  "OxBin Web UI - Blob Viewer",
		BlobID: blobID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use the service to read the file
	response, err := ws.service.ReadFile(ctx, blobID)
	if err != nil {
		data.Error = fmt.Sprintf("Failed to fetch blob: %v", err)
		return data
	}

	if !response.Success {
		data.Error = response.Error
		return data
	}

	data.RawContent = response.RawContent
	data.Metadata = response.FileMetadata
	data.IsOxBinFile = response.IsOxBinFile
	data.IsBinary = utils.IsBinaryContent(response.RawContent)
	data.Size = int64(len(response.RawContent))

	// Determine content type
	if data.IsOxBinFile && data.Metadata != nil {
		data.ContentType = data.Metadata.ContentType
	} else {
		data.ContentType = "application/octet-stream"
	}

	// Process content for display
	if data.IsBinary {
		data.Content = utils.GenerateBinaryPreview(response.RawContent, response.FileMetadata)
	} else {
		data.Content = string(response.RawContent)
	}

	return data
}

// isMediaType checks if the content type is a media type that can be displayed
func isMediaType(contentType string) bool {
	mediaTypes := []string{
		"image/", "video/", "audio/", "application/pdf",
	}

	for _, mediaType := range mediaTypes {
		if strings.HasPrefix(contentType, mediaType) {
			return true
		}
	}
	return false
}

// detectLanguage detects the programming language based on content type and filename
func detectLanguage(contentType string, metadata *core.FileMetadata) string {
	// First try to detect from content type
	switch contentType {
	case "application/json":
		return "json"
	case "application/xml", "text/xml":
		return "xml"
	case "text/html":
		return "html"
	case "text/css":
		return "css"
	case "text/javascript", "application/javascript":
		return "javascript"
	case "application/yaml", "text/yaml":
		return "yaml"
	case "application/toml":
		return "toml"
	}

	// If we have metadata, try to detect from filename extension
	if metadata != nil && metadata.Filename != "" {
		ext := strings.ToLower(metadata.Extension)
		switch ext {
		case ".go":
			return "go"
		case ".js", ".mjs":
			return "javascript"
		case ".ts":
			return "typescript"
		case ".py":
			return "python"
		case ".rs":
			return "rust"
		case ".java":
			return "java"
		case ".c":
			return "c"
		case ".cpp", ".cc", ".cxx":
			return "cpp"
		case ".h", ".hpp":
			return "c"
		case ".cs":
			return "csharp"
		case ".php":
			return "php"
		case ".rb":
			return "ruby"
		case ".swift":
			return "swift"
		case ".kt":
			return "kotlin"
		case ".scala":
			return "scala"
		case ".sh", ".bash":
			return "bash"
		case ".ps1":
			return "powershell"
		case ".sql":
			return "sql"
		case ".html", ".htm":
			return "html"
		case ".css":
			return "css"
		case ".scss":
			return "scss"
		case ".less":
			return "less"
		case ".json":
			return "json"
		case ".xml":
			return "xml"
		case ".yaml", ".yml":
			return "yaml"
		case ".toml":
			return "toml"
		case ".ini":
			return "ini"
		case ".cfg", ".conf":
			return "ini"
		case ".md", ".markdown":
			return "markdown"
		case ".tex":
			return "latex"
		case ".r":
			return "r"
		case ".m":
			return "matlab"
		case ".pl":
			return "perl"
		case ".lua":
			return "lua"
		case ".vim":
			return "vim"
		case ".dockerfile":
			return "dockerfile"
		case ".makefile":
			return "makefile"
		}
	}

	// Default to plain text for text content
	if strings.HasPrefix(contentType, "text/") {
		return "text"
	}

	return ""
}
