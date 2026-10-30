package domain

import "github.com/google/uuid"

type TemplateVersion struct {
	ID         uuid.UUID
	TemplateID uuid.UUID
	Version    int
	Body       string
	EditorID   *string
}
