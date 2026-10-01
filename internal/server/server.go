// Package server arma el http.Handler de la API (rutas, CORS, recover).
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/contact"
)

// New devuelve el handler completo de la API. Se usa tanto en el servidor local como en Lambda.
func New(cfg config.Config, notifier contact.Notifier) http.Handler {
	mux := http.NewServeMux()
	h := contact.NewHandler(cfg, notifier)

	mux.HandleFunc("POST /api/v1/contact", h.Submit)
	mux.HandleFunc("POST /api/v1/contact/", h.Submit)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": cfg.AppName,
			"version": cfg.Version,
		})
	})

	return recoverer(cors(cfg.AllowedOrigins, noStore(mux)))
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func cors(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(allowed, origin) || slices.Contains(allowed, "*")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join([]string{"GET", "POST", "OPTIONS"}, ", "))
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-api-key")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		} else if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent) // origen no permitido: sin cabeceras CORS
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic no controlado", "path", r.URL.Path, "panic", rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"success":false,"message":"Error interno del servidor"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
