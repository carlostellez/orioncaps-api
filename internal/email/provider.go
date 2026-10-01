// Package email define la abstraccion del proveedor de correo y su implementacion Mailgun.
package email

import (
	"context"
	"errors"
)

// ErrDelivery indica que el proveedor no pudo entregar el correo.
var ErrDelivery = errors.New("email delivery failed")

type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
	ReplyTo string
	From    string // vacio = remitente por defecto
	Tags    []string
}

// Provider envia un mensaje o devuelve un error que envuelve ErrDelivery.
type Provider interface {
	Send(ctx context.Context, m Message) error
}
