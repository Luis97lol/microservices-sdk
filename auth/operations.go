package auth

import (
	"context"
	"net/http"
)

func (c *Client) Login(ctx context.Context, credentials CredentialsRequest) (TokenResponse, error) {
	var response TokenResponse
	builder := c.requestBuilder(ctx, http.MethodPost, "/login", "").
		WithHeader("Content-Type", "application/json").
		WithJSON(credentials)
	err := sendAndDecodeJSON(builder, "login", &response)
	return response, err
}

func (c *Client) Register(ctx context.Context, credentials CredentialsRequest) (RegisterResponse, error) {
	var response RegisterResponse
	builder := c.requestBuilder(ctx, http.MethodPost, "/register", "").
		WithHeader("Content-Type", "application/json").
		WithJSON(credentials)
	err := sendAndDecodeJSON(builder, "register", &response)
	return response, err
}

func (c *Client) Validate(ctx context.Context, token string) (ValidateResponse, error) {
	var response ValidateResponse
	err := sendAndDecodeJSON(c.requestBuilder(ctx, http.MethodGet, "/validate", token), "validate", &response)
	return response, err
}

func (c *Client) Renew(ctx context.Context, token string) (TokenResponse, error) {
	var response TokenResponse
	err := sendAndDecodeJSON(c.requestBuilder(ctx, http.MethodGet, "/renew", token), "renew", &response)
	return response, err
}

func (c *Client) Logout(ctx context.Context, token string) (MessageResponse, error) {
	var response MessageResponse
	err := sendAndDecodeJSON(c.requestBuilder(ctx, http.MethodGet, "/logout", token), "logout", &response)
	return response, err
}

func (c *Client) DeleteUser(ctx context.Context, token string, request DeleteUserRequest) (MessageResponse, error) {
	var response MessageResponse
	builder := c.requestBuilder(ctx, http.MethodDelete, "/user", token).
		WithHeader("Content-Type", "application/json").
		WithJSON(request)
	err := sendAndDecodeJSON(builder, "delete user", &response)
	return response, err
}
