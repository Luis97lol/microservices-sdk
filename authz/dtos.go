package authz

type PermissionScoped struct {
	AppID        string `json:"app_id"`
	CredentialID string `json:"credential_id"`
	PermissionID string `json:"permission_id"`
}

type VerifyPermissionRequest struct {
	AppID        string `json:"app_id"`
	PermissionID string `json:"permission_id"`
}

type OperationResponse struct {
	Body string
}
