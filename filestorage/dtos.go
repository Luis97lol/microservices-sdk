package filestorage

import "io"

// UploadRequest describes a file upload. Content is streamed as multipart data.
type UploadRequest struct {
	Module   string
	FileID   string
	Filename string
	Content  io.Reader
}

type UploadResponse struct {
	Module string `json:"module"`
	FileID string `json:"fileId"`
}

type DownloadResponse struct {
	Body            io.ReadCloser
	ContentType     string
	ContentEncoding string
}

type ErrorResponse struct {
	Error string `json:"error"`
}
