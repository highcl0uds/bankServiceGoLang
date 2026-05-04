package service

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"bank-service-cbr/internal/models"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/pkg/crypto"
)

type CardService interface {
	IssueCard(ctx context.Context, userID, accountID uuid.UUID) (*models.CardResponse, error)
	GetCards(ctx context.Context, userID uuid.UUID) ([]models.CardResponse, error)
}

type cardService struct {
	cardRepository    repository.CardRepository
	accountRepository repository.AccountRepository
	pgpPubKey         string
	pgpPrivKey        string
	hmacSecret        []byte
	log               *logrus.Logger
}

func NewCardService(
	cardRepository repository.CardRepository,
	accountRepository repository.AccountRepository,
	pgpPubKey, pgpPrivKey string,
	hmacSecret []byte,
	log *logrus.Logger,
) CardService {
	return &cardService{
		cardRepository:    cardRepository,
		accountRepository: accountRepository,
		pgpPubKey:         pgpPubKey,
		pgpPrivKey:        pgpPrivKey,
		hmacSecret:        hmacSecret,
		log:               log,
	}
}

func (s *cardService) IssueCard(ctx context.Context, userID, accountID uuid.UUID) (*models.CardResponse, error) {
	acc, err := s.accountRepository.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	var number string
	for {
		n, err := crypto.GenerateCardNumber("4532")
		if err != nil {
			return nil, fmt.Errorf("generate card number: %w", err)
		}
		mac := crypto.ComputeHMAC(n, s.hmacSecret)
		exists, err := s.cardRepository.ExistsWithHMAC(ctx, mac)
		if err != nil {
			return nil, err
		}
		if !exists {
			number = n
			break
		}
	}

	expiry := time.Now().AddDate(3, 0, 0).Format("01/06")

	cvv, err := randomCVV()
	if err != nil {
		return nil, fmt.Errorf("generate cvv: %w", err)
	}

	encNumber, err := crypto.EncryptPGPArmored(number, s.pgpPubKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt card number: %w", err)
	}
	encExpiry, err := crypto.EncryptPGPArmored(expiry, s.pgpPubKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt expiry: %w", err)
	}

	cvvHash, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	mac := crypto.ComputeHMAC(number, s.hmacSecret)

	card := &models.Card{
		AccountID:       acc.ID,
		NumberEncrypted: encNumber,
		NumberHMAC:      mac,
		ExpiryEncrypted: encExpiry,
		CVVHash:         string(cvvHash),
	}
	if err := s.cardRepository.Create(ctx, card); err != nil {
		return nil, err
	}

	s.log.WithFields(logrus.Fields{"card_id": card.ID, "account_id": accountID}).Info("card issued")

	return &models.CardResponse{
		ID:        card.ID,
		AccountID: card.AccountID,
		Number:    formatCardNumber(number),
		Expiry:    expiry,
		CreatedAt: card.CreatedAt,
	}, nil
}

func (s *cardService) GetCards(ctx context.Context, userID uuid.UUID) ([]models.CardResponse, error) {
	if s.pgpPrivKey == "" {
		return nil, fmt.Errorf("PGP private key not configured")
	}

	cards, err := s.cardRepository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.CardResponse, 0, len(cards))
	for _, c := range cards {
		number, err := crypto.DecryptPGPArmored(c.NumberEncrypted, s.pgpPrivKey)
		if err != nil {
			s.log.WithError(err).Warn("failed to decrypt card number")
			continue
		}
		if !crypto.VerifyHMAC(number, c.NumberHMAC, s.hmacSecret) {
			s.log.WithField("card_id", c.ID).Error("card HMAC verification failed: data integrity compromised")
			continue
		}
		expiry, err := crypto.DecryptPGPArmored(c.ExpiryEncrypted, s.pgpPrivKey)
		if err != nil {
			s.log.WithError(err).Warn("failed to decrypt card expiry")
			continue
		}
		result = append(result, models.CardResponse{
			ID:        c.ID,
			AccountID: c.AccountID,
			Number:    formatCardNumber(number),
			Expiry:    expiry,
			CreatedAt: c.CreatedAt,
		})
	}
	return result, nil
}

func formatCardNumber(n string) string {
	if len(n) != 16 {
		return n
	}
	return fmt.Sprintf("%s %s %s %s", n[0:4], n[4:8], n[8:12], n[12:16])
}

func randomCVV() (string, error) {
	n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(1000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%03d", n.Int64()), nil
}
