package apperrors

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrDuplicateEmail     = errors.New("email already registered")
	ErrDuplicateUsername  = errors.New("username already taken")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrSelfTransfer       = errors.New("cannot transfer to same account")
)
