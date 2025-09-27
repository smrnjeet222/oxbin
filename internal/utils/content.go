package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jeet/oxbin/internal/core"
)

// IsBinaryContent checks if content is binary (contains non-printable characters)
func IsBinaryContent(data []byte) bool {
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

// IsValidUTF8 checks if content is valid UTF-8 text
func IsValidUTF8(data []byte) bool {
	return utf8.Valid(data)
}

// ProcessContentForDisplay processes content for display in UI
func ProcessContentForDisplay(rawContent []byte, fileMetadata *core.FileMetadata) string {
	// First check if it's binary content
	if IsBinaryContent(rawContent) {
		return GenerateBinaryPreview(rawContent, fileMetadata)
	}

	// Convert to string and check if it's valid UTF-8
	content := string(rawContent)
	if !IsValidUTF8(rawContent) {
		return GenerateBinaryPreview(rawContent, fileMetadata)
	}

	// For text content, return full content - viewport will handle scrolling
	return content
}

// GenerateBinaryPreview generates a preview for binary files
func GenerateBinaryPreview(data []byte, fileMetadata *core.FileMetadata) string {
	var preview strings.Builder

	// File type information
	if fileMetadata != nil {
		preview.WriteString(fmt.Sprintf("📄 Binary File: %s\n", fileMetadata.ContentType))
		preview.WriteString(fmt.Sprintf("📏 Size: %s\n\n", FormatFileSize(fileMetadata.Size)))
	} else {
		preview.WriteString("📄 Binary File\n")
		preview.WriteString(fmt.Sprintf("📏 Size: %s\n\n", FormatFileSize(int64(len(data)))))
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

// UnwrapFileWithMetadata unwraps file content and metadata
func UnwrapFileWithMetadata(data []byte) ([]byte, *core.FileMetadata, bool, error) {
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

// WrapFileWithMetadata wraps file content with metadata
func WrapFileWithMetadata(filePath string, content []byte, fileInfo interface{}) ([]byte, error) {
	// This would need to be implemented based on the file info structure
	// For now, return a placeholder implementation
	return content, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
