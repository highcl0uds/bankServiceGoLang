package models

import (
	"time"

	"github.com/google/uuid"
)

type ScheduleStatus string

const (
	SchedulePending ScheduleStatus = "pending"
	SchedulePaid    ScheduleStatus = "paid"
	ScheduleOverdue ScheduleStatus = "overdue"
)

type PaymentSchedule struct {
	ID       uuid.UUID      `json:"id"`
	CreditID uuid.UUID      `json:"credit_id"`
	DueDate  time.Time      `json:"due_date"`
	Amount   string         `json:"amount"`
	Status   ScheduleStatus `json:"status"`
	PaidAt   *time.Time     `json:"paid_at,omitempty"`
}
