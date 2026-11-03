package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/ch4rniauski/support-brain/internal/template/service"
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
		writeError(w, http.StatusBadRequest, "status query parameter is required")
		return
	}

	resp, err := h.service.ListByStatus(r.Context(), status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidStatus):
			writeError(w, http.StatusBadRequest, "status must be one of: draft, approved, rejected, archived")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
