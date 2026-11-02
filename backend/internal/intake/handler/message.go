package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/service"
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
		writeError(w, http.StatusBadRequest, "ids query parameter is required")
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
			writeError(w, http.StatusBadRequest, "invalid qa pair id")
			return
		}
		ids = append(ids, id)
	}

	resp, err := h.service.ListQAPairs(r.Context(), ids)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.SendMessage(r.Context(), req)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *MessageHandler) ReplyMessage(w http.ResponseWriter, r *http.Request) {
	var req dto.ReplyMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.ReplyMessage(r.Context(), req)
	if err != nil {
		writeMessageError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConversationNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConversationClosed):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
