package domain

import "github.com/google/uuid"

type TemplateVersion struct {
	ID         uuid.UUID `db:"id"`
	TemplateID uuid.UUID `db:"template_id"`
	Version    int       `db:"version"`
	Body       string    `db:"body"`
	EditorID   *string   `db:"editor_id"`
}
