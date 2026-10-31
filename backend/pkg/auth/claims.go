package auth

import "github.com/golang-jwt/jwt/v5"

type Role string

const (
	RoleClient   Role = "client"
	RoleOperator Role = "operator"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleClient, RoleOperator:
		return true
	default:
		return false
	}
}

type Claims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}
