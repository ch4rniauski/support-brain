package auth

import "time"

type Config struct {
	JWTSecret string        `envconfig:"AUTH_JWT_SECRET" default:"dev-secret-change-me"`
	TokenTTL  time.Duration `envconfig:"AUTH_TOKEN_TTL" default:"24h"`
}
