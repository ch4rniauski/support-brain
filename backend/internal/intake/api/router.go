package api

import (
	"github.com/ch4rniauski/support-brain/internal/intake/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(messageHandler *handler.MessageHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Post("/messages", messageHandler.SendMessage)

	return r
}
