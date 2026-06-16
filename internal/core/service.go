package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	walrus "github.com/namihq/walrus-go"
)

// Service provides core business logic for OxBin
type Service struct {
	config       *Config
	walrusClient *walrus.Client
}

// NewService creates a new service instance
func NewService(config *Config) (*Service, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Walrus.HTTPTimeout,
	}

	// Initialize Walrus client
	client := walrus.NewClient(
		walrus.WithPublisherURLs(config.Walrus.PublisherURLs),
		walrus.WithAggregatorURLs(config.Walrus.AggregatorURLs),
		walrus.WithHTTPClient(httpClient),
	)

	return &Service{
		config:       config,
		walrusClient: client,
	}, nil
}

// UploadFile uploads a file to Walrus with metadata
func (s *Service) UploadFile(ctx context.Context, filePath string) (*UploadResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get file info: %v", err),
		}, nil
	}

	if fileInfo.IsDir() {
		return &UploadResponse{
			Success: false,
			Error:   "Cannot upload a directory",
		}, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read file: %v", err),
		}, nil
	}

	return s.uploadContent(ctx, filepath.Base(filePath), content, fileInfo.Size())
}

// UploadBytes uploads in-memory file content to Walrus with metadata.
func (s *Service) UploadBytes(ctx context.Context, filename string, content []byte) (*UploadResponse, error) {
	return s.uploadContent(ctx, filename, content, int64(len(content)))
}

// MaxFileSize returns the configured upload limit in bytes.
func (s *Service) MaxFileSize() int64 {
	return s.config.Walrus.MaxFileSize
}

func (s *Service) uploadContent(ctx context.Context, filename string, content []byte, size int64) (*UploadResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if size > s.config.Walrus.MaxFileSize {
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("File too large. Maximum size is %d bytes", s.config.Walrus.MaxFileSize),
		}, nil
	}

	filename = sanitizeFilename(filename)
	metadata := s.createFileMetadata(filename, size, content)

	// Wrap content with metadata
	wrappedData, err := s.wrapFileWithMetadata(content, metadata)
	if err != nil {
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to wrap file: %v", err),
		}, nil
	}

	select {
	case <-ctx.Done():
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Upload cancelled: %v", ctx.Err()),
		}, nil
	default:
	}

	// Upload to Walrus
	resp, err := s.walrusClient.Store(wrappedData, &walrus.StoreOptions{
		Epochs: 1,
	})
	if err != nil {
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to upload to Walrus: %v", err),
		}, nil
	}

	select {
	case <-ctx.Done():
		return &UploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Upload cancelled: %v", ctx.Err()),
		}, nil
	default:
	}

	if resp == nil {
		return &UploadResponse{
			Success: false,
			Error:   "Upload failed: empty response from Walrus",
		}, nil
	}

	// Extract blob ID from response
	var blobID string
	if resp.NewlyCreated != nil && resp.NewlyCreated.BlobObject.BlobID != "" {
		blobID = resp.NewlyCreated.BlobObject.BlobID
	} else if resp.AlreadyCertified != nil && resp.AlreadyCertified.BlobID != "" {
		blobID = resp.AlreadyCertified.BlobID
	} else {
		return &UploadResponse{
			Success: false,
			Error:   "Upload failed: no blob ID in response",
		}, nil
	}

	// Generate URLs
	publicURL := fmt.Sprintf("%s/v1/blobs/%s", s.config.Walrus.AggregatorURLs[0], blobID)

	// Determine network for WalrusScan
	network := "testnet"
	if strings.Contains(s.config.Walrus.AggregatorURLs[0], "mainnet") {
		network = "mainnet"
	}
	walrusScanURL := fmt.Sprintf("https://walruscan.com/%s/blob/%s", network, blobID)

	return &UploadResponse{
		BlobID:        blobID,
		PublicURL:     publicURL,
		WalrusScanURL: walrusScanURL,
		Success:       true,
	}, nil
}

// ReadFile reads a file from Walrus by blob ID
func (s *Service) ReadFile(ctx context.Context, blobID string) (*ReadResponse, error) {
	// Read content
	rawContent, err := s.walrusClient.Read(blobID, nil)
	if err != nil {
		return &ReadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read file: %v", err),
		}, nil
	}

	// Try to unwrap metadata
	unwrappedContent, fileMetadata, isOxBinFile, err := s.unwrapFileWithMetadata(rawContent)
	if err != nil {
		return &ReadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to unwrap file: %v", err),
		}, nil
	}

	return &ReadResponse{
		RawContent:   unwrappedContent,
		FileMetadata: fileMetadata,
		IsOxBinFile:  isOxBinFile,
		Success:      true,
	}, nil
}

// Helper methods

func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return "oxbin-upload.bin"
	}
	return filename
}

func (s *Service) createFileMetadata(filename string, size int64, content []byte) FileMetadata {
	extension := filepath.Ext(filename)
	contentType := s.getContentType(extension)
	if contentType == "application/octet-stream" && len(content) > 0 {
		contentType = http.DetectContentType(content)
	}

	return FileMetadata{
		Filename:    filename,
		Extension:   extension,
		Size:        size,
		ContentType: contentType,
		UploadTime:  time.Now(),
		Version:     "1.0",
	}
}

func (s *Service) getContentType(extension string) string {
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

func (s *Service) wrapFileWithMetadata(content []byte, metadata FileMetadata) ([]byte, error) {
	// Encode content as base64
	encodedContent := base64.StdEncoding.EncodeToString(content)

	// Create wrapper
	wrapper := BlobWithMetadata{
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

func (s *Service) unwrapFileWithMetadata(data []byte) ([]byte, *FileMetadata, bool, error) {
	// Try to parse as OxBin file with metadata
	var wrapper BlobWithMetadata
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
