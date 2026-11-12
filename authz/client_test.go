package authz

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
		w.Header().Set("Content-Type", "text/plain")
		switch r.URL.Path {
		case "/add":
			assertRequest(t, r, http.MethodPost, "")
			assertPermissionBody(t, r, true)
		case "/verify":
			assertRequest(t, r, http.MethodPost, "Bearer session-token")
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode verify request: %v", err)
			}
			if request["app_id"] != "app-id" || request["permission_id"] != "permission-id" {
				t.Errorf("unexpected verify request: %+v", request)
			}
			if _, hasCredentialID := request["credential_id"]; hasCredentialID {
				t.Error("verify request must not send credential_id")
			}
		case "/check":
			assertRequest(t, r, http.MethodPost, "")
			assertPermissionBody(t, r, true)
		case "/remove":
			assertRequest(t, r, http.MethodDelete, "")
			assertPermissionBody(t, r, true)
		default:
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	permission := PermissionScoped{
		AppID:        "app-id",
		CredentialID: "credential-id",
		PermissionID: "permission-id",
	}
	for name, operation := range map[string]func() (OperationResponse, error){
		"add": func() (OperationResponse, error) {
			return client.AddPermission(ctx, permission)
		},
		"verify": func() (OperationResponse, error) {
			return client.VerifyPermission(ctx, "session-token", VerifyPermissionRequest{
				AppID: "app-id", PermissionID: "permission-id",
			})
		},
		"check": func() (OperationResponse, error) {
			return client.CheckPermission(ctx, permission)
		},
		"remove": func() (OperationResponse, error) {
			return client.RemovePermission(ctx, permission)
		},
	} {
		response, err := operation()
		if err != nil || response.Body != "OK" {
			t.Errorf("%s() = %+v, %v; want body OK", name, response, err)
		}
	}
}

func TestClientErrorsPreserveCauseAndPlainTextDetails(t *testing.T) {
	originalErr := errors.New("connection refused")
	client, err := NewClient(Config{
		BaseURL: "http://authz.example",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, originalErr
		})},
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.CheckPermission(context.Background(), PermissionScoped{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, originalErr) {
		t.Fatalf("CheckPermission() error = %v; want APIError wrapping original cause", err)
	}
	if apiErr.Operation != "check permission" {
		t.Errorf("APIError.Operation = %q, want check permission", apiErr.Operation)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "Unauthorized")
	}))
	defer server.Close()
	httpClient, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = httpClient.VerifyPermission(context.Background(), "bad-token", VerifyPermissionRequest{})
	if !errors.As(err, &apiErr) {
		t.Fatalf("VerifyPermission() error = %v; want APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "Unauthorized" {
		t.Errorf("APIError = %+v; want HTTP 401 Unauthorized", apiErr)
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
		t.Errorf("%s: method = %s, want %s", request.URL.Path, request.Method, method)
	}
	if got := request.Header.Get("Authorization"); got != authorization {
		t.Errorf("%s: Authorization = %q, want %q", request.URL.Path, got, authorization)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("%s: Content-Type = %q, want application/json", request.URL.Path, got)
	}
}

func assertPermissionBody(t *testing.T, request *http.Request, includeCredentialID bool) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Errorf("decode permission request: %v", err)
		return
	}
	if body["app_id"] != "app-id" || body["permission_id"] != "permission-id" {
		t.Errorf("unexpected permission body: %+v", body)
	}
	if includeCredentialID && body["credential_id"] != "credential-id" {
		t.Errorf("credential_id = %q, want credential-id", body["credential_id"])
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
