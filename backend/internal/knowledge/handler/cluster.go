package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ch4rniauski/support-brain/internal/knowledge/dto"
	"github.com/ch4rniauski/support-brain/internal/knowledge/service"
	"github.com/ch4rniauski/support-brain/pkg/httpx"
)

type ClusterHandler struct {
	service *service.ClusterService
}

func NewClusterHandler(svc *service.ClusterService) *ClusterHandler {
	return &ClusterHandler{service: svc}
}

func (h *ClusterHandler) FindTemplatesByQAPairs(w http.ResponseWriter, r *http.Request) {
	var req dto.FindTemplatesByQAPairsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.FindTemplatesByQAPairIDs(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}
