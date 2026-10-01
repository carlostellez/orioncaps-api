package config

import (
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDefaults(t *testing.T) {
	c := FromEnv(env(nil))
	if c.BrandName != "Orion Caps" || c.Port != "8080" || c.MailgunRetries != 2 || c.MailgunTimeout != 10*time.Second {
		t.Fatalf("defaults inesperados: %+v", c)
	}
	if c.EmailConfigured() {
		t.Fatal("no deberia estar configurado")
	}
}

func TestListsAndSender(t *testing.T) {
	c := FromEnv(env(map[string]string{
		"ADMIN_EMAIL":     "a@x.com, b@x.com",
		"ALLOWED_ORIGINS": "http://a.com,http://b.com",
		"MAILGUN_DOMAIN":  "mg.x.com",
		"MAILGUN_API_KEY": "k",
	}))
	if len(c.AdminEmails) != 2 || c.AdminEmails[1] != "b@x.com" {
		t.Fatalf("admins: %v", c.AdminEmails)
	}
	if len(c.AllowedOrigins) != 2 {
		t.Fatalf("origins: %v", c.AllowedOrigins)
	}
	if got := c.Sender(); got != "Orion Caps <no-reply@mg.x.com>" {
		t.Fatalf("sender: %s", got)
	}
	if !c.EmailConfigured() {
		t.Fatal("deberia estar configurado")
	}
}

func TestExplicitFromAndInvalidValues(t *testing.T) {
	c := FromEnv(env(map[string]string{
		"FROM_EMAIL":      "X <x@x.com>",
		"MAILGUN_TIMEOUT": "nope",
		"MAILGUN_RETRIES": "-3",
		"DEBUG":           "true",
	}))
	if c.Sender() != "X <x@x.com>" || c.MailgunTimeout != 10*time.Second || c.MailgunRetries != 2 || !c.Debug {
		t.Fatalf("config: %+v", c)
	}
}
