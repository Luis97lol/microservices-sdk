package filestorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer session-token" {
			t.Errorf("Authorization = %q, want Bearer session-token", got)
		}

		switch r.URL.Path {
		case "/reports/upload/fixed-id":
			if r.Method != http.MethodPost {
				t.Errorf("upload method = %s, want POST", r.Method)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Errorf("read multipart file: %v", err)
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("read uploaded content: %v", err)
			}
			if header.Filename != "report.txt" || string(content) != "report content" {
				t.Errorf("uploaded file = %q (%q)", header.Filename, content)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"module":"reports","fileId":"fixed-id"}`)
		case "/reports/download/fixed-id":
			if r.Method != http.MethodGet {
				t.Errorf("download method = %s, want GET", r.Method)
			}
			if got := r.Header.Get("Accept"); got != "application/octet-stream" {
				t.Errorf("download Accept = %q", got)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte{0x00, 0x01, 0xff})
		case "/reports/delete/fixed-id":
			if r.Method != http.MethodDelete {
				t.Errorf("delete method = %s, want DELETE", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	uploaded, err := client.Upload(context.Background(), "session-token", UploadRequest{
		Module:   "reports",
		FileID:   "fixed-id",
		Filename: "report.txt",
		Content:  bytes.NewBufferString("report content"),
	})
	if err != nil || uploaded.Module != "reports" || uploaded.FileID != "fixed-id" {
		t.Fatalf("Upload() = %+v, %v", uploaded, err)
	}

	downloaded, err := client.Download(context.Background(), "session-token", "reports", "fixed-id")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer downloaded.Body.Close()
	body, err := io.ReadAll(downloaded.Body)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if !bytes.Equal(body, []byte{0x00, 0x01, 0xff}) ||
		downloaded.ContentType != "application/octet-stream" ||
		downloaded.ContentEncoding != "" {
		t.Errorf("Download() = body %v, content type %q, encoding %q",
			body, downloaded.ContentType, downloaded.ContentEncoding)
	}

	if err := client.DeleteFile(context.Background(), "session-token", "reports", "fixed-id"); err != nil {
		t.Errorf("DeleteFile() error = %v", err)
	}
}

func TestClientErrorsPreserveCauseAndHTTPDetails(t *testing.T) {
	originalErr := errors.New("connection refused")
	client, err := NewClient(Config{
		BaseURL: "http://filestorage.example",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, originalErr
		})},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Download(context.Background(), "token", "", "file-id")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, originalErr) {
		t.Fatalf("Download() error = %v; want APIError wrapping original cause", err)
	}
	if apiErr.Operation != "download" {
		t.Errorf("APIError.Operation = %q, want download", apiErr.Operation)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"Unauthorized"}`)
	}))
	defer server.Close()
	httpClient, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = httpClient.Download(context.Background(), "bad-token", "", "file-id")
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "Unauthorized" {
		t.Errorf("Download() error = %#v; want APIError HTTP 401 Unauthorized", err)
	}
}

func TestClientRejectsMissingInputs(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "http://filestorage.example"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.Upload(context.Background(), "token", UploadRequest{Content: bytes.NewBuffer(nil)})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Operation != "upload" {
		t.Errorf("Upload() error = %v; want upload APIError for missing filename", err)
	}
	if err := client.DeleteFile(context.Background(), "token", "", ""); !errors.As(err, &apiErr) {
		t.Errorf("DeleteFile() error = %v; want APIError for missing file ID", err)
	}
}

func TestUploadRequestContentTypeIsMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("parse Content-Type: %v", err)
			return
		}
		if mediaType != "multipart/form-data" || params["boundary"] == "" {
			t.Errorf("Content-Type = %q, want multipart/form-data with boundary", r.Header.Get("Content-Type"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("read multipart file: %v", err)
			return
		}
		_ = file.Close()
		_, _ = fmt.Fprint(w, `{"module":"","fileId":"id"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.Upload(context.Background(), "token", UploadRequest{
		Filename: "file.bin",
		Content:  bytes.NewReader(nil),
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
