package api

import (
	"github.com/ch4rniauski/support-brain/internal/embedding/handler"
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(similarityHandler *handler.SimilarityHandler, authMiddleware *auth.Middleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Post("/similar", similarityHandler.FindSimilar)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRoles(auth.RoleOperator))
		r.Post("/similar/by-text", similarityHandler.FindSimilarByText)
	})

	return r
}
