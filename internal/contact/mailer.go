package contact

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"strings"
	"time"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/email"
)

//go:embed templates/*.html
var templatesFS embed.FS

// html/template aplica autoescape contextual a todos los campos del usuario.
var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

const previewRunes = 200

type templateData struct {
	Form      Form
	Brand     string
	When      string
	Preview   string
	Truncated bool
}

type Mailer struct {
	provider email.Provider
	cfg      config.Config
	now      func() time.Time
}

func NewMailer(p email.Provider, cfg config.Config) *Mailer {
	return &Mailer{provider: p, cfg: cfg, now: time.Now}
}

// SendContact notifica al admin (obligatorio) y confirma al cliente (best-effort).
func (m *Mailer) SendContact(ctx context.Context, f Form) error {
	when := m.now().Format("02/01/2006 15:04")
	data := templateData{Form: f, Brand: m.cfg.BrandName, When: when}
	if runes := []rune(f.Message); len(runes) > previewRunes {
		data.Preview, data.Truncated = string(runes[:previewRunes]), true
	} else {
		data.Preview = f.Message
	}

	adminHTML, err := render("contact_admin.html", data)
	if err != nil {
		return err
	}
	customerHTML, err := render("contact_customer.html", data)
	if err != nil {
		return err
	}

	admin := email.Message{
		To:      m.cfg.AdminEmails,
		Subject: "Nuevo contacto de " + f.FullName,
		HTML:    adminHTML,
		Text:    adminText(f, when),
		ReplyTo: f.Email,
		Tags:    []string{"contact-admin"},
	}
	if err := m.provider.Send(ctx, admin); err != nil {
		return err // el correo al admin es indispensable
	}

	customer := email.Message{
		To:      []string{f.Email},
		Subject: "Gracias por contactarnos - " + m.cfg.BrandName,
		HTML:    customerHTML,
		Text: fmt.Sprintf("Hola %s,\n\nRecibimos tu mensaje y te responderemos pronto.\n\nSaludos,\nEl equipo de %s",
			f.FullName, m.cfg.BrandName),
		Tags: []string{"contact-customer"},
	}
	if err := m.provider.Send(ctx, customer); err != nil {
		if errors.Is(err, email.ErrDelivery) {
			slog.Warn("no se pudo enviar la confirmacion al cliente")
			return nil
		}
		return err
	}
	return nil
}

func render(name string, data templateData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}

func adminText(f Form, when string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Nombre: %s\nEmail: %s\nTelefono: %s\n", f.FullName, f.Email, f.Phone)
	if f.Company != "" {
		fmt.Fprintf(&b, "Empresa: %s\n", f.Company)
	}
	if f.ProductType != "" {
		fmt.Fprintf(&b, "Producto: %s\n", f.ProductType)
	}
	if f.Quantity != "" {
		fmt.Fprintf(&b, "Cantidad: %s\n", f.Quantity)
	}
	fmt.Fprintf(&b, "\nMensaje:\n%s\n\nFecha: %s", f.Message, when)
	return b.String()
}
