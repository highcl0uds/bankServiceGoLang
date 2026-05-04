package repository

import (
	"context"
	"database/sql"
	"errors"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/pkg/apperrors"
	"github.com/google/uuid"
)

type AccountRepository interface {
	Create(ctx context.Context, a *models.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error)
	GetByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Account, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error)
	GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id uuid.UUID) (*models.Account, error)
	UpdateBalance(ctx context.Context, tx *sql.Tx, id uuid.UUID, delta string) error
}

type postgresAccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) AccountRepository {
	return &postgresAccountRepository{db: db}
}

func (r *postgresAccountRepository) Create(ctx context.Context, a *models.Account) error {
	query := `INSERT INTO accounts (user_id, currency) VALUES ($1, $2)
	          RETURNING id, balance, created_at`
	return r.db.QueryRowContext(ctx, query, a.UserID, a.Currency).
		Scan(&a.ID, &a.Balance, &a.CreatedAt)
}

func (r *postgresAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Account, error) {
	a := &models.Account{}
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&a.ID, &a.UserID, &a.Balance, &a.Currency, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return a, err
}

func (r *postgresAccountRepository) GetByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Account, error) {
	a := &models.Account{}
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE id = $1 AND user_id = $2`
	err := r.db.QueryRowContext(ctx, query, id, userID).
		Scan(&a.ID, &a.UserID, &a.Balance, &a.Currency, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return a, err
}

func (r *postgresAccountRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error) {
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE user_id = $1 ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.Balance, &a.Currency, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *postgresAccountRepository) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id uuid.UUID) (*models.Account, error) {
	a := &models.Account{}
	query := `SELECT id, user_id, balance, currency, created_at FROM accounts WHERE id = $1 FOR UPDATE`
	err := tx.QueryRowContext(ctx, query, id).
		Scan(&a.ID, &a.UserID, &a.Balance, &a.Currency, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return a, err
}

func (r *postgresAccountRepository) UpdateBalance(ctx context.Context, tx *sql.Tx, id uuid.UUID, delta string) error {
	query := `UPDATE accounts SET balance = balance + $1 WHERE id = $2`
	_, err := tx.ExecContext(ctx, query, delta, id)
	return err
}
