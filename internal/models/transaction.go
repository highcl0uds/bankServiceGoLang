package models

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionDeposit            TransactionType = "deposit"
	TransactionWithdrawal         TransactionType = "withdrawal"
	TransactionTransferOut        TransactionType = "transfer_out"
	TransactionTransferIn         TransactionType = "transfer_in"
	TransactionCreditDisbursement TransactionType = "credit_disbursement"
	TransactionCreditRepayment    TransactionType = "credit_repayment"
)

type Transaction struct {
	ID            uuid.UUID       `json:"id"`
	FromAccountID *uuid.UUID      `json:"from_account_id,omitempty"`
	ToAccountID   *uuid.UUID      `json:"to_account_id,omitempty"`
	Amount        string          `json:"amount"`
	Type          TransactionType `json:"type"`
	Description   string          `json:"description"`
	CreatedAt     time.Time       `json:"created_at"`
}
