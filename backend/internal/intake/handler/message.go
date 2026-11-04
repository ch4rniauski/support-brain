package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/service"
	"github.com/ch4rniauski/support-brain/pkg/httpx"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{service: svc}
}

func (h *MessageHandler) ListQAPairs(w http.ResponseWriter, r *http.Request) {
	var req dto.ListQAPairsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.ListQAPairs(r.Context(), req)
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
