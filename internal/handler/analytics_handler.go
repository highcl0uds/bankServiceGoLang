package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/service"
)

type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.analyticsService.GetAnalytics(r.Context(), userID)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, result)
}

func (h *AnalyticsHandler) GetPrediction(w http.ResponseWriter, r *http.Request) {
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

	days := parseIntQuery(r, "days", 30)

	result, err := h.analyticsService.GetPrediction(r.Context(), userID, accountID, days)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, result)
}
