package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/auth/domain"
	"github.com/ch4rniauski/support-brain/internal/auth/dto"
	"github.com/ch4rniauski/support-brain/internal/auth/repository"
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrLoginTaken         = errors.New("login already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type userStore interface {
	Create(ctx context.Context, user *domain.User) error
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
}

type tokenIssuer interface {
	Issue(role auth.Role) (string, error)
}

type AuthService struct {
	users    userStore
	tokens   tokenIssuer
	validate *validator.Validate
}

func NewAuthService(
	users userStore,
	tokens tokenIssuer,
) *AuthService {
	return &AuthService{
		users:    users,
		tokens:   tokens,
		validate: validator.New(),
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Login:        req.Login,
		PasswordHash: string(passwordHash),
		Role:         auth.RoleClient,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, ErrLoginTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.tokens.Issue(user.Role)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &dto.AuthResponse{
		Token: token,
		Role:  string(user.Role),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	user, err := s.users.GetByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user.Role)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &dto.AuthResponse{
		Token: token,
		Role:  string(user.Role),
	}, nil
}
