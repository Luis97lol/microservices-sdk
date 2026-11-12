package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type RequestBuilder struct {
	req    *http.Request
	client *http.Client
	err    error
}

func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		req: &http.Request{
			Method: http.MethodGet,
			Header: make(http.Header),
			Body:   http.NoBody,
		},
		client: http.DefaultClient,
	}
}

func (rb *RequestBuilder) WithClient(client *http.Client) *RequestBuilder {
	if client == nil {
		rb.client = http.DefaultClient
	} else {
		rb.client = client
	}
	return rb
}

func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	if ctx == nil {
		rb.err = errors.New("request context is nil")
		return rb
	}
	rb.req = rb.req.WithContext(ctx)
	return rb
}

func (rb *RequestBuilder) From(request *http.Request) *RequestBuilder {
	if request == nil {
		rb.err = errors.New("request is nil")
		return rb
	}
	rb.req = request.Clone(request.Context())
	rb.req.Header = request.Header.Clone()
	return rb
}

func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.req.Header.Set(key, value)
	return rb
}

func (rb *RequestBuilder) WithMethod(method string) *RequestBuilder {
	rb.req.Method = method
	return rb
}

func (rb *RequestBuilder) WithBody(body []byte) *RequestBuilder {
	bodyCopy := append([]byte(nil), body...)
	rb.req.Body = io.NopCloser(bytes.NewReader(bodyCopy))
	rb.req.ContentLength = int64(len(bodyCopy))
	rb.req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyCopy)), nil
	}
	return rb
}

func (rb *RequestBuilder) WithJSON(body any) *RequestBuilder {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		rb.err = fmt.Errorf("encode request body: %w", err)
		return rb
	}
	return rb.WithBody(encodedBody)
}

func (rb *RequestBuilder) WithURL(rawURL string) *RequestBuilder {
	request, err := http.NewRequestWithContext(rb.req.Context(), rb.req.Method, rawURL, rb.req.Body)
	if err != nil {
		rb.err = fmt.Errorf("set request URL: %w", err)
		return rb
	}
	request.Header = rb.req.Header.Clone()
	rb.req = request
	return rb
}

func (rb *RequestBuilder) Send() (*http.Response, error) {
	if rb.err != nil {
		return nil, fmt.Errorf("build request: %w", rb.err)
	}
	if rb.req == nil {
		return nil, errors.New("build request: request is nil")
	}
	response, err := rb.client.Do(rb.req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	return response, nil
}

func (rb *RequestBuilder) Build() *http.Request {
	return rb.req
}
