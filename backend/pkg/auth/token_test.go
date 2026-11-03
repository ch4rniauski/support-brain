package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManager_Issue(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"
	ttl := time.Hour
	manager := NewTokenManager(secret, ttl)

	t.Run("issues token for valid roles", func(t *testing.T) {
		t.Parallel()

		roles := []Role{RoleClient, RoleOperator}
		for _, role := range roles {
			t.Run(string(role), func(t *testing.T) {
				t.Parallel()

				before := time.Now().UTC()
				token, err := manager.Issue(role)
				after := time.Now().UTC()
				if err != nil {
					t.Fatalf("Issue() error = %v", err)
				}
				if token == "" {
					t.Fatal("expected non-empty token")
				}

				claims, err := manager.Parse(token)
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				if claims.Role != role {
					t.Errorf("Role = %q, want %q", claims.Role, role)
				}
				if claims.ExpiresAt == nil {
					t.Fatal("expected ExpiresAt to be set")
				}

				expiresAt := claims.ExpiresAt.Time
				minExpiry := before.Add(ttl)
				maxExpiry := after.Add(ttl)
				if expiresAt.Before(minExpiry.Add(-time.Second)) || expiresAt.After(maxExpiry.Add(time.Second)) {
					t.Errorf("ExpiresAt = %v, want between %v and %v", expiresAt, minExpiry, maxExpiry)
				}
			})
		}
	})

	t.Run("rejects invalid role", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			role Role
		}{
			{name: "empty", role: ""},
			{name: "unknown", role: "admin"},
			{name: "wrong case", role: "CLIENT"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				token, err := manager.Issue(tt.role)
				if !errors.Is(err, ErrInvalidRole) {
					t.Fatalf("error = %v, want %v", err, ErrInvalidRole)
				}
				if token != "" {
					t.Errorf("token = %q, want empty", token)
				}
			})
		}
	})

	t.Run("token is signed with HS256", func(t *testing.T) {
		t.Parallel()

		token, err := manager.Issue(RoleClient)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		parsed, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				t.Fatalf("alg = %v, want HS256", token.Method.Alg())
			}
			return []byte(secret), nil
		})
		if err != nil {
			t.Fatalf("jwt.Parse() error = %v", err)
		}
		if !parsed.Valid {
			t.Fatal("expected valid JWT")
		}
	})
}

func signToken(t *testing.T, method jwt.SigningMethod, secret []byte, claims Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestTokenManager_Parse(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"
	manager := NewTokenManager(secret, time.Hour)

	t.Run("parses valid token", func(t *testing.T) {
		t.Parallel()

		token, err := manager.Issue(RoleOperator)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		claims, err := manager.Parse(token)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if claims.Role != RoleOperator {
			t.Errorf("Role = %q, want %q", claims.Role, RoleOperator)
		}
		if claims.ExpiresAt == nil {
			t.Fatal("expected ExpiresAt to be set")
		}
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			token string
		}{
			{name: "empty", token: ""},
			{name: "not a jwt", token: "not-a-jwt"},
			{name: "truncated", token: "a.b"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				claims, err := manager.Parse(tt.token)
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("error = %v, want %v", err, ErrInvalidToken)
				}
				if claims != nil {
					t.Errorf("claims = %#v, want nil", claims)
				}
			})
		}
	})

	t.Run("rejects token signed with another secret", func(t *testing.T) {
		t.Parallel()

		other := NewTokenManager("other-secret", time.Hour)
		token, err := other.Issue(RoleClient)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		claims, err := manager.Parse(token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidToken)
		}
		if claims != nil {
			t.Errorf("claims = %#v, want nil", claims)
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		t.Parallel()

		token := signToken(t, jwt.SigningMethodHS256, []byte(secret), Claims{
			Role: RoleClient,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute)),
			},
		})

		claims, err := manager.Parse(token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidToken)
		}
		if claims != nil {
			t.Errorf("claims = %#v, want nil", claims)
		}
	})

	t.Run("rejects unexpected signing method", func(t *testing.T) {
		t.Parallel()

		token := signToken(t, jwt.SigningMethodHS384, []byte(secret), Claims{
			Role: RoleClient,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			},
		})

		claims, err := manager.Parse(token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidToken)
		}
		if claims != nil {
			t.Errorf("claims = %#v, want nil", claims)
		}
	})

	t.Run("rejects invalid role claim", func(t *testing.T) {
		t.Parallel()

		token := signToken(t, jwt.SigningMethodHS256, []byte(secret), Claims{
			Role: "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			},
		})

		claims, err := manager.Parse(token)
		if !errors.Is(err, ErrInvalidRole) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidRole)
		}
		if claims != nil {
			t.Errorf("claims = %#v, want nil", claims)
		}
	})
}
