package filestorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/Luis97lol/microservices-sdk/core/requestbuilder"
)

func (c *Client) Upload(ctx context.Context, token string, request UploadRequest) (UploadResponse, error) {
	if request.Filename == "" {
		return UploadResponse{}, &APIError{Operation: "upload", Err: errors.New("filename is required")}
	}
	if request.Content == nil {
		return UploadResponse{}, &APIError{Operation: "upload", Err: errors.New("file content is required")}
	}

	path := fileRoute(request.Module, "upload", request.FileID)
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	writeErr := make(chan error, 1)
	go func() {
		part, err := form.CreateFormFile("file", request.Filename)
		if err == nil {
			_, err = io.Copy(part, request.Content)
		}
		if closeErr := form.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = writer.CloseWithError(err)
		} else {
			_ = writer.Close()
		}
		writeErr <- err
	}()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reader)
	if err != nil {
		_ = reader.CloseWithError(err)
		<-writeErr
		return UploadResponse{}, &APIError{Operation: "upload", Err: fmt.Errorf("create request: %w", err)}
	}
	httpRequest.Header.Set("Content-Type", form.FormDataContentType())
	builder := requestbuilder.NewRequestBuilder().From(httpRequest)
	if c.httpClient != nil {
		builder.WithClient(c.httpClient)
	}
	builder.WithHeader("Authorization", "Bearer "+token)
	builder.WithHeader("Accept", "application/json")

	response, sendErr := builder.Send()
	if sendErr != nil {
		_ = reader.CloseWithError(sendErr)
		<-writeErr
		return UploadResponse{}, &APIError{Operation: "upload", Err: sendErr}
	}
	defer response.Body.Close()
	if err := <-writeErr; err != nil {
		return UploadResponse{}, &APIError{
			Operation: "upload",
			Err:       fmt.Errorf("write multipart request: %w", err),
		}
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return UploadResponse{}, &APIError{
			Operation:  "upload",
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("read response: %w", err),
		}
	}
	if response.StatusCode != http.StatusOK {
		return UploadResponse{}, newAPIError("upload", response.StatusCode, body)
	}

	var result UploadResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return UploadResponse{}, &APIError{
			Operation:  "upload",
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("decode response: %w", err),
		}
	}
	return result, nil
}

func (c *Client) Download(ctx context.Context, token, module, fileID string) (DownloadResponse, error) {
	if fileID == "" {
		return DownloadResponse{}, &APIError{Operation: "download", Err: errors.New("file ID is required")}
	}
	path := fileRoute(module, "download", fileID)
	return sendDownload(c.requestBuilder(ctx, http.MethodGet, path, token, "application/octet-stream"))
}

func (c *Client) DeleteFile(ctx context.Context, token, module, fileID string) error {
	if fileID == "" {
		return &APIError{Operation: "delete", Err: errors.New("file ID is required")}
	}
	path := fileRoute(module, "delete", fileID)
	return sendJSON(c.requestBuilder(ctx, http.MethodDelete, path, token, "application/json"), "delete", http.StatusNoContent, nil)
}
