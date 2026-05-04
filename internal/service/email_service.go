package service

import (
	"crypto/tls"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

type EmailService interface {
	SendTransactionNotification(to, username, amount, txType string) error
	SendCreditNotification(to, username, creditID, amount string) error
	SendPaymentProcessed(to, username, amount string) error
	SendPaymentOverdue(to, username, amount, penalty string) error
}

type emailService struct {
	host     string
	port     int
	user     string
	password string
	log      *logrus.Logger
}

func NewEmailService(host string, port int, user, password string, log *logrus.Logger) EmailService {
	return &emailService{host: host, port: port, user: user, password: password, log: log}
}

func (s *emailService) send(to, subject, body string) error {
	if s.host == "" {
		s.log.WithField("to", to).Debug("SMTP not configured, skipping email")
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.user)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.host, s.port, s.user, s.password)
	d.TLSConfig = &tls.Config{ServerName: s.host}

	if err := d.DialAndSend(m); err != nil {
		s.log.WithError(err).WithField("to", to).Error("failed to send email")
		return err
	}
	s.log.WithField("to", to).Info("email sent")
	return nil
}

func (s *emailService) SendTransactionNotification(to, username, amount, txType string) error {
	body := fmt.Sprintf(`<h2>Уведомление об операции</h2>
<p>%s,</p>
<p>Выполнена операция: <strong>%s</strong></p>
<p>Сумма: <strong>%s RUB</strong></p>
<small>Это автоматически сгенерированное сообщение, не отвечайте на него.</small>`, username, txType, amount)
	return s.send(to, "Уведомление о операции", body)
}

func (s *emailService) SendCreditNotification(to, username, creditID, amount string) error {
	body := fmt.Sprintf(`<h2>Уведомление об оформлении кредита</h2>
<p>%s,</p>
<p>Кредит <strong>%s</strong> на сумму <strong>%s RUB</strong> оформлен.</p>
<small>Это автоматически сгенерированное сообщение, не отвечайте на него.</small>`, username, creditID, amount)
	return s.send(to, "Кредит оформлен", body)
}

func (s *emailService) SendPaymentProcessed(to, username, amount string) error {
	body := fmt.Sprintf(`<h2>Уведомление о произведении платежа</h2>
<p>%s,</p>
<p>Платеж по кредиту на сумму <strong>%s RUB</strong> произведен.</p>
<small>Это автоматически сгенерированное сообщение, не отвечайте на него.</small>`, username, amount)
	return s.send(to, "Платеж успешно проведен", body)
}

func (s *emailService) SendPaymentOverdue(to, username, amount, penalty string) error {
	body := fmt.Sprintf(`<h2>Уведомление о просрочке платежа по кредиту</h2>
<p>Уважаемый(ая) %s,</p>
<p>Не удалось списать платеж по кредиту на сумму <strong>%s RUB</strong>.</p>
<p>Начислен штраф: <strong>%s RUB</strong>.</p>
<small>Это автоматически сгенерированное сообщение, не отвечайте на него.</small>`, username, amount, penalty)
	return s.send(to, "Просроченный платеж по кредиту", body)
}
