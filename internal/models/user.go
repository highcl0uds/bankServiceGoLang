package models

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]{3,64}$`)
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterInput) Validate() error {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(r.Email)
	if r.Username == "" || r.Email == "" || r.Password == "" {
		return errors.New("username, email and password are required")
	}
	if !emailRegex.MatchString(r.Email) {
		return errors.New("invalid email format")
	}
	if !usernameRegex.MatchString(r.Username) {
		return errors.New("username must be 3-64 characters: letters, digits, _ or -")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginInput) Validate() error {
	if strings.TrimSpace(l.Email) == "" || l.Password == "" {
		return errors.New("email and password are required")
	}
	return nil
}
