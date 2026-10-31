package api

import (
	"github.com/ch4rniauski/support-brain/internal/auth/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *handler.AuthHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	return r
}
