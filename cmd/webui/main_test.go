package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smrnjeet222/oxbin/internal/core"
)

type fakeBlobService struct {
	maxFileSize int64
	filename    string
	content     []byte
	uploadCalls int
}

func (f *fakeBlobService) ReadFile(ctx context.Context, blobID string) (*core.ReadResponse, error) {
	return nil, nil
}

func (f *fakeBlobService) UploadBytes(ctx context.Context, filename string, content []byte) (*core.UploadResponse, error) {
	f.uploadCalls++
	f.filename = filename
	f.content = append([]byte(nil), content...)
	return &core.UploadResponse{
		Success:       true,
		BlobID:        "blob123",
		PublicURL:     "https://aggregator.test/v1/blobs/blob123",
		WalrusScanURL: "https://walruscan.com/testnet/blob/blob123",
	}, nil
}

func (f *fakeBlobService) MaxFileSize() int64 {
	return f.maxFileSize
}

func TestHandleUploadReturnsJSONForWebUpload(t *testing.T) {
	service := &fakeBlobService{maxFileSize: 1024}
	server := &WebServer{service: service}
	body, contentType := multipartBody(t, "file", "cute.txt", []byte("hello walrus"))

	request := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()

	server.handleUpload(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload WebUploadResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if !payload.Success || payload.BlobID != "blob123" || payload.ViewerURL != "/blob/blob123" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	if payload.Filename != "cute.txt" {
		t.Fatalf("unexpected payload filename: %s", payload.Filename)
	}

	if payload.Size != int64(len("hello walrus")) {
		t.Fatalf("unexpected payload size: %d", payload.Size)
	}

	if payload.ContentType == "" {
		t.Fatal("expected content type for history metadata")
	}

	if payload.UploadedAt == "" {
		t.Fatal("expected upload timestamp for history metadata")
	}

	if service.uploadCalls != 1 {
		t.Fatalf("expected one upload call, got %d", service.uploadCalls)
	}

	if service.filename != "cute.txt" {
		t.Fatalf("unexpected filename: %s", service.filename)
	}

	if string(service.content) != "hello walrus" {
		t.Fatalf("unexpected content: %q", string(service.content))
	}
}

func TestHandleUploadRejectsOversizedFilesBeforeServiceUpload(t *testing.T) {
	service := &fakeBlobService{maxFileSize: 3}
	server := &WebServer{service: service}
	body, contentType := multipartBody(t, "file", "too-big.txt", []byte("large"))

	request := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()

	server.handleUpload(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d: %s", response.Code, response.Body.String())
	}

	if service.uploadCalls != 0 {
		t.Fatalf("expected upload service not to be called, got %d calls", service.uploadCalls)
	}
}

func multipartBody(t *testing.T, fieldName string, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write multipart content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}
