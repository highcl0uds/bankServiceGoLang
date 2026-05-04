package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
)

type AnalyticsResult struct {
	MonthlyIncome  string          `json:"monthly_income"`
	MonthlyExpense string          `json:"monthly_expense"`
	CreditLoad     string          `json:"credit_load"`
	ActiveCredits  int             `json:"active_credits"`
	Credits        []models.Credit `json:"credits"`
}

type ProjectedDebit struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
	Note   string `json:"note"`
}

type PredictionResult struct {
	CurrentBalance   string           `json:"current_balance"`
	PredictedBalance string           `json:"predicted_balance"`
	Days             int              `json:"days"`
	ProjectedDebits  []ProjectedDebit `json:"projected_debits"`
}

type AnalyticsService interface {
	GetAnalytics(ctx context.Context, userID uuid.UUID) (*AnalyticsResult, error)
	GetPrediction(ctx context.Context, userID, accountID uuid.UUID, days int) (*PredictionResult, error)
}

type analyticsService struct {
	accountRepository repository.AccountRepository
	txRepository      repository.TransactionRepository
	creditRepository  repository.CreditRepository
	schedRepository   repository.PaymentScheduleRepository
	log               *logrus.Logger
}

func NewAnalyticsService(
	accountRepository repository.AccountRepository,
	txRepository repository.TransactionRepository,
	creditRepository repository.CreditRepository,
	schedRepository repository.PaymentScheduleRepository,
	log *logrus.Logger,
) AnalyticsService {
	return &analyticsService{
		accountRepository: accountRepository,
		txRepository:      txRepository,
		creditRepository:  creditRepository,
		schedRepository:   schedRepository,
		log:               log,
	}
}

func (s *analyticsService) GetAnalytics(ctx context.Context, userID uuid.UUID) (*AnalyticsResult, error) {
	now := time.Now()
	stats, err := s.txRepository.GetMonthlyStats(ctx, userID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, err
	}

	credits, err := s.creditRepository.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var totalMonthlyLoad float64
	for _, c := range credits {
		mp, _ := strconv.ParseFloat(c.MonthlyPayment, 64)
		totalMonthlyLoad += mp
	}

	return &AnalyticsResult{
		MonthlyIncome:  stats.TotalIncome,
		MonthlyExpense: stats.TotalExpense,
		CreditLoad:     fmt.Sprintf("%.2f", totalMonthlyLoad),
		ActiveCredits:  len(credits),
		Credits:        credits,
	}, nil
}

func (s *analyticsService) GetPrediction(ctx context.Context, userID, accountID uuid.UUID, days int) (*PredictionResult, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	acc, err := s.accountRepository.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	currentBalance, _ := strconv.ParseFloat(acc.Balance, 64)

	credits, err := s.creditRepository.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	horizon := now.AddDate(0, 0, days)
	var totalDebits float64
	var projected []ProjectedDebit

	for _, c := range credits {
		if c.AccountID != accountID {
			continue
		}
		schedules, err := s.schedRepository.GetByCreditID(ctx, c.ID)
		if err != nil {
			continue
		}
		for _, s := range schedules {
			if (s.Status == models.SchedulePending || s.Status == models.ScheduleOverdue) && s.DueDate.After(now) && s.DueDate.Before(horizon) {
				amt, _ := strconv.ParseFloat(s.Amount, 64)
				totalDebits += amt
				projected = append(projected, ProjectedDebit{
					Date:   s.DueDate.Format("2006-01-02"),
					Amount: s.Amount,
					Note:   fmt.Sprintf("Платеж по кредиту %s", c.ID),
				})
			}
		}
	}

	predicted := currentBalance - totalDebits
	return &PredictionResult{
		CurrentBalance:   acc.Balance,
		PredictedBalance: fmt.Sprintf("%.2f", predicted),
		Days:             days,
		ProjectedDebits:  projected,
	}, nil
}
