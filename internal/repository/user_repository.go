package repository

import (
	"context"
	"database/sql"
	"errors"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/pkg/apperrors"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, u *models.User) error {
	query := `INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)
	          RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, query, u.Username, u.Email, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if pqErr.Constraint == "users_email_key" {
				return apperrors.ErrDuplicateEmail
			}
			return apperrors.ErrDuplicateUsername
		}
		return err
	}
	return nil
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return u, err
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u := &models.User{}
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return u, err
}
