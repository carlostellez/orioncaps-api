package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/email"
)

type fakeProvider struct {
	sent   []email.Message
	failOn map[int]error
}

func (p *fakeProvider) Send(_ context.Context, m email.Message) error {
	i := len(p.sent)
	p.sent = append(p.sent, m)
	return p.failOn[i]
}

func testCfg() config.Config {
	return config.Config{BrandName: "Orion Caps", AdminEmails: []string{"admin@test.com", "ventas@test.com"}}
}

func TestSendsAdminAndCustomer(t *testing.T) {
	p := &fakeProvider{}
	if err := NewMailer(p, testCfg()).SendContact(context.Background(), validForm()); err != nil {
		t.Fatal(err)
	}
	if len(p.sent) != 2 {
		t.Fatalf("enviados: %d", len(p.sent))
	}
	admin, customer := p.sent[0], p.sent[1]
	if len(admin.To) != 2 || admin.ReplyTo != "juan@example.com" {
		t.Errorf("admin: %+v", admin)
	}
	if customer.To[0] != "juan@example.com" || !strings.Contains(customer.Subject, "Orion Caps") {
		t.Errorf("customer: %+v", customer)
	}
	if !strings.Contains(admin.Text, "Gorras") || !strings.Contains(admin.HTML, "Empresa S.A.") {
		t.Errorf("faltan datos en el correo del admin")
	}
}

func TestHTMLIsEscaped(t *testing.T) {
	f := validForm()
	f.FullName = "<script>alert(1)</script>"
	f.Message = "<img src=x onerror=alert(1)> mensaje largo"
	p := &fakeProvider{}
	if err := NewMailer(p, testCfg()).SendContact(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	for _, m := range p.sent {
		if strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, "<img") {
			t.Errorf("HTML sin escapar: %s", m.HTML)
		}
	}
	if !strings.Contains(p.sent[0].HTML, "&lt;script&gt;") {
		t.Error("deberia contener el texto escapado")
	}
}

func TestAdminFailurePropagates(t *testing.T) {
	boom := fmt.Errorf("%w: boom", email.ErrDelivery)
	p := &fakeProvider{failOn: map[int]error{0: boom}}
	err := NewMailer(p, testCfg()).SendContact(context.Background(), validForm())
	if !errors.Is(err, email.ErrDelivery) {
		t.Fatalf("err=%v", err)
	}
	if len(p.sent) != 1 {
		t.Fatal("no debe intentar la confirmacion si falla el admin")
	}
}

func TestCustomerDeliveryFailureIsSwallowed(t *testing.T) {
	p := &fakeProvider{failOn: map[int]error{1: fmt.Errorf("%w: boom", email.ErrDelivery)}}
	if err := NewMailer(p, testCfg()).SendContact(context.Background(), validForm()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestCustomerUnexpectedErrorPropagates(t *testing.T) {
	p := &fakeProvider{failOn: map[int]error{1: errors.New("otro")}}
	if err := NewMailer(p, testCfg()).SendContact(context.Background(), validForm()); err == nil {
		t.Fatal("se esperaba error")
	}
}

func TestLongMessageTruncatedForCustomer(t *testing.T) {
	f := validForm()
	f.Message = strings.Repeat("a", 500)
	p := &fakeProvider{}
	if err := NewMailer(p, testCfg()).SendContact(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	html := p.sent[1].HTML
	if !strings.Contains(html, "...") || strings.Contains(html, strings.Repeat("a", 201)) {
		t.Error("el mensaje deberia truncarse a 200 caracteres")
	}
}
