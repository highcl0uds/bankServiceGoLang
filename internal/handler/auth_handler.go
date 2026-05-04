package handler

import (
	"net/http"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterInput
	if err := decodeJSON(r, &req); err != nil {
		jsonBadRequest(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		jsonBadRequest(w, err)
		return
	}

	user, err := h.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginInput
	if err := decodeJSON(r, &req); err != nil {
		jsonBadRequest(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		jsonBadRequest(w, err)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, map[string]string{"token": token})
}
