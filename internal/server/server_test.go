package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/contact"
	"orioncaps-api/internal/email"
)

type fakeNotifier struct {
	calls []contact.Form
	err   error
}

func (n *fakeNotifier) SendContact(_ context.Context, f contact.Form) error {
	n.calls = append(n.calls, f)
	return n.err
}

func cfg() config.Config {
	return config.Config{
		AppName: "Orion Caps API", Version: "test",
		MailgunAPIKey: "k", MailgunDomain: "mg.test.com",
		AdminEmails:    []string{"admin@test.com"},
		AllowedOrigins: []string{"http://localhost:3000"},
	}
}

const validBody = `{"full_name":"Juan Pérez","email":"juan@example.com","phone":"+52 123 456 7890",
"company":"Empresa","product_type":"Gorras","quantity":"500","message":"Quiero cotizar gorras bordadas."}`

func do(h http.Handler, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSuccess(t *testing.T) {
	n := &fakeNotifier{}
	rec := do(New(cfg(), n), "POST", "/api/v1/contact", validBody)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"success":true`) || len(n.calls) != 1 {
		t.Fatalf("code=%d body=%s calls=%d", rec.Code, rec.Body, len(n.calls))
	}
}

func TestTrailingSlash(t *testing.T) {
	if rec := do(New(cfg(), &fakeNotifier{}), "POST", "/api/v1/contact/", validBody); rec.Code != 200 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestValidationError422(t *testing.T) {
	n := &fakeNotifier{}
	body := strings.Replace(validBody, "juan@example.com", "mal", 1)
	rec := do(New(cfg(), n), "POST", "/api/v1/contact", body)
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), `"email"`) || len(n.calls) != 0 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body)
	}
}

func TestInvalidJSON400(t *testing.T) {
	if rec := do(New(cfg(), &fakeNotifier{}), "POST", "/api/v1/contact", "{no-json"); rec.Code != 400 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestBodyTooLarge(t *testing.T) {
	big := `{"message":"` + strings.Repeat("a", 70<<10) + `"}`
	if rec := do(New(cfg(), &fakeNotifier{}), "POST", "/api/v1/contact", big); rec.Code != 400 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestHoneypotSkipsSending(t *testing.T) {
	n := &fakeNotifier{}
	body := strings.Replace(validBody, `"company"`, `"website":"http://spam.com","company"`, 1)
	rec := do(New(cfg(), n), "POST", "/api/v1/contact", body)
	if rec.Code != 200 || len(n.calls) != 0 {
		t.Fatalf("code=%d calls=%d", rec.Code, len(n.calls))
	}
}

func TestProviderFailure502(t *testing.T) {
	n := &fakeNotifier{err: fmt.Errorf("%w: secreto interno", email.ErrDelivery)}
	rec := do(New(cfg(), n), "POST", "/api/v1/contact", validBody)
	if rec.Code != 502 || strings.Contains(rec.Body.String(), "secreto") {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body)
	}
}

func TestUnexpectedError500IsGeneric(t *testing.T) {
	n := &fakeNotifier{err: errors.New("detalle sensible")}
	rec := do(New(cfg(), n), "POST", "/api/v1/contact", validBody)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "sensible") {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body)
	}
}

func TestNotConfigured500(t *testing.T) {
	c := cfg()
	c.MailgunAPIKey = ""
	n := &fakeNotifier{}
	rec := do(New(c, n), "POST", "/api/v1/contact", validBody)
	if rec.Code != 500 || len(n.calls) != 0 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestHealthAndMethodNotAllowed(t *testing.T) {
	h := New(cfg(), &fakeNotifier{})
	if rec := do(h, "GET", "/health", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "healthy") {
		t.Fatalf("health: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/api/v1/contact", ""); rec.Code != 405 {
		t.Fatalf("GET contact: %d", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	h := New(cfg(), &fakeNotifier{})

	rec := do(h, "OPTIONS", "/api/v1/contact", "",
		"Origin", "http://localhost:3000", "Access-Control-Request-Method", "POST")
	if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("preflight permitido: %d %v", rec.Code, rec.Header())
	}

	rec = do(h, "OPTIONS", "/api/v1/contact", "",
		"Origin", "http://evil.com", "Access-Control-Request-Method", "POST")
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("origen no permitido no debe recibir cabeceras CORS")
	}

	rec = do(h, "POST", "/api/v1/contact", validBody, "Origin", "http://localhost:3000")
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("la respuesta real debe incluir CORS")
	}
}

type panicNotifier struct{}

func (panicNotifier) SendContact(context.Context, contact.Form) error { panic("boom") }

func TestRecoverer(t *testing.T) {
	rec := do(New(cfg(), panicNotifier{}), "POST", "/api/v1/contact", validBody)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body)
	}
}
