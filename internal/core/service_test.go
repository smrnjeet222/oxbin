package core

import (
	"context"
	"testing"
)

func TestUploadBytesRejectsOversizedContentBeforeWalrus(t *testing.T) {
	service := &Service{
		config: &Config{
			Walrus: WalrusConfig{
				MaxFileSize: 3,
			},
		},
	}

	response, err := service.UploadBytes(context.Background(), "notes.txt", []byte("tiny"))
	if err != nil {
		t.Fatalf("UploadBytes returned unexpected error: %v", err)
	}

	if response == nil || response.Success {
		t.Fatalf("expected failed response for oversized file, got %#v", response)
	}

	if response.Error == "" {
		t.Fatal("expected an oversized-file error message")
	}
}

func TestWrapAndUnwrapMetadataPreservesUploadedBytes(t *testing.T) {
	service := &Service{}
	content := []byte("hello from oxbin")
	metadata := service.createFileMetadata("note.md", int64(len(content)), content)

	wrapped, err := service.wrapFileWithMetadata(content, metadata)
	if err != nil {
		t.Fatalf("wrapFileWithMetadata returned error: %v", err)
	}

	unwrapped, fileMetadata, isOxBinFile, err := service.unwrapFileWithMetadata(wrapped)
	if err != nil {
		t.Fatalf("unwrapFileWithMetadata returned error: %v", err)
	}

	if !isOxBinFile {
		t.Fatal("expected wrapped content to be detected as an OxBin file")
	}

	if string(unwrapped) != string(content) {
		t.Fatalf("unexpected unwrapped content: %q", string(unwrapped))
	}

	if fileMetadata == nil {
		t.Fatal("expected metadata to be returned")
	}

	if fileMetadata.Filename != "note.md" {
		t.Fatalf("unexpected filename: %s", fileMetadata.Filename)
	}

	if fileMetadata.ContentType != "text/markdown" {
		t.Fatalf("unexpected content type: %s", fileMetadata.ContentType)
	}
}

func TestSanitizeFilenameHandlesBrowserFakePaths(t *testing.T) {
	got := sanitizeFilename(`C:\fakepath\walrus.png`)
	if got != "walrus.png" {
		t.Fatalf("unexpected sanitized filename: %s", got)
	}
}
