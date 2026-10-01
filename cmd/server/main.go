// Servidor HTTP local para desarrollo.
package main

import (
	"bufio"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/contact"
	"orioncaps-api/internal/email"
	"orioncaps-api/internal/server"
)

func main() {
	loadDotEnv(".env")
	cfg := config.Load()

	mailer := contact.NewMailer(email.NewMailgun(cfg), cfg)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, mailer),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	slog.Info("escuchando", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("servidor detenido", "error", err)
		os.Exit(1)
	}
}

// loadDotEnv carga KEY=VALUE de un archivo .env sin pisar variables ya definidas.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
