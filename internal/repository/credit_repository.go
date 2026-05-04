package repository

import (
	"context"
	"database/sql"
	"errors"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/pkg/apperrors"
	"github.com/google/uuid"
)

type CreditRepository interface {
	Create(ctx context.Context, tx *sql.Tx, c *models.Credit) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Credit, error)
	GetByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Credit, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]models.Credit, error)
	GetAllActive(ctx context.Context) ([]models.Credit, error)
	UpdateRemainingBalance(ctx context.Context, tx *sql.Tx, id uuid.UUID, newBalance string) error
	UpdateStatus(ctx context.Context, tx *sql.Tx, id uuid.UUID, status models.CreditStatus) error
}

type postgresCreditRepository struct {
	db *sql.DB
}

func NewCreditRepository(db *sql.DB) CreditRepository {
	return &postgresCreditRepository{db: db}
}

func (r *postgresCreditRepository) Create(ctx context.Context, tx *sql.Tx, c *models.Credit) error {
	query := `INSERT INTO credits (account_id, principal, interest_rate, term_months, monthly_payment, remaining_balance)
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, status, created_at`
	return tx.QueryRowContext(ctx, query,
		c.AccountID, c.Principal, c.InterestRate, c.TermMonths, c.MonthlyPayment, c.RemainingBalance).
		Scan(&c.ID, &c.Status, &c.CreatedAt)
}

func (r *postgresCreditRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Credit, error) {
	c := &models.Credit{}
	query := `SELECT id, account_id, principal, interest_rate, term_months, monthly_payment, remaining_balance, status, created_at
	          FROM credits WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&c.ID, &c.AccountID, &c.Principal, &c.InterestRate, &c.TermMonths, &c.MonthlyPayment, &c.RemainingBalance, &c.Status, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return c, err
}

func (r *postgresCreditRepository) GetByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Credit, error) {
	c := &models.Credit{}
	query := `SELECT cr.id, cr.account_id, cr.principal, cr.interest_rate, cr.term_months,
	                 cr.monthly_payment, cr.remaining_balance, cr.status, cr.created_at
	          FROM credits cr
	          JOIN accounts a ON a.id = cr.account_id
	          WHERE cr.id = $1 AND a.user_id = $2`
	err := r.db.QueryRowContext(ctx, query, id, userID).
		Scan(&c.ID, &c.AccountID, &c.Principal, &c.InterestRate, &c.TermMonths, &c.MonthlyPayment, &c.RemainingBalance, &c.Status, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return c, err
}

func (r *postgresCreditRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]models.Credit, error) {
	query := `SELECT cr.id, cr.account_id, cr.principal, cr.interest_rate, cr.term_months,
	                 cr.monthly_payment, cr.remaining_balance, cr.status, cr.created_at
	          FROM credits cr
	          JOIN accounts a ON a.id = cr.account_id
	          WHERE a.user_id = $1 AND cr.status = 'active'
	          ORDER BY cr.created_at`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credits []models.Credit
	for rows.Next() {
		var c models.Credit
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Principal, &c.InterestRate, &c.TermMonths, &c.MonthlyPayment, &c.RemainingBalance, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		credits = append(credits, c)
	}
	return credits, rows.Err()
}

func (r *postgresCreditRepository) GetAllActive(ctx context.Context) ([]models.Credit, error) {
	query := `SELECT id, account_id, principal, interest_rate, term_months, monthly_payment, remaining_balance, status, created_at
	          FROM credits WHERE status = 'active'`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credits []models.Credit
	for rows.Next() {
		var c models.Credit
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Principal, &c.InterestRate, &c.TermMonths, &c.MonthlyPayment, &c.RemainingBalance, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		credits = append(credits, c)
	}
	return credits, rows.Err()
}

func (r *postgresCreditRepository) UpdateRemainingBalance(ctx context.Context, tx *sql.Tx, id uuid.UUID, newBalance string) error {
	_, err := tx.ExecContext(ctx, `UPDATE credits SET remaining_balance = $1 WHERE id = $2`, newBalance, id)
	return err
}

func (r *postgresCreditRepository) UpdateStatus(ctx context.Context, tx *sql.Tx, id uuid.UUID, status models.CreditStatus) error {
	_, err := tx.ExecContext(ctx, `UPDATE credits SET status = $1 WHERE id = $2`, status, id)
	return err
}
