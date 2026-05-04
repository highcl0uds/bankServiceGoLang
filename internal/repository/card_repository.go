package repository

import (
	"context"
	"database/sql"

	"bank-service-cbr/internal/models"
	"github.com/google/uuid"
)

type CardRepository interface {
	Create(ctx context.Context, c *models.Card) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Card, error)
	ExistsWithHMAC(ctx context.Context, hmac []byte) (bool, error)
}

type postgresCardRepository struct {
	db *sql.DB
}

func NewCardRepository(db *sql.DB) CardRepository {
	return &postgresCardRepository{db: db}
}

func (r *postgresCardRepository) Create(ctx context.Context, c *models.Card) error {
	query := `INSERT INTO cards (account_id, number_encrypted, number_hmac, expiry_encrypted, cvv_hash)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		c.AccountID, c.NumberEncrypted, c.NumberHMAC, c.ExpiryEncrypted, c.CVVHash).
		Scan(&c.ID, &c.CreatedAt)
}

func (r *postgresCardRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Card, error) {
	query := `SELECT c.id, c.account_id, c.number_encrypted, c.number_hmac, c.expiry_encrypted, c.cvv_hash, c.created_at
	          FROM cards c
	          JOIN accounts a ON a.id = c.account_id
	          WHERE a.user_id = $1
	          ORDER BY c.created_at`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var c models.Card
		if err := rows.Scan(&c.ID, &c.AccountID, &c.NumberEncrypted, &c.NumberHMAC, &c.ExpiryEncrypted, &c.CVVHash, &c.CreatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *postgresCardRepository) ExistsWithHMAC(ctx context.Context, hmac []byte) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cards WHERE number_hmac = $1)`, hmac).Scan(&exists)
	return exists, err
}
