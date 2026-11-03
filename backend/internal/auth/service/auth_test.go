package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ch4rniauski/support-brain/internal/auth/domain"
	"github.com/ch4rniauski/support-brain/internal/auth/dto"
	"github.com/ch4rniauski/support-brain/internal/auth/repository"
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserStore struct {
	createErr     error
	createdUser   *domain.User
	getByLogin    *domain.User
	getByLoginErr error
}

func (f *fakeUserStore) Create(ctx context.Context, user *domain.User) error {
	f.createdUser = user
	return f.createErr
}

func (f *fakeUserStore) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	if f.getByLoginErr != nil {
		return nil, f.getByLoginErr
	}
	return f.getByLogin, nil
}

type fakeTokenIssuer struct {
	token string
	err   error
	role  auth.Role
}

func (f *fakeTokenIssuer) Issue(role auth.Role) (string, error) {
	f.role = role
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

func TestAuthService_Register(t *testing.T) {
	t.Parallel()

	validReq := dto.RegisterRequest{
		Login:    "alice",
		Password: "password123",
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		users := &fakeUserStore{}
		tokens := &fakeTokenIssuer{token: "jwt-token"}
		svc := NewAuthService(users, tokens)

		resp, err := svc.Register(context.Background(), validReq)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if resp.Token != "jwt-token" {
			t.Errorf("Token = %q, want %q", resp.Token, "jwt-token")
		}
		if resp.Role != string(auth.RoleClient) {
			t.Errorf("Role = %q, want %q", resp.Role, auth.RoleClient)
		}
		if users.createdUser == nil {
			t.Fatal("expected user to be created")
		}
		if users.createdUser.Login != validReq.Login {
			t.Errorf("created login = %q, want %q", users.createdUser.Login, validReq.Login)
		}
		if users.createdUser.Role != auth.RoleClient {
			t.Errorf("created role = %q, want %q", users.createdUser.Role, auth.RoleClient)
		}
		if users.createdUser.ID.String() == "" {
			t.Error("expected non-empty user ID")
		}
		if err := bcrypt.CompareHashAndPassword(
			[]byte(users.createdUser.PasswordHash),
			[]byte(validReq.Password),
		); err != nil {
			t.Errorf("password hash does not match: %v", err)
		}
		if tokens.role != auth.RoleClient {
			t.Errorf("issued role = %q, want %q", tokens.role, auth.RoleClient)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			req  dto.RegisterRequest
		}{
			{
				name: "short login",
				req:  dto.RegisterRequest{Login: "ab", Password: "password123"},
			},
			{
				name: "empty login",
				req:  dto.RegisterRequest{Login: "", Password: "password123"},
			},
			{
				name: "short password",
				req:  dto.RegisterRequest{Login: "alice", Password: "short"},
			},
			{
				name: "empty password",
				req:  dto.RegisterRequest{Login: "alice", Password: ""},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				users := &fakeUserStore{}
				svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

				resp, err := svc.Register(context.Background(), tt.req)
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
				}
				if resp != nil {
					t.Errorf("response = %#v, want nil", resp)
				}
				if users.createdUser != nil {
					t.Error("user must not be created on invalid input")
				}
			})
		}
	})

	t.Run("login taken", func(t *testing.T) {
		t.Parallel()

		users := &fakeUserStore{createErr: repository.ErrAlreadyExists}
		svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

		resp, err := svc.Register(context.Background(), validReq)
		if !errors.Is(err, ErrLoginTaken) {
			t.Fatalf("error = %v, want %v", err, ErrLoginTaken)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("create user fails", func(t *testing.T) {
		t.Parallel()

		storeErr := errors.New("db down")
		users := &fakeUserStore{createErr: storeErr}
		svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

		resp, err := svc.Register(context.Background(), validReq)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, storeErr) {
			t.Fatalf("error = %v, want wrapped %v", err, storeErr)
		}
		if errors.Is(err, ErrLoginTaken) {
			t.Fatal("must not map generic store error to ErrLoginTaken")
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("issue token fails", func(t *testing.T) {
		t.Parallel()

		issueErr := errors.New("signer failed")
		svc := NewAuthService(&fakeUserStore{}, &fakeTokenIssuer{err: issueErr})

		resp, err := svc.Register(context.Background(), validReq)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, issueErr) {
			t.Fatalf("error = %v, want wrapped %v", err, issueErr)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func TestAuthService_Login(t *testing.T) {
	t.Parallel()

	const password = "password123"
	validReq := dto.LoginRequest{
		Login:    "alice",
		Password: password,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		users := &fakeUserStore{
			getByLogin: &domain.User{
				Login:        validReq.Login,
				PasswordHash: mustHashPassword(t, password),
				Role:         auth.RoleOperator,
			},
		}
		tokens := &fakeTokenIssuer{token: "jwt-token"}
		svc := NewAuthService(users, tokens)

		resp, err := svc.Login(context.Background(), validReq)
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if resp.Token != "jwt-token" {
			t.Errorf("Token = %q, want %q", resp.Token, "jwt-token")
		}
		if resp.Role != string(auth.RoleOperator) {
			t.Errorf("Role = %q, want %q", resp.Role, auth.RoleOperator)
		}
		if tokens.role != auth.RoleOperator {
			t.Errorf("issued role = %q, want %q", tokens.role, auth.RoleOperator)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			req  dto.LoginRequest
		}{
			{
				name: "empty login",
				req:  dto.LoginRequest{Login: "", Password: password},
			},
			{
				name: "empty password",
				req:  dto.LoginRequest{Login: "alice", Password: ""},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				svc := NewAuthService(&fakeUserStore{}, &fakeTokenIssuer{token: "jwt-token"})

				resp, err := svc.Login(context.Background(), tt.req)
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
				}
				if resp != nil {
					t.Errorf("response = %#v, want nil", resp)
				}
			})
		}
	})

	t.Run("user not found", func(t *testing.T) {
		t.Parallel()

		users := &fakeUserStore{getByLoginErr: repository.ErrNotFound}
		svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

		resp, err := svc.Login(context.Background(), validReq)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidCredentials)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()

		users := &fakeUserStore{
			getByLogin: &domain.User{
				Login:        validReq.Login,
				PasswordHash: mustHashPassword(t, password),
				Role:         auth.RoleClient,
			},
		}
		svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

		resp, err := svc.Login(context.Background(), dto.LoginRequest{
			Login:    validReq.Login,
			Password: "wrong-password",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidCredentials)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("get user fails", func(t *testing.T) {
		t.Parallel()

		storeErr := errors.New("db down")
		users := &fakeUserStore{getByLoginErr: storeErr}
		svc := NewAuthService(users, &fakeTokenIssuer{token: "jwt-token"})

		resp, err := svc.Login(context.Background(), validReq)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, storeErr) {
			t.Fatalf("error = %v, want wrapped %v", err, storeErr)
		}
		if errors.Is(err, ErrInvalidCredentials) {
			t.Fatal("must not map generic store error to ErrInvalidCredentials")
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("issue token fails", func(t *testing.T) {
		t.Parallel()

		issueErr := errors.New("signer failed")
		users := &fakeUserStore{
			getByLogin: &domain.User{
				Login:        validReq.Login,
				PasswordHash: mustHashPassword(t, password),
				Role:         auth.RoleClient,
			},
		}
		svc := NewAuthService(users, &fakeTokenIssuer{err: issueErr})

		resp, err := svc.Login(context.Background(), validReq)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, issueErr) {
			t.Fatalf("error = %v, want wrapped %v", err, issueErr)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})
}
