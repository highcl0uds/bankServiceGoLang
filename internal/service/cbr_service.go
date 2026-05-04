package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beevik/etree"
	"github.com/sirupsen/logrus"
)

type CBRService interface {
	GetKeyRate(ctx context.Context) (float64, error)
}

type cbrService struct {
	httpClient *http.Client
	log        *logrus.Logger
}

func NewCBRService(log *logrus.Logger) CBRService {
	return &cbrService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        log,
	}
}

func (s *cbrService) GetKeyRate(ctx context.Context) (float64, error) {
	soap := s.buildSOAPRequest()
	body, err := s.sendRequest(ctx, soap)
	if err != nil {
		return 0, fmt.Errorf("cbr request: %w", err)
	}
	rate, err := s.parseResponse(body)
	if err != nil {
		return 0, fmt.Errorf("cbr parse: %w", err)
	}
	rate += 5
	s.log.WithField("rate", rate).Info("fetched CBR key rate")
	return rate, nil
}

func (s *cbrService) buildSOAPRequest() string {
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <KeyRate xmlns="http://web.cbr.ru/">
      <fromDate>%s</fromDate>
      <ToDate>%s</ToDate>
    </KeyRate>
  </soap12:Body>
</soap12:Envelope>`, from, to)
}

func (s *cbrService) sendRequest(ctx context.Context, soapBody string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx",
		bytes.NewBufferString(soapBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (s *cbrService) parseResponse(body []byte) (float64, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return 0, fmt.Errorf("parse xml: %w", err)
	}

	elements := doc.FindElements("//KeyRate/KR/Rate")
	if len(elements) == 0 {
		elements = doc.FindElements("//KR/Rate")
	}
	if len(elements) == 0 {
		return 0, errors.New("rate element not found in CBR response")
	}

	rateStr := elements[len(elements)-1].Text()
	var rate float64
	if _, err := fmt.Sscanf(rateStr, "%f", &rate); err != nil {
		return 0, fmt.Errorf("parse rate value %q: %w", rateStr, err)
	}
	return rate, nil
}
