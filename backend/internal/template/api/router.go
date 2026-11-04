package api

import (
	"github.com/ch4rniauski/support-brain/internal/template/handler"
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(templateHandler *handler.TemplateHandler, authMiddleware *auth.Middleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRoles(auth.RoleOperator))
		r.Get("/templates", templateHandler.ListTemplates)
		r.Get("/templates/{id}", templateHandler.GetTemplate)
		r.Patch("/templates/{id}", templateHandler.UpdateTemplate)
	})

	return r
}
