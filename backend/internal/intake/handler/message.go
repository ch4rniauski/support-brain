package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/service"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{service: svc}
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.SendMessage(r.Context(), req)
	if err != nil {
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
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
