package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetGitHubOIDCToken(t *testing.T) {
	// Test successful token request
	t.Run("successful token request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the request
			if r.Header.Get("Authorization") != "bearer test-token" {
				t.Errorf("Expected Authorization header 'bearer test-token', got '%s'", r.Header.Get("Authorization"))
			}

			if !r.URL.Query().Has("audience") {
				t.Error("Expected audience query parameter")
			}

			// Return mock OIDC token
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"value": "mock-oidc-token"}`))
		}))
		defer server.Close()

		token, err := getGitHubOIDCToken(server.URL, "test-token", "api://AzureADTokenExchange")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if token != "mock-oidc-token" {
			t.Errorf("Expected token 'mock-oidc-token', got '%s'", token)
		}
	})

	// Test error response
	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
		}))
		defer server.Close()

		_, err := getGitHubOIDCToken(server.URL, "invalid-token", "api://AzureADTokenExchange")
		if err == nil {
			t.Error("Expected error for unauthorized response")
		}
	})

	// Test empty token response
	t.Run("empty token response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"value": ""}`))
		}))
		defer server.Close()

		_, err := getGitHubOIDCToken(server.URL, "test-token", "api://AzureADTokenExchange")
		if err == nil {
			t.Error("Expected error for empty token response")
		}
	})
}
