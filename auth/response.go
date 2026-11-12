package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Luis97lol/microservices-sdk/core/requestbuilder"
)

func sendAndDecodeJSON(builder *requestbuilder.RequestBuilder, operation string, output any) error {
	response, err := builder.Send()
	if err != nil {
		return &APIError{Operation: operation, Err: err}
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("read response: %w", err),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var errorResponse ErrorResponse
		decodeErr := json.Unmarshal(responseBody, &errorResponse)
		message := errorResponse.Error
		if message == "" {
			message = strings.TrimSpace(string(responseBody))
		}
		apiErr := &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Message:    message,
		}
		if decodeErr != nil && len(responseBody) > 0 {
			apiErr.Err = fmt.Errorf("decode error response: %w", decodeErr)
		}
		return apiErr
	}

	if output == nil {
		return nil
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("decode response: %w", err),
		}
	}
	return nil
}
