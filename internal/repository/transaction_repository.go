package repository

import (
	"context"
	"database/sql"
	"time"

	"bank-service-cbr/internal/models"
	"github.com/google/uuid"
)

type MonthlyStats struct {
	TotalIncome  string
	TotalExpense string
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *sql.Tx, t *models.Transaction) error
	GetByAccountID(ctx context.Context, accountID uuid.UUID, from, to time.Time) ([]models.Transaction, error)
	GetMonthlyStats(ctx context.Context, userID uuid.UUID, year, month int) (*MonthlyStats, error)
}

type postgresTransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &postgresTransactionRepository{db: db}
}

func (r *postgresTransactionRepository) Create(ctx context.Context, tx *sql.Tx, t *models.Transaction) error {
	query := `INSERT INTO transactions (from_account_id, to_account_id, amount, type, description)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return tx.QueryRowContext(ctx, query, t.FromAccountID, t.ToAccountID, t.Amount, t.Type, t.Description).
		Scan(&t.ID, &t.CreatedAt)
}

func (r *postgresTransactionRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID, from, to time.Time) ([]models.Transaction, error) {
	query := `SELECT id, from_account_id, to_account_id, amount, type, description, created_at
	          FROM transactions
	          WHERE (from_account_id = $1 OR to_account_id = $1)
	            AND created_at BETWEEN $2 AND $3
	          ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, accountID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (r *postgresTransactionRepository) GetMonthlyStats(ctx context.Context, userID uuid.UUID, year, month int) (*MonthlyStats, error) {
	query := `
		SELECT
		    COALESCE(SUM(CASE WHEN t.type IN ('deposit','transfer_in','credit_disbursement') THEN t.amount ELSE 0 END), 0) AS income,
		    COALESCE(SUM(CASE WHEN t.type IN ('withdrawal','transfer_out','credit_repayment') THEN t.amount ELSE 0 END), 0) AS expense
		FROM transactions t
		WHERE (
		    (t.type IN ('deposit','transfer_in','credit_disbursement')
		     AND t.to_account_id IN (SELECT id FROM accounts WHERE user_id = $1))
		    OR
		    (t.type IN ('withdrawal','transfer_out','credit_repayment')
		     AND t.from_account_id IN (SELECT id FROM accounts WHERE user_id = $1))
		)
		  AND EXTRACT(YEAR  FROM t.created_at) = $2
		  AND EXTRACT(MONTH FROM t.created_at) = $3`

	stats := &MonthlyStats{}
	err := r.db.QueryRowContext(ctx, query, userID, year, month).
		Scan(&stats.TotalIncome, &stats.TotalExpense)
	return stats, err
}
