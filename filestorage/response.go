package filestorage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Luis97lol/microservices-sdk/core/requestbuilder"
)

func sendJSON(builder *requestbuilder.RequestBuilder, operation string, expectedStatus int, output any) error {
	response, err := builder.Send()
	if err != nil {
		return &APIError{Operation: operation, Err: err}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("read response: %w", err),
		}
	}
	if response.StatusCode != expectedStatus {
		return newAPIError(operation, response.StatusCode, body)
	}
	if output == nil {
		return nil
	}
	if err := json.Unmarshal(body, output); err != nil {
		return &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("decode response: %w", err),
		}
	}
	return nil
}

func sendDownload(builder *requestbuilder.RequestBuilder) (DownloadResponse, error) {
	response, err := builder.Send()
	if err != nil {
		return DownloadResponse{}, &APIError{Operation: "download", Err: err}
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return DownloadResponse{}, &APIError{
				Operation:  "download",
				StatusCode: response.StatusCode,
				Err:        fmt.Errorf("read error response: %w", err),
			}
		}
		return DownloadResponse{}, newAPIError("download", response.StatusCode, body)
	}
	return DownloadResponse{
		Body:            response.Body,
		ContentType:     response.Header.Get("Content-Type"),
		ContentEncoding: response.Header.Get("Content-Encoding"),
	}, nil
}

func newAPIError(operation string, statusCode int, body []byte) *APIError {
	var errorResponse ErrorResponse
	decodeErr := json.Unmarshal(body, &errorResponse)
	message := errorResponse.Error
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	apiErr := &APIError{
		Operation:  operation,
		StatusCode: statusCode,
		Message:    message,
	}
	if decodeErr != nil && len(body) > 0 {
		apiErr.Err = fmt.Errorf("decode error response: %w", decodeErr)
	}
	return apiErr
}
