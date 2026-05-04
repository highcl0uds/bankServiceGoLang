package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/service"
)

type AccountHandler struct {
	accountService service.AccountService
}

func NewAccountHandler(accountService service.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	acc, err := h.accountService.Create(r.Context(), userID)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, acc)
}

func (h *AccountHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	accounts, err := h.accountService.GetByUser(r.Context(), userID)
	if err != nil {
		jsonError(w, err)
		return
	}
	if accounts == nil {
		jsonOK(w, []struct{}{})
		return
	}
	jsonOK(w, accounts)
}

func (h *AccountHandler) GetOne(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	accountID, err := uuid.Parse(mux.Vars(r)["accountId"])
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}
	acc, err := h.accountService.GetByID(r.Context(), userID, accountID)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, acc)
}

func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	accountID, err := uuid.Parse(mux.Vars(r)["accountId"])
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Amount string `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	acc, err := h.accountService.Deposit(r.Context(), userID, accountID, req.Amount)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, acc)
}

func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	accountID, err := uuid.Parse(mux.Vars(r)["accountId"])
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Amount string `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	acc, err := h.accountService.Withdraw(r.Context(), userID, accountID, req.Amount)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, acc)
}
