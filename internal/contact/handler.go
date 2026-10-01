package contact

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"orioncaps-api/internal/config"
	"orioncaps-api/internal/email"
)

const (
	successMessage = "Mensaje enviado exitosamente. Te responderemos pronto."
	maxBodyBytes   = 64 << 10
)

// Notifier envia las notificaciones de un formulario (lo implementa *Mailer).
type Notifier interface {
	SendContact(ctx context.Context, f Form) error
}

type Handler struct {
	cfg      config.Config
	notifier Notifier
}

func NewHandler(cfg config.Config, n Notifier) *Handler {
	return &Handler{cfg: cfg, notifier: n}
}

type response struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var form Form
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Message: "JSON invalido"})
		return
	}

	if errs := form.Validate(); errs != nil {
		writeJSON(w, http.StatusUnprocessableEntity, response{Message: "Datos invalidos", Errors: errs})
		return
	}

	if form.Website != "" { // honeypot: respondemos igual para no alertar al bot
		slog.Info("honeypot activado; envio omitido")
		writeJSON(w, http.StatusOK, response{Success: true, Message: successMessage})
		return
	}

	if !h.cfg.EmailConfigured() {
		slog.Error("servicio de correo no configurado")
		writeJSON(w, http.StatusInternalServerError, response{Message: "Servicio de correo no configurado"})
		return
	}

	if err := h.notifier.SendContact(r.Context(), form); err != nil {
		slog.Error("fallo el envio del correo de contacto", "error", err)
		if errors.Is(err, email.ErrDelivery) {
			writeJSON(w, http.StatusBadGateway, response{Message: "No pudimos enviar tu mensaje. Intenta de nuevo mas tarde."})
			return
		}
		writeJSON(w, http.StatusInternalServerError, response{Message: "Error interno del servidor"})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: successMessage})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
