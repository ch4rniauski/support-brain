package api

import (
	"github.com/ch4rniauski/support-brain/internal/knowledge/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(clusterHandler *handler.ClusterHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Post("/clusters/templates-by-qa-pairs", clusterHandler.FindTemplatesByQAPairs)

	return r
}
