package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/pkg/apperrors"
)

type CreditService interface {
	TakeCredit(ctx context.Context, userID, accountID uuid.UUID, principal string, termMonths int) (*models.Credit, []models.PaymentSchedule, error)
	GetSchedule(ctx context.Context, userID uuid.UUID, creditID uuid.UUID) ([]models.PaymentSchedule, error)
}

type creditService struct {
	db                *sql.DB
	creditRepository  repository.CreditRepository
	schedRepository   repository.PaymentScheduleRepository
	accountRepository repository.AccountRepository
	txRepository      repository.TransactionRepository
	userRepository    repository.UserRepository
	cbrService        CBRService
	emailService      EmailService
	log               *logrus.Logger
}

func NewCreditService(
	db *sql.DB,
	creditRepository repository.CreditRepository,
	schedRepository repository.PaymentScheduleRepository,
	accountRepository repository.AccountRepository,
	txRepository repository.TransactionRepository,
	userRepository repository.UserRepository,
	cbrService CBRService,
	emailService EmailService,
	log *logrus.Logger,
) CreditService {
	return &creditService{
		db:                db,
		creditRepository:  creditRepository,
		schedRepository:   schedRepository,
		accountRepository: accountRepository,
		txRepository:      txRepository,
		userRepository:    userRepository,
		cbrService:        cbrService,
		emailService:      emailService,
		log:               log,
	}
}

func (s *creditService) TakeCredit(ctx context.Context, userID, accountID uuid.UUID, principal string, termMonths int) (*models.Credit, []models.PaymentSchedule, error) {
	acc, err := s.accountRepository.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, nil, err
	}

	principalVal, err := strconv.ParseFloat(principal, 64)
	if err != nil || principalVal <= 0 {
		return nil, nil, apperrors.ErrInvalidAmount
	}
	if termMonths <= 0 || termMonths > 360 {
		return nil, nil, fmt.Errorf("term_months must be between 1 and 360")
	}

	annualRate, err := s.cbrService.GetKeyRate(ctx)
	if err != nil {
		s.log.WithError(err).Warn("CBR unavailable, using default rate 21%")
		annualRate = 21.0
	}
	rateDecimal := annualRate / 100

	monthlyPayment := calcAnnuity(principalVal, rateDecimal, termMonths)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func(tx *sql.Tx) {
		tx.Rollback()
	}(tx)

	credit := &models.Credit{
		AccountID:        accountID,
		Principal:        fmt.Sprintf("%.2f", principalVal),
		InterestRate:     fmt.Sprintf("%.4f", rateDecimal),
		TermMonths:       termMonths,
		MonthlyPayment:   fmt.Sprintf("%.2f", monthlyPayment),
		RemainingBalance: fmt.Sprintf("%.2f", principalVal),
	}
	if err := s.creditRepository.Create(ctx, tx, credit); err != nil {
		return nil, nil, err
	}

	if err := s.accountRepository.UpdateBalance(ctx, tx, accountID, credit.Principal); err != nil {
		return nil, nil, err
	}

	toAccID := accountID
	disbTx := &models.Transaction{
		ToAccountID: &toAccID,
		Amount:      credit.Principal,
		Type:        models.TransactionCreditDisbursement,
		Description: fmt.Sprintf("Выдача кредита %s", credit.ID),
	}
	if err := s.txRepository.Create(ctx, tx, disbTx); err != nil {
		return nil, nil, err
	}

	schedules := make([]models.PaymentSchedule, termMonths)
	remaining := principalVal
	for i := 0; i < termMonths; i++ {
		payment := monthlyPayment
		if i == termMonths-1 {
			payment = remaining + remaining*rateDecimal/12
		}
		schedules[i] = models.PaymentSchedule{
			CreditID: credit.ID,
			DueDate:  time.Now().AddDate(0, i+1, 0),
			Amount:   fmt.Sprintf("%.2f", payment),
			Status:   models.SchedulePending,
		}
		interest := remaining * rateDecimal / 12
		remaining -= (payment - interest)
		if remaining < 0 {
			remaining = 0
		}
	}

	if err := s.schedRepository.CreateBatch(ctx, tx, schedules); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	s.log.WithFields(logrus.Fields{
		"credit_id": credit.ID,
		"principal": principal,
		"term":      termMonths,
		"rate":      annualRate,
		"monthly":   monthlyPayment,
	}).Info("credit issued")

	if user, err := s.userRepository.GetByID(ctx, acc.UserID); err == nil {
		s.emailService.SendCreditNotification(user.Email, user.Username, credit.ID.String(), credit.Principal)
	}

	return credit, schedules, nil
}

func (s *creditService) GetSchedule(ctx context.Context, userID uuid.UUID, creditID uuid.UUID) ([]models.PaymentSchedule, error) {
	_, err := s.creditRepository.GetByIDForUser(ctx, creditID, userID)
	if err != nil {
		return nil, err
	}
	return s.schedRepository.GetByCreditID(ctx, creditID)
}

func calcAnnuity(principal, annualRate float64, termMonths int) float64 {
	r := annualRate / 12
	n := float64(termMonths)
	if r == 0 {
		return principal / n
	}
	factor := math.Pow(1+r, n)
	return principal * (r * factor) / (factor - 1)
}
