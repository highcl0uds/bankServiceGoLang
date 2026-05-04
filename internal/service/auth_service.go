package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/pkg/apperrors"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authService struct {
	userRepository repository.UserRepository
	jwtSecret      string
	log            *logrus.Logger
}

func NewAuthService(userRepository repository.UserRepository, jwtSecret string, log *logrus.Logger) AuthService {
	return &authService{userRepository: userRepository, jwtSecret: jwtSecret, log: log}
}

func (s *authService) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := s.userRepository.Create(ctx, u); err != nil {
		return nil, err
	}
	s.log.WithField("user_id", u.ID).Info("user registered")
	return u, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return "", apperrors.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", apperrors.ErrInvalidCredentials
	}
	token, err := generateJWT(u.ID, s.jwtSecret)
	if err != nil {
		return "", err
	}
	s.log.WithField("user_id", u.ID).Info("user logged in")
	return token, nil
}

func generateJWT(userID uuid.UUID, secret string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
