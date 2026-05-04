package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/service"
)

type CreditHandler struct {
	creditService service.CreditService
}

func NewCreditHandler(creditService service.CreditService) *CreditHandler {
	return &CreditHandler{creditService: creditService}
}

func (h *CreditHandler) TakeCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		AccountID  string `json:"account_id"`
		Principal  string `json:"principal"`
		TermMonths int    `json:"term_months"`
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

	credit, schedules, err := h.creditService.TakeCredit(r.Context(), userID, accountID, req.Principal, req.TermMonths)
	if err != nil {
		jsonError(w, err)
		return
	}

	jsonCreated(w, map[string]interface{}{
		"credit":   credit,
		"schedule": schedules,
	})
}

func (h *CreditHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	creditID, err := uuid.Parse(mux.Vars(r)["creditId"])
	if err != nil {
		http.Error(w, `{"error":"invalid credit id"}`, http.StatusBadRequest)
		return
	}

	schedules, err := h.creditService.GetSchedule(r.Context(), userID, creditID)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, schedules)
}
