package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/env"
)

const mailtrapAPIURL = "https://send.api.mailtrap.io/api/send"

type mailtrapProvider struct {
	apiToken    string
	senderEmail string
	httpClient  *http.Client
}

func NewMailtrapProvider() (domain.EmailProvider, error) {
	token := env.GetString("MAILTRAP_API_TOKEN", "b37a230f5c8ceb229308f43093aa999f")
	sender := env.GetString("MAILTRAP_SENDER_EMAIL", "")

	if token == "" || sender == "" {
		return nil, errors.New("MAILTRAP_API_TOKEN và MAILTRAP_SENDER_EMAIL là bắt buộc")
	}

	return &mailtrapProvider{
		apiToken:    token,
		senderEmail: sender,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

type mailtrapEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type mailtrapPayload struct {
	From mailtrapEmail   `json:"from"`
	To   []mailtrapEmail `json:"to"`
	Subject string        `json:"subject"`
	HTML    string        `json:"html"`
}

func (p *mailtrapProvider) SendEmail(ctx context.Context, toEmail string, subject string, htmlBody string) error {
	payload := mailtrapPayload{
		From: mailtrapEmail{
			Email: p.senderEmail,
			Name:  "JQK Study",
		},
		To: []mailtrapEmail{
			{Email: toEmail},
		},
		Subject: subject,
		HTML:    htmlBody,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("lỗi marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", mailtrapAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("lỗi tạo http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("lỗi gửi request đến Mailtrap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Mailtrap trả về lỗi: %s", resp.Status)
	}

	return nil
}