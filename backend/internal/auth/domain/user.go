package domain

import (
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	Role         auth.Role
}
