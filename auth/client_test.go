package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			assertRequest(t, r, http.MethodPost, "")
			var credentials CredentialsRequest
			if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
				t.Errorf("decode login request: %v", err)
			}
			if credentials.Username != "alice" || credentials.Organization != "org" {
				t.Errorf("unexpected login credentials: %+v", credentials)
			}
			_, _ = fmt.Fprint(w, `{"token":"session","ttl":123}`)
		case "/register":
			assertRequest(t, r, http.MethodPost, "")
			_, _ = fmt.Fprint(w, `{"uuid":"user-id"}`)
		case "/validate":
			assertRequest(t, r, http.MethodGet, "Bearer session")
			_, _ = fmt.Fprint(w, `{"user":"user-id","appId":"app","exp":456}`)
		case "/renew":
			assertRequest(t, r, http.MethodGet, "Bearer session")
			_, _ = fmt.Fprint(w, `{"token":"renewed","ttl":789}`)
		case "/logout":
			assertRequest(t, r, http.MethodGet, "Bearer session")
			_, _ = fmt.Fprint(w, `{"response":"logged out"}`)
		case "/user":
			assertRequest(t, r, http.MethodDelete, "Bearer session")
			var request DeleteUserRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode delete user request: %v", err)
			}
			if request.UserID != "user-id" {
				t.Errorf("unexpected user ID: %q", request.UserID)
			}
			_, _ = fmt.Fprint(w, `{"response":"accepted"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	credentials := CredentialsRequest{Organization: "org", Username: "alice", Password: "secret"}
	if got, err := client.Login(ctx, credentials); err != nil || got.Token != "session" || got.TTL != 123 {
		t.Errorf("Login() = %+v, %v", got, err)
	}
	if got, err := client.Register(ctx, credentials); err != nil || got.UUID != "user-id" {
		t.Errorf("Register() = %+v, %v", got, err)
	}
	if got, err := client.Validate(ctx, "session"); err != nil || got.User != "user-id" || got.AppID != "app" || got.ExpiresAt != 456 {
		t.Errorf("Validate() = %+v, %v", got, err)
	}
	if got, err := client.Renew(ctx, "session"); err != nil || got.Token != "renewed" || got.TTL != 789 {
		t.Errorf("Renew() = %+v, %v", got, err)
	}
	if got, err := client.Logout(ctx, "session"); err != nil || got.Response != "logged out" {
		t.Errorf("Logout() = %+v, %v", got, err)
	}
	if got, err := client.DeleteUser(ctx, "session", DeleteUserRequest{UserID: "user-id"}); err != nil || got.Response != "accepted" {
		t.Errorf("DeleteUser() = %+v, %v", got, err)
	}
}

func TestClientErrorsPreserveCauseAndHTTPDetails(t *testing.T) {
	originalErr := errors.New("connection refused")
	client, err := NewClient(Config{
		BaseURL: "http://auth.example",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, originalErr
		})},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Login(context.Background(), CredentialsRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, originalErr) {
		t.Fatalf("Login() error = %v; want APIError wrapping original cause", err)
	}
	if apiErr.Operation != "login" {
		t.Errorf("APIError.Operation = %q, want login", apiErr.Operation)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"Invalid token"}`)
	}))
	defer server.Close()
	httpClient, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = httpClient.Validate(context.Background(), "bad-token")
	if !errors.As(err, &apiErr) {
		t.Fatalf("Validate() error = %v; want APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "Invalid token" {
		t.Errorf("APIError = %+v; want HTTP 401 Invalid token", apiErr)
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "not a URL"})
	var configErr *ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("NewClient() error = %v; want ConfigError", err)
	}
}

func assertRequest(t *testing.T, request *http.Request, method, authorization string) {
	t.Helper()
	if request.Method != method {
		t.Errorf("%s %s: method = %s, want %s", request.Method, request.URL.Path, request.Method, method)
	}
	if got := request.Header.Get("Authorization"); got != authorization {
		t.Errorf("%s: Authorization = %q, want %q", request.URL.Path, got, authorization)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
