package models

import (
	"time"

	"github.com/google/uuid"
)

type CreditStatus string

const (
	CreditActive    CreditStatus = "active"
	CreditPaid      CreditStatus = "paid"
	CreditDefaulted CreditStatus = "defaulted"
)

type Credit struct {
	ID               uuid.UUID    `json:"id"`
	AccountID        uuid.UUID    `json:"account_id"`
	Principal        string       `json:"principal"`
	InterestRate     string       `json:"interest_rate"`
	TermMonths       int          `json:"term_months"`
	MonthlyPayment   string       `json:"monthly_payment"`
	RemainingBalance string       `json:"remaining_balance"`
	Status           CreditStatus `json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
}
