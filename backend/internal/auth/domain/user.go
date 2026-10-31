package domain

import (
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
	Role         auth.Role `db:"role"`
}
