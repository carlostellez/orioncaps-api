package contact

import (
	"strings"
	"testing"
)

func validForm() Form {
	return Form{
		FullName: "Juan Pérez", Email: "juan@example.com", Phone: "+52 123 456 7890",
		Company: "Empresa S.A.", ProductType: "Gorras", Quantity: "500",
		Message: "Quiero cotizar una orden de gorras bordadas.",
	}
}

func TestValidFormAndTrim(t *testing.T) {
	f := validForm()
	f.FullName = "  Juan Pérez  "
	if errs := f.Validate(); errs != nil {
		t.Fatalf("errs: %v", errs)
	}
	if f.FullName != "Juan Pérez" {
		t.Fatalf("no se hizo trim: %q", f.FullName)
	}
}

func TestOptionalFieldsCanBeEmpty(t *testing.T) {
	f := validForm()
	f.Company, f.ProductType, f.Quantity = "", "", ""
	if errs := f.Validate(); errs != nil {
		t.Fatalf("errs: %v", errs)
	}
}

func TestInvalidFields(t *testing.T) {
	cases := map[string]func(*Form){
		"full_name":    func(f *Form) { f.FullName = "J" },
		"email":        func(f *Form) { f.Email = "no-es-email" },
		"phone":        func(f *Form) { f.Phone = "abc-def-ghij" },
		"message":      func(f *Form) { f.Message = "corto" },
		"company":      func(f *Form) { f.Company = strings.Repeat("x", 256) },
		"quantity":     func(f *Form) { f.Quantity = strings.Repeat("x", 101) },
		"product_type": func(f *Form) { f.ProductType = "a\nb" },
	}
	for field, mutate := range cases {
		f := validForm()
		mutate(&f)
		if errs := f.Validate(); errs[field] == "" {
			t.Errorf("%s: se esperaba error, got %v", field, errs)
		}
	}
}

func TestEmailWithDisplayNameRejected(t *testing.T) {
	f := validForm()
	f.Email = "Juan <juan@example.com>"
	if errs := f.Validate(); errs["email"] == "" {
		t.Fatal("deberia rechazar nombre + direccion")
	}
}

func TestHeaderInjectionInName(t *testing.T) {
	f := validForm()
	f.FullName = "Juan\r\nBcc: victima@x.com"
	if errs := f.Validate(); errs["full_name"] == "" {
		t.Fatal("deberia rechazar saltos de linea")
	}
}

func TestMessageLimitCountsRunes(t *testing.T) {
	f := validForm()
	f.Message = strings.Repeat("ñ", 2000)
	if errs := f.Validate(); errs != nil {
		t.Fatalf("2000 runas deberian ser validas: %v", errs)
	}
	f.Message = strings.Repeat("ñ", 2001)
	if errs := f.Validate(); errs["message"] == "" {
		t.Fatal("2001 runas deberian fallar")
	}
}
