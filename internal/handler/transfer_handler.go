package handler

import (
	"net/http"

	"github.com/google/uuid"

	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/service"
)

type TransferHandler struct {
	transferService service.TransferService
}

func NewTransferHandler(transferService service.TransferService) *TransferHandler {
	return &TransferHandler{transferService: transferService}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        string `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	fromID, err := uuid.Parse(req.FromAccountID)
	if err != nil {
		http.Error(w, `{"error":"invalid from_account_id"}`, http.StatusBadRequest)
		return
	}
	toID, err := uuid.Parse(req.ToAccountID)
	if err != nil {
		http.Error(w, `{"error":"invalid to_account_id"}`, http.StatusBadRequest)
		return
	}

	if err := h.transferService.Transfer(r.Context(), userID, fromID, toID, req.Amount); err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}
