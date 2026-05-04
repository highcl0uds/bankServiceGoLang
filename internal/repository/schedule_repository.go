package repository

import (
	"context"
	"database/sql"
	"time"

	"bank-service-cbr/internal/models"
	"github.com/google/uuid"
)

type PaymentScheduleRepository interface {
	CreateBatch(ctx context.Context, tx *sql.Tx, schedules []models.PaymentSchedule) error
	GetByCreditID(ctx context.Context, creditID uuid.UUID) ([]models.PaymentSchedule, error)
	GetOverdue(ctx context.Context, asOf time.Time) ([]models.PaymentSchedule, error)
	MarkPaid(ctx context.Context, tx *sql.Tx, id uuid.UUID, paidAt time.Time) error
	MarkOverdue(ctx context.Context, tx *sql.Tx, id uuid.UUID) error
	UpdateAmount(ctx context.Context, tx *sql.Tx, id uuid.UUID, newAmount string) error
}

type postgresScheduleRepository struct {
	db *sql.DB
}

func NewScheduleRepository(db *sql.DB) PaymentScheduleRepository {
	return &postgresScheduleRepository{db: db}
}

func (r *postgresScheduleRepository) CreateBatch(ctx context.Context, tx *sql.Tx, schedules []models.PaymentSchedule) error {
	query := `INSERT INTO payment_schedules (credit_id, due_date, amount) VALUES ($1, $2, $3) RETURNING id`
	for i := range schedules {
		err := tx.QueryRowContext(ctx, query, schedules[i].CreditID, schedules[i].DueDate, schedules[i].Amount).
			Scan(&schedules[i].ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresScheduleRepository) GetByCreditID(ctx context.Context, creditID uuid.UUID) ([]models.PaymentSchedule, error) {
	query := `SELECT id, credit_id, due_date, amount, status, paid_at
	          FROM payment_schedules WHERE credit_id = $1 ORDER BY due_date`
	rows, err := r.db.QueryContext(ctx, query, creditID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.PaymentSchedule
	for rows.Next() {
		var s models.PaymentSchedule
		if err := rows.Scan(&s.ID, &s.CreditID, &s.DueDate, &s.Amount, &s.Status, &s.PaidAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *postgresScheduleRepository) GetOverdue(ctx context.Context, asOf time.Time) ([]models.PaymentSchedule, error) {
	query := `SELECT id, credit_id, due_date, amount, status, paid_at
	          FROM payment_schedules
	          WHERE due_date < $1 AND status IN ('pending', 'overdue')
	          ORDER BY due_date`
	rows, err := r.db.QueryContext(ctx, query, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []models.PaymentSchedule
	for rows.Next() {
		var s models.PaymentSchedule
		if err := rows.Scan(&s.ID, &s.CreditID, &s.DueDate, &s.Amount, &s.Status, &s.PaidAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *postgresScheduleRepository) MarkPaid(ctx context.Context, tx *sql.Tx, id uuid.UUID, paidAt time.Time) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE payment_schedules SET status = 'paid', paid_at = $1 WHERE id = $2`, paidAt, id)
	return err
}

func (r *postgresScheduleRepository) MarkOverdue(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE payment_schedules SET status = 'overdue' WHERE id = $1`, id)
	return err
}

func (r *postgresScheduleRepository) UpdateAmount(ctx context.Context, tx *sql.Tx, id uuid.UUID, newAmount string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE payment_schedules SET amount = $1 WHERE id = $2`, newAmount, id)
	return err
}
