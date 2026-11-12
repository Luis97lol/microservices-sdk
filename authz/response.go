package authz

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Luis97lol/microservices-sdk/core/requestbuilder"
)

func sendText(builder *requestbuilder.RequestBuilder, operation string) (OperationResponse, error) {
	response, err := builder.Send()
	if err != nil {
		return OperationResponse{}, &APIError{Operation: operation, Err: err}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return OperationResponse{}, &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Err:        fmt.Errorf("read response: %w", err),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return OperationResponse{}, &APIError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Message:    string(body),
		}
	}

	return OperationResponse{Body: string(body)}, nil
}
