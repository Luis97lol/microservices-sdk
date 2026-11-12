package authz

import (
	"context"
	"net/http"
)

func (c *Client) AddPermission(ctx context.Context, permission PermissionScoped) (OperationResponse, error) {
	builder := c.requestBuilder(ctx, http.MethodPost, "/add", "").
		WithHeader("Content-Type", "application/json").
		WithJSON(permission)
	return sendText(builder, "add permission")
}

func (c *Client) VerifyPermission(ctx context.Context, token string, permission VerifyPermissionRequest) (OperationResponse, error) {
	builder := c.requestBuilder(ctx, http.MethodPost, "/verify", token).
		WithHeader("Content-Type", "application/json").
		WithJSON(permission)
	return sendText(builder, "verify permission")
}

func (c *Client) CheckPermission(ctx context.Context, permission PermissionScoped) (OperationResponse, error) {
	builder := c.requestBuilder(ctx, http.MethodPost, "/check", "").
		WithHeader("Content-Type", "application/json").
		WithJSON(permission)
	return sendText(builder, "check permission")
}

func (c *Client) RemovePermission(ctx context.Context, permission PermissionScoped) (OperationResponse, error) {
	builder := c.requestBuilder(ctx, http.MethodDelete, "/remove", "").
		WithHeader("Content-Type", "application/json").
		WithJSON(permission)
	return sendText(builder, "remove permission")
}
