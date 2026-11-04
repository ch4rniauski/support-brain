package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/ch4rniauski/support-brain/internal/template/service"
	"github.com/ch4rniauski/support-brain/pkg/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	service *service.TemplateService
}

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: svc}
}

func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	status := domain.TemplateStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		httpx.Error(w, http.StatusBadRequest, "status query parameter is required")
		return
	}

	resp, err := h.service.ListByStatus(r.Context(), status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidStatus):
			httpx.Error(w, http.StatusBadRequest, "status must be one of: draft, approved, rejected, archived")
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid template id")
		return
	}

	resp, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTemplateNotFound):
			httpx.Error(w, http.StatusNotFound, err.Error())
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}
