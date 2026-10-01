// Package contact implementa el formulario de contacto: validacion, correos y handler HTTP.
package contact

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Form struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Company     string `json:"company"`
	ProductType string `json:"product_type"`
	Quantity    string `json:"quantity"`
	Message     string `json:"message"`
	// Website es un honeypot: los usuarios reales nunca lo llenan (campo oculto en el sitio).
	Website string `json:"website"`
}

// Validate normaliza (trim) y valida el formulario. Devuelve errores por campo.
func (f *Form) Validate() map[string]string {
	f.FullName = strings.TrimSpace(f.FullName)
	f.Email = strings.TrimSpace(f.Email)
	f.Phone = strings.TrimSpace(f.Phone)
	f.Company = strings.TrimSpace(f.Company)
	f.ProductType = strings.TrimSpace(f.ProductType)
	f.Quantity = strings.TrimSpace(f.Quantity)
	f.Message = strings.TrimSpace(f.Message)
	f.Website = strings.TrimSpace(f.Website)

	errs := map[string]string{}

	if !between(f.FullName, 2, 100) {
		errs["full_name"] = "debe tener entre 2 y 100 caracteres"
	} else if hasControl(f.FullName) {
		errs["full_name"] = "contiene caracteres no permitidos"
	}

	if addr, err := mail.ParseAddress(f.Email); err != nil || addr.Address != f.Email || len(f.Email) > 254 {
		errs["email"] = "correo electronico invalido"
	}

	if !between(f.Phone, 10, 20) || countDigits(f.Phone) < 7 {
		errs["phone"] = "telefono invalido (10 a 20 caracteres, al menos 7 digitos)"
	}

	optional := map[string]struct {
		value string
		max   int
	}{
		"company":      {f.Company, 255},
		"product_type": {f.ProductType, 100},
		"quantity":     {f.Quantity, 100},
		"website":      {f.Website, 255},
	}
	for name, o := range optional {
		if utf8.RuneCountInString(o.value) > o.max {
			errs[name] = "demasiado largo"
		} else if name != "website" && hasControl(o.value) {
			errs[name] = "contiene caracteres no permitidos"
		}
	}

	if !between(f.Message, 10, 2000) {
		errs["message"] = "debe tener entre 10 y 2000 caracteres"
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func between(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			n++
		}
	}
	return n
}

// hasControl detecta saltos de linea y otros caracteres de control (evita inyeccion en cabeceras/asuntos).
func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
