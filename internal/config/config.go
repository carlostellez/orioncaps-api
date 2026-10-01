// Package config carga la configuracion desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName   string
	Version   string
	BrandName string
	Debug     bool
	Port      string

	MailgunAPIKey  string
	MailgunDomain  string
	MailgunBaseURL string
	MailgunTimeout time.Duration
	MailgunRetries int
	FromEmail      string

	AdminEmails    []string
	AllowedOrigins []string
}

// Load lee la configuracion del entorno del proceso.
func Load() Config { return FromEnv(os.Getenv) }

// FromEnv construye la configuracion a partir de una funcion de lookup (util en tests).
func FromEnv(get func(string) string) Config {
	str := func(key, def string) string {
		if v := strings.TrimSpace(get(key)); v != "" {
			return v
		}
		return def
	}

	timeout, err := time.ParseDuration(str("MAILGUN_TIMEOUT", "10s"))
	if err != nil || timeout <= 0 {
		timeout = 10 * time.Second
	}
	retries, err := strconv.Atoi(str("MAILGUN_RETRIES", "2"))
	if err != nil || retries < 0 {
		retries = 2
	}
	debug, _ := strconv.ParseBool(str("DEBUG", "false"))

	return Config{
		AppName:        str("APP_NAME", "Orion Caps API"),
		Version:        str("APP_VERSION", "0.1.0"),
		BrandName:      str("BRAND_NAME", "Orion Caps"),
		Debug:          debug,
		Port:           str("PORT", "8080"),
		MailgunAPIKey:  str("MAILGUN_API_KEY", ""),
		MailgunDomain:  str("MAILGUN_DOMAIN", ""),
		MailgunBaseURL: strings.TrimRight(str("MAILGUN_BASE_URL", "https://api.mailgun.net/v3"), "/"),
		MailgunTimeout: timeout,
		MailgunRetries: retries,
		FromEmail:      str("FROM_EMAIL", ""),
		AdminEmails:    splitList(get("ADMIN_EMAIL")),
		AllowedOrigins: splitList(str("ALLOWED_ORIGINS", "http://localhost:3000")),
	}
}

// Sender devuelve el remitente por defecto.
func (c Config) Sender() string {
	if c.FromEmail != "" {
		return c.FromEmail
	}
	return fmt.Sprintf("%s <no-reply@%s>", c.BrandName, c.MailgunDomain)
}

// EmailConfigured indica si hay lo minimo para enviar correos.
func (c Config) EmailConfigured() bool {
	return c.MailgunAPIKey != "" && c.MailgunDomain != "" && len(c.AdminEmails) > 0
}

func splitList(v string) []string {
	var out []string
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
