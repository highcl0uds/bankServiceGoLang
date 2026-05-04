package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"bank-service-cbr/pkg/apperrors"
)

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func jsonBadRequest(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func jsonError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, apperrors.ErrUnauthorized), errors.Is(err, apperrors.ErrInvalidCredentials), errors.Is(err, apperrors.ErrInvalidToken):
		code = http.StatusUnauthorized
	case errors.Is(err, apperrors.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, apperrors.ErrDuplicateEmail), errors.Is(err, apperrors.ErrDuplicateUsername):
		code = http.StatusConflict
	case errors.Is(err, apperrors.ErrInsufficientFunds), errors.Is(err, apperrors.ErrInvalidAmount), errors.Is(err, apperrors.ErrSelfTransfer):
		code = http.StatusUnprocessableEntity
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
