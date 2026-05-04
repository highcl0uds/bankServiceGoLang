package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/pkg/apperrors"
)

type AccountService interface {
	Create(ctx context.Context, userID uuid.UUID) (*models.Account, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]models.Account, error)
	GetByID(ctx context.Context, userID, accountID uuid.UUID) (*models.Account, error)
	Deposit(ctx context.Context, userID, accountID uuid.UUID, amount string) (*models.Account, error)
	Withdraw(ctx context.Context, userID, accountID uuid.UUID, amount string) (*models.Account, error)
}

type accountService struct {
	db                *sql.DB
	accountRepository repository.AccountRepository
	txRepository      repository.TransactionRepository
	userRepository    repository.UserRepository
	emailService      EmailService
	log               *logrus.Logger
}

func NewAccountService(
	db *sql.DB,
	accountRepository repository.AccountRepository,
	txRepository repository.TransactionRepository,
	userRepository repository.UserRepository,
	emailService EmailService,
	log *logrus.Logger,
) AccountService {
	return &accountService{
		db:                db,
		accountRepository: accountRepository,
		txRepository:      txRepository,
		userRepository:    userRepository,
		emailService:      emailService,
		log:               log,
	}
}

func (s *accountService) Create(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	a := &models.Account{
		UserID:   userID,
		Currency: "RUB",
	}
	if err := s.accountRepository.Create(ctx, a); err != nil {
		return nil, err
	}
	s.log.WithFields(logrus.Fields{"user_id": userID, "account_id": a.ID}).Info("account created")
	return a, nil
}

func (s *accountService) GetByUser(ctx context.Context, userID uuid.UUID) ([]models.Account, error) {
	return s.accountRepository.GetByUserID(ctx, userID)
}

func (s *accountService) GetByID(ctx context.Context, userID, accountID uuid.UUID) (*models.Account, error) {
	return s.accountRepository.GetByIDForUser(ctx, accountID, userID)
}

func (s *accountService) Deposit(ctx context.Context, userID, accountID uuid.UUID, amount string) (*models.Account, error) {
	if err := validatePositiveAmount(amount); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	acc, err := s.accountRepository.GetByIDForUpdate(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}
	if acc.UserID != userID {
		return nil, apperrors.ErrForbidden
	}

	if err := s.accountRepository.UpdateBalance(ctx, tx, accountID, amount); err != nil {
		return nil, err
	}

	toAccID := accountID
	t := &models.Transaction{
		ToAccountID: &toAccID,
		Amount:      amount,
		Type:        models.TransactionDeposit,
		Description: fmt.Sprintf("Пополнение счета на %s RUB", amount),
	}
	if err := s.txRepository.Create(ctx, tx, t); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.log.WithFields(logrus.Fields{"account_id": accountID, "amount": amount}).Info("deposit")

	if user, err := s.userRepository.GetByID(ctx, acc.UserID); err == nil {
		s.emailService.SendTransactionNotification(user.Email, user.Username, amount, "Пополнение счета")
	}

	return s.accountRepository.GetByIDForUser(ctx, accountID, userID)
}

func (s *accountService) Withdraw(ctx context.Context, userID, accountID uuid.UUID, amount string) (*models.Account, error) {
	if err := validatePositiveAmount(amount); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	acc, err := s.accountRepository.GetByIDForUpdate(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}
	if acc.UserID != userID {
		return nil, apperrors.ErrForbidden
	}

	bal, _ := strconv.ParseFloat(acc.Balance, 64)
	amt, _ := strconv.ParseFloat(amount, 64)
	if bal < amt {
		return nil, apperrors.ErrInsufficientFunds
	}

	neg := fmt.Sprintf("-%s", amount)
	if err := s.accountRepository.UpdateBalance(ctx, tx, accountID, neg); err != nil {
		return nil, err
	}

	fromAccID := accountID
	t := &models.Transaction{
		FromAccountID: &fromAccID,
		Amount:        amount,
		Type:          models.TransactionWithdrawal,
		Description:   fmt.Sprintf("Снятие со счета %s RUB", amount),
	}
	if err := s.txRepository.Create(ctx, tx, t); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.log.WithFields(logrus.Fields{"account_id": accountID, "amount": amount}).Info("withdrawal")

	if user, err := s.userRepository.GetByID(ctx, acc.UserID); err == nil {
		s.emailService.SendTransactionNotification(user.Email, user.Username, amount, "Снятие со счета")
	}

	return s.accountRepository.GetByIDForUser(ctx, accountID, userID)
}

func validatePositiveAmount(amount string) error {
	v, err := strconv.ParseFloat(amount, 64)
	if err != nil || v <= 0 {
		return apperrors.ErrInvalidAmount
	}
	return nil
}
