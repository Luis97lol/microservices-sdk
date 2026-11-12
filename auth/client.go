package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Config contains the auth service settings. Values may be supplied from
// environment variables by the application that constructs the client.
type Config struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Client calls the auth service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an auth client from explicit configuration.
func NewClient(config Config) (*Client, error) {
	rawBaseURL := strings.TrimSpace(config.BaseURL)
	if rawBaseURL == "" {
		return nil, &ConfigError{Field: "BaseURL", Err: errBaseURLRequired}
	}

	parsedURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, &ConfigError{Field: "BaseURL", Err: err}
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" ||
		parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, &ConfigError{
			Field: "BaseURL",
			Err:   fmt.Errorf("must be an absolute http(s) URL without query or fragment"),
		}
	}

	return &Client{
		baseURL:    strings.TrimRight(parsedURL.String(), "/"),
		httpClient: config.HTTPClient,
	}, nil
}

func (c *Client) requestBuilder(ctx context.Context, method, path, token string) *RequestBuilder {
	builder := NewRequestBuilder().
		WithContext(ctx).
		WithMethod(method).
		WithURL(c.baseURL+path).
		WithHeader("Accept", "application/json")
	if token != "" {
		builder.WithHeader("Authorization", "Bearer "+token)
	}
	if c.httpClient != nil {
		builder.WithClient(c.httpClient)
	}
	return builder
}
