package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/pkg/apperrors"
)

type TransferService interface {
	Transfer(ctx context.Context, userID uuid.UUID, fromAccountID, toAccountID uuid.UUID, amount string) error
}

type transferService struct {
	db                *sql.DB
	accountRepository repository.AccountRepository
	txRepository      repository.TransactionRepository
	userRepository    repository.UserRepository
	emailService      EmailService
	log               *logrus.Logger
}

func NewTransferService(
	db *sql.DB,
	accountRepository repository.AccountRepository,
	txRepository repository.TransactionRepository,
	userRepository repository.UserRepository,
	emailService EmailService,
	log *logrus.Logger,
) TransferService {
	return &transferService{
		db:                db,
		accountRepository: accountRepository,
		txRepository:      txRepository,
		userRepository:    userRepository,
		emailService:      emailService,
		log:               log,
	}
}

func (s *transferService) Transfer(ctx context.Context, userID uuid.UUID, fromID, toID uuid.UUID, amount string) error {
	if fromID == toID {
		return apperrors.ErrSelfTransfer
	}
	if err := validatePositiveAmount(amount); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ids := []uuid.UUID{fromID, toID}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	acc0, err := s.accountRepository.GetByIDForUpdate(ctx, tx, ids[0])
	if err != nil {
		return err
	}
	acc1, err := s.accountRepository.GetByIDForUpdate(ctx, tx, ids[1])
	if err != nil {
		return err
	}

	var fromAcc *models.Account
	if ids[0] == fromID {
		fromAcc = acc0
	} else {
		fromAcc = acc1
	}

	if fromAcc.UserID != userID {
		return apperrors.ErrForbidden
	}

	bal, _ := strconv.ParseFloat(fromAcc.Balance, 64)
	amt, _ := strconv.ParseFloat(amount, 64)
	if bal < amt {
		return apperrors.ErrInsufficientFunds
	}

	neg := fmt.Sprintf("-%s", amount)
	if err := s.accountRepository.UpdateBalance(ctx, tx, fromID, neg); err != nil {
		return err
	}
	if err := s.accountRepository.UpdateBalance(ctx, tx, toID, amount); err != nil {
		return err
	}

	outTx := &models.Transaction{
		FromAccountID: &fromID,
		ToAccountID:   &toID,
		Amount:        amount,
		Type:          models.TransactionTransferOut,
		Description:   fmt.Sprintf("Перевод на счет %s", toID),
	}
	if err := s.txRepository.Create(ctx, tx, outTx); err != nil {
		return err
	}

	inTx := &models.Transaction{
		FromAccountID: &fromID,
		ToAccountID:   &toID,
		Amount:        amount,
		Type:          models.TransactionTransferIn,
		Description:   fmt.Sprintf("Перевод со счета %s", fromID),
	}
	if err := s.txRepository.Create(ctx, tx, inTx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.log.WithFields(logrus.Fields{"from": fromID, "to": toID, "amount": amount}).Info("transfer completed")

	if user, err := s.userRepository.GetByID(ctx, fromAcc.UserID); err == nil {
		s.emailService.SendTransactionNotification(user.Email, user.Username, amount, "Перевод средств")
	}

	return nil
}
