package auth

type CredentialsRequest struct {
	Organization string `json:"org_id"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
	TTL   int64  `json:"ttl"`
}

type RegisterResponse struct {
	UUID string `json:"uuid"`
}

type ValidateResponse struct {
	User      string `json:"user"`
	AppID     string `json:"appId"`
	ExpiresAt int    `json:"exp"`
}

type MessageResponse struct {
	Response string `json:"response"`
}

type DeleteUserRequest struct {
	UserID string `json:"userId"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
