package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
)

type SchedulerService struct {
	db                *sql.DB
	creditRepository  repository.CreditRepository
	schedRepository   repository.PaymentScheduleRepository
	accountRepository repository.AccountRepository
	txRepository      repository.TransactionRepository
	userRepository    repository.UserRepository
	emailService      EmailService
	log               *logrus.Logger
}

func NewSchedulerService(
	db *sql.DB,
	creditRepository repository.CreditRepository,
	schedRepository repository.PaymentScheduleRepository,
	accountRepository repository.AccountRepository,
	txRepository repository.TransactionRepository,
	userRepository repository.UserRepository,
	emailService EmailService,
	log *logrus.Logger,
) *SchedulerService {
	return &SchedulerService{
		db:                db,
		creditRepository:  creditRepository,
		schedRepository:   schedRepository,
		accountRepository: accountRepository,
		txRepository:      txRepository,
		userRepository:    userRepository,
		emailService:      emailService,
		log:               log,
	}
}

func (s *SchedulerService) Start(ctx context.Context) {
	s.log.Info("scheduler started")
	s.ProcessOverduePayments(ctx)

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.ProcessOverduePayments(ctx)
		case <-ctx.Done():
			s.log.Info("scheduler stopped")
			return
		}
	}
}

func (s *SchedulerService) ProcessOverduePayments(ctx context.Context) {
	s.log.Info("processing overdue payments")

	overdueSchedules, err := s.schedRepository.GetOverdue(ctx, time.Now())
	if err != nil {
		s.log.WithError(err).Error("failed to fetch overdue schedules")
		return
	}

	processed, failed := 0, 0
	for _, sched := range overdueSchedules {
		if err := s.processPayment(ctx, sched); err != nil {
			s.log.WithError(err).WithField("schedule_id", sched.ID).Error("payment processing failed")
			failed++
		} else {
			processed++
		}
	}

	s.log.WithFields(logrus.Fields{
		"processed": processed,
		"failed":    failed,
		"total":     len(overdueSchedules),
	}).Info("overdue payment processing complete")
}

func (s *SchedulerService) processPayment(ctx context.Context, sched models.PaymentSchedule) error {
	credit, err := s.creditRepository.GetByID(ctx, sched.CreditID)
	if err != nil {
		return fmt.Errorf("get credit: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	acc, err := s.accountRepository.GetByIDForUpdate(ctx, tx, credit.AccountID)
	if err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	balance, _ := strconv.ParseFloat(acc.Balance, 64)
	amount, _ := strconv.ParseFloat(sched.Amount, 64)

	if balance >= amount {
		neg := fmt.Sprintf("-%.2f", amount)
		if err := s.accountRepository.UpdateBalance(ctx, tx, acc.ID, neg); err != nil {
			return err
		}

		fromAccID := acc.ID
		repayTx := &models.Transaction{
			FromAccountID: &fromAccID,
			Amount:        sched.Amount,
			Type:          models.TransactionCreditRepayment,
			Description:   fmt.Sprintf("Платеж по кредиту %s", credit.ID),
		}
		if err := s.txRepository.Create(ctx, tx, repayTx); err != nil {
			return err
		}

		if err := s.schedRepository.MarkPaid(ctx, tx, sched.ID, time.Now()); err != nil {
			return err
		}

		interestRate, _ := strconv.ParseFloat(credit.InterestRate, 64)
		remaining, _ := strconv.ParseFloat(credit.RemainingBalance, 64)
		interestPortion := remaining * interestRate / 12
		principalPortion := amount - interestPortion
		if principalPortion < 0 {
			principalPortion = 0
		}
		if principalPortion > remaining {
			principalPortion = remaining
		}
		remaining -= principalPortion
		if remaining < 0 {
			remaining = 0
		}
		newBalance := fmt.Sprintf("%.2f", remaining)
		if err := s.creditRepository.UpdateRemainingBalance(ctx, tx, credit.ID, newBalance); err != nil {
			return err
		}

		if remaining == 0 {
			if err := s.creditRepository.UpdateStatus(ctx, tx, credit.ID, models.CreditPaid); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		user, _ := s.userRepository.GetByID(ctx, acc.UserID)
		if user != nil {
			s.emailService.SendPaymentProcessed(user.Email, user.Username, sched.Amount)
		}
	} else {
		penalty := amount * 0.10
		newAmount := fmt.Sprintf("%.2f", amount+penalty)
		penaltyStr := fmt.Sprintf("%.2f", penalty)

		if err := s.schedRepository.MarkOverdue(ctx, tx, sched.ID); err != nil {
			return err
		}
		if err := s.schedRepository.UpdateAmount(ctx, tx, sched.ID, newAmount); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		s.log.WithFields(logrus.Fields{
			"schedule_id": sched.ID,
			"amount":      amount,
			"balance":     balance,
			"penalty":     penalty,
		}).Warn("insufficient funds for credit payment, penalty applied")

		user, _ := s.userRepository.GetByID(ctx, acc.UserID)
		if user != nil {
			s.emailService.SendPaymentOverdue(user.Email, user.Username, sched.Amount, penaltyStr)
		}
	}
	return nil
}
