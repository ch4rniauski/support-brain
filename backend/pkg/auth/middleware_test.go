package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func decodeErrorBody(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body["error"]
}

func TestMiddleware_Authenticate(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-secret", time.Hour)
	mw := NewMiddleware(manager)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			http.Error(w, "missing claims", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"role": string(claims.Role)})
	})

	handler := mw.Authenticate(okHandler)

	t.Run("allows valid bearer token", func(t *testing.T) {
		t.Parallel()

		token, err := manager.Issue(RoleClient)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var body map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body["role"] != string(RoleClient) {
			t.Errorf("role = %q, want %q", body["role"], RoleClient)
		}
	})

	t.Run("rejects missing authorization header", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
		if got := decodeErrorBody(t, rr); got != "missing authorization header" {
			t.Errorf("error = %q, want %q", got, "missing authorization header")
		}
	})

	t.Run("rejects invalid authorization header", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			header string
		}{
			{name: "no bearer prefix", header: "Token abc"},
			{name: "lowercase bearer", header: "bearer abc"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", tt.header)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				if rr.Code != http.StatusUnauthorized {
					t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
				}
				if got := decodeErrorBody(t, rr); got != "invalid authorization header" {
					t.Errorf("error = %q, want %q", got, "invalid authorization header")
				}
			})
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer not-a-valid-token")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
		if got := decodeErrorBody(t, rr); got != "invalid token" {
			t.Errorf("error = %q, want %q", got, "invalid token")
		}
	})
}

func TestMiddleware_RequireRoles(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-secret", time.Hour)
	mw := NewMiddleware(manager)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("allows matching role", func(t *testing.T) {
		t.Parallel()

		handler := mw.RequireRoles(RoleOperator)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ContextWithClaims(req.Context(), &Claims{Role: RoleOperator}))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
		}
	})

	t.Run("forbids non-matching role", func(t *testing.T) {
		t.Parallel()

		handler := mw.RequireRoles(RoleOperator)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ContextWithClaims(req.Context(), &Claims{Role: RoleClient}))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
		if got := decodeErrorBody(t, rr); got != "forbidden" {
			t.Errorf("error = %q, want %q", got, "forbidden")
		}
	})

	t.Run("rejects missing claims", func(t *testing.T) {
		t.Parallel()

		handler := mw.RequireRoles(RoleClient)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
		if got := decodeErrorBody(t, rr); got != "unauthorized" {
			t.Errorf("error = %q, want %q", got, "unauthorized")
		}
	})

	t.Run("works with authenticate chain", func(t *testing.T) {
		t.Parallel()

		handler := mw.Authenticate(mw.RequireRoles(RoleOperator)(okHandler))

		token, err := manager.Issue(RoleOperator)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
		}
	})
}
