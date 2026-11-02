package api

import (
	"github.com/ch4rniauski/support-brain/internal/intake/handler"
	"github.com/ch4rniauski/support-brain/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(messageHandler *handler.MessageHandler, authMiddleware *auth.Middleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Post("/messages", messageHandler.SendMessage)
	r.Get("/qa-pairs", messageHandler.ListQAPairs)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRoles(auth.RoleOperator))
		r.Post("/messages/replies", messageHandler.ReplyMessage)
	})

	return r
}
