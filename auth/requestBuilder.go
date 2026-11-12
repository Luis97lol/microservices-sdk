package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// RequestBuilder define el struct para construir y enviar requests HTTP
type RequestBuilder struct {
	req    *http.Request
	client *http.Client
	err    error
}

// NewRequestBuilder crea una nueva instancia de RequestBuilder con una request vacía
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

// WithClient establece el cliente HTTP usado para enviar la request.
// Si client es nil, se utiliza http.DefaultClient.
func (rb *RequestBuilder) WithClient(client *http.Client) *RequestBuilder {
	if client == nil {
		rb.client = http.DefaultClient
	} else {
		rb.client = client
	}
	return rb
}

// WithContext establece el contexto de la request.
func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	if ctx == nil {
		rb.err = errors.New("request context is nil")
		return rb
	}
	rb.req = rb.req.WithContext(ctx)
	return rb
}

// From clona una request existente, copiando sus encabezados y configuraciones
func (rb *RequestBuilder) From(r *http.Request) *RequestBuilder {
	if r == nil {
		rb.err = errors.New("request is nil")
		return rb
	}
	rb.req = r.Clone(r.Context())    // Clona la request
	rb.req.Header = r.Header.Clone() // Clona los encabezados
	return rb
}

// WithHeader añade o sobrescribe un encabezado en la request
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.req.Header.Set(key, value) // Añade o sobrescribe el encabezado
	return rb
}

// WithMethod establece el método HTTP de la request (GET, POST, PUT, etc.)
func (rb *RequestBuilder) WithMethod(method string) *RequestBuilder {
	rb.req.Method = method // Cambia el método de la request
	return rb
}

// WithBody establece el cuerpo de la request
func (rb *RequestBuilder) WithBody(body []byte) *RequestBuilder {
	bodyCopy := append([]byte(nil), body...)
	rb.req.Body = io.NopCloser(bytes.NewReader(bodyCopy))
	rb.req.ContentLength = int64(len(bodyCopy))
	rb.req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyCopy)), nil
	}
	return rb
}

func (rb *RequestBuilder) WithJSON(jsonBody interface{}) *RequestBuilder {
	body, err := json.Marshal(jsonBody)
	if err != nil {
		rb.err = fmt.Errorf("encode request body: %w", err)
		return rb
	}
	return rb.WithBody(body)
}

// WithURL crea una nueva request con la URL proporcionada, manteniendo el método, encabezados y cuerpo
func (rb *RequestBuilder) WithURL(url string) *RequestBuilder {
	newRequest, err := http.NewRequestWithContext(rb.req.Context(), rb.req.Method, url, rb.req.Body)
	if err != nil {
		rb.err = fmt.Errorf("set request URL: %w", err)
		return rb
	}

	newRequest.Header = rb.req.Header.Clone()
	rb.req = newRequest
	return rb
}

// Send envía la request construida y devuelve la respuesta o un error
func (rb *RequestBuilder) Send() (*http.Response, error) {
	if rb.err != nil {
		return nil, fmt.Errorf("build request: %w", rb.err)
	}
	if rb.req == nil {
		return nil, errors.New("build request: request is nil")
	}
	resp, err := rb.client.Do(rb.req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	return resp, nil
}

// Build devuelve la request construida.
func (rb *RequestBuilder) Build() *http.Request {
	return rb.req
}
