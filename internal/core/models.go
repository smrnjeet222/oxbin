package core

import "time"

// FileMetadata represents metadata for uploaded files
type FileMetadata struct {
	Filename    string    `json:"filename"`
	Extension   string    `json:"extension"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	UploadTime  time.Time `json:"uploadTime"`
	Version     string    `json:"version"`
}

// BlobWithMetadata wraps file content with metadata
type BlobWithMetadata struct {
	Metadata FileMetadata `json:"metadata"`
	Content  string       `json:"content"` // Base64 encoded content
	Type     string       `json:"type"`    // Always "oxbin-file" for identification
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
}

// UploadResponse represents a file upload response
type UploadResponse struct {
	BlobID        string `json:"blob_id"`
	PublicURL     string `json:"public_url"`
	WalrusScanURL string `json:"walrus_scan_url"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

// ReadRequest represents a file read request
type ReadRequest struct {
	BlobID string `json:"blob_id"`
}

// ReadResponse represents a file read response
type ReadResponse struct {
	Content      string        `json:"content,omitempty"`
	RawContent   []byte        `json:"raw_content,omitempty"`
	FileMetadata *FileMetadata `json:"file_metadata,omitempty"`
	IsOxBinFile  bool          `json:"is_oxbin_file"`
	IsBinary     bool          `json:"is_binary"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
}

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}
