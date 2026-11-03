package auth

import (
	"context"
	"testing"
)

func TestContextWithClaims(t *testing.T) {
	t.Parallel()

	t.Run("stores claims in context", func(t *testing.T) {
		t.Parallel()

		claims := &Claims{Role: RoleOperator}
		ctx := ContextWithClaims(context.Background(), claims)

		got, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected claims in context")
		}
		if got != claims {
			t.Fatalf("claims = %#v, want %#v", got, claims)
		}
		if got.Role != RoleOperator {
			t.Errorf("Role = %q, want %q", got.Role, RoleOperator)
		}
	})

	t.Run("overwrites previous claims", func(t *testing.T) {
		t.Parallel()

		ctx := ContextWithClaims(context.Background(), &Claims{Role: RoleClient})
		ctx = ContextWithClaims(ctx, &Claims{Role: RoleOperator})

		got, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected claims in context")
		}
		if got.Role != RoleOperator {
			t.Errorf("Role = %q, want %q", got.Role, RoleOperator)
		}
	})

	t.Run("stores nil claims", func(t *testing.T) {
		t.Parallel()

		ctx := ContextWithClaims(context.Background(), nil)

		got, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected claims key to be present")
		}
		if got != nil {
			t.Errorf("claims = %#v, want nil", got)
		}
	})
}

func TestClaimsFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns false when claims are missing", func(t *testing.T) {
		t.Parallel()

		got, ok := ClaimsFromContext(context.Background())
		if ok {
			t.Fatal("expected ok = false")
		}
		if got != nil {
			t.Errorf("claims = %#v, want nil", got)
		}
	})

	t.Run("returns claims when present", func(t *testing.T) {
		t.Parallel()

		want := &Claims{Role: RoleClient}
		ctx := ContextWithClaims(context.Background(), want)

		got, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected ok = true")
		}
		if got != want {
			t.Fatalf("claims = %#v, want %#v", got, want)
		}
	})

	t.Run("returns false when value has wrong type", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), claimsContextKey, "not-claims")

		got, ok := ClaimsFromContext(ctx)
		if ok {
			t.Fatal("expected ok = false")
		}
		if got != nil {
			t.Errorf("claims = %#v, want nil", got)
		}
	})
}
