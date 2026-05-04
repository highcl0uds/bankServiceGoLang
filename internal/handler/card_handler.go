package handler

import (
	"net/http"

	"github.com/google/uuid"

	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/service"
)

type CardHandler struct {
	cardService service.CardService
}

func NewCardHandler(cardService service.CardService) *CardHandler {
	return &CardHandler{cardService: cardService}
}

func (h *CardHandler) Issue(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		AccountID string `json:"account_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		http.Error(w, `{"error":"invalid account_id"}`, http.StatusBadRequest)
		return
	}

	card, err := h.cardService.IssueCard(r.Context(), userID, accountID)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, card)
}

func (h *CardHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	cards, err := h.cardService.GetCards(r.Context(), userID)
	if err != nil {
		jsonError(w, err)
		return
	}
	if cards == nil {
		jsonOK(w, []struct{}{})
		return
	}
	jsonOK(w, cards)
}
