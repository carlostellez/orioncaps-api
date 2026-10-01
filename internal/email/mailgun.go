package email

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"orioncaps-api/internal/config"
)

type Mailgun struct {
	apiKey  string
	domain  string
	baseURL string
	sender  string
	retries int
	backoff time.Duration
	client  *http.Client
}

func NewMailgun(cfg config.Config) *Mailgun {
	return &Mailgun{
		apiKey:  cfg.MailgunAPIKey,
		domain:  cfg.MailgunDomain,
		baseURL: cfg.MailgunBaseURL,
		sender:  cfg.Sender(),
		retries: cfg.MailgunRetries,
		backoff: 300 * time.Millisecond,
		client:  &http.Client{Timeout: cfg.MailgunTimeout},
	}
}

func (m *Mailgun) Send(ctx context.Context, msg Message) error {
	if m.apiKey == "" || m.domain == "" {
		return fmt.Errorf("%w: mailgun no esta configurado", ErrDelivery)
	}

	form := url.Values{}
	from := msg.From
	if from == "" {
		from = m.sender
	}
	form.Set("from", from)
	for _, to := range msg.To {
		form.Add("to", to)
	}
	form.Set("subject", msg.Subject)
	form.Set("text", msg.Text)
	form.Set("html", msg.HTML)
	if msg.ReplyTo != "" {
		form.Set("h:Reply-To", msg.ReplyTo)
	}
	for _, tag := range msg.Tags {
		form.Add("o:tag", tag)
	}
	body := form.Encode()
	endpoint := fmt.Sprintf("%s/%s/messages", m.baseURL, m.domain)

	attempts := m.retries + 1
	lastErr := "desconocido"
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("%w: %v", ErrDelivery, ctx.Err())
			case <-time.After(m.backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDelivery, err)
		}
		req.SetBasicAuth("api", m.apiKey)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = "error de red"
			slog.Warn("mailgun fallo", "attempt", attempt, "of", attempts, "reason", lastErr)
			if ctx.Err() != nil {
				break
			}
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		slog.Warn("mailgun fallo", "attempt", attempt, "of", attempts, "reason", lastErr)
		if resp.StatusCode < 500 {
			break // error del cliente: reintentar no ayuda
		}
	}
	return fmt.Errorf("%w: %s", ErrDelivery, lastErr)
}
