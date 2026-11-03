package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/service"
	"github.com/ch4rniauski/support-brain/pkg/httpx"
	"github.com/google/uuid"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{service: svc}
}

func (h *MessageHandler) ListQAPairs(w http.ResponseWriter, r *http.Request) {
	rawIDs := strings.TrimSpace(r.URL.Query().Get("ids"))
	if rawIDs == "" {
		httpx.Error(w, http.StatusBadRequest, "ids query parameter is required")
		return
	}

	parts := strings.Split(rawIDs, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid qa pair id")
			return
		}
		ids = append(ids, id)
	}

	resp, err := h.service.ListQAPairs(r.Context(), ids)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.SendMessage(r.Context(), req)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, resp)
}

func (h *MessageHandler) ReplyMessage(w http.ResponseWriter, r *http.Request) {
	var req dto.ReplyMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.ReplyMessage(r.Context(), req)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, resp)
}

func writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConversationNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConversationClosed):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
