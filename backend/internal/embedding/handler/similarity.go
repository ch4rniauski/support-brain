package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ch4rniauski/support-brain/internal/embedding/dto"
	"github.com/ch4rniauski/support-brain/internal/embedding/service"
	"github.com/ch4rniauski/support-brain/pkg/httpx"
)

type SimilarityHandler struct {
	service *service.EmbeddingService
}

func NewSimilarityHandler(svc *service.EmbeddingService) *SimilarityHandler {
	return &SimilarityHandler{service: svc}
}

func (h *SimilarityHandler) FindSimilar(w http.ResponseWriter, r *http.Request) {
	var req dto.FindSimilarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.FindSimilar(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrEmbeddingNotFound):
			httpx.Error(w, http.StatusNotFound, err.Error())
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

func (h *SimilarityHandler) FindSimilarByText(w http.ResponseWriter, r *http.Request) {
	var req dto.FindSimilarByTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.FindSimilarByText(r.Context(), req)
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
