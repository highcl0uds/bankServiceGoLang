package models

import (
	"time"

	"github.com/google/uuid"
)

type Card struct {
	ID              uuid.UUID `json:"-"`
	AccountID       uuid.UUID `json:"account_id"`
	NumberEncrypted string    `json:"-"`
	NumberHMAC      []byte    `json:"-"`
	ExpiryEncrypted string    `json:"-"`
	CVVHash         string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}

type CardResponse struct {
	ID        uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	Number    string    `json:"number"`
	Expiry    string    `json:"expiry"`
	CreatedAt time.Time `json:"created_at"`
}
