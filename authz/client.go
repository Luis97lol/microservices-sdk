package authz

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(config Config) (*Client, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, &ConfigError{Field: "BaseURL", Err: errBaseURLRequired}
	}

	parsedURL, err := url.Parse(baseURL)
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
		WithHeader("Accept", "text/plain")
	if token != "" {
		builder.WithHeader("Authorization", "Bearer "+token)
	}
	if c.httpClient != nil {
		builder.WithClient(c.httpClient)
	}
	return builder
}
