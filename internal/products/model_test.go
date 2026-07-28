package products

import (
	"testing"
)

func TestProduct_DisplayValue_Name(t *testing.T) {
	p := Product{Name: "Test Product"}
	v, err := p.DisplayValue("Name")
	if err != nil {
		t.Fatal(err)
	}
	if v != "Test Product" {
		t.Fatalf("expected 'Test Product', got %q", v)
	}
}

func TestProduct_DisplayValue_OrganizationName(t *testing.T) {
	p := Product{OrganizationName: "ООО Ромашка"}
	v, err := p.DisplayValue("OrganizationName")
	if err != nil {
		t.Fatal(err)
	}
	if v != "ООО Ромашка" {
		t.Fatalf("expected 'ООО Ромашка', got %q", v)
	}
}

func TestProduct_DisplayValue_Unit(t *testing.T) {
	p := Product{Unit: "шт"}
	v, err := p.DisplayValue("Unit")
	if err != nil {
		t.Fatal(err)
	}
	if v != "шт" {
		t.Fatalf("expected 'шт', got %q", v)
	}
}

func TestProduct_DisplayValue_Active(t *testing.T) {
	p := Product{Active: true}
	v, err := p.DisplayValue("Active")
	if err != nil {
		t.Fatal(err)
	}
	if v != "Активен" {
		t.Fatalf("expected 'Активен', got %q", v)
	}

	p = Product{Active: false}
	v, err = p.DisplayValue("Active")
	if err != nil {
		t.Fatal(err)
	}
	if v != "Неактивен" {
		t.Fatalf("expected 'Неактивен', got %q", v)
	}
}

func TestProduct_DisplayValue_UnknownField(t *testing.T) {
	p := Product{}
	_, err := p.DisplayValue("NonExistent")
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestProduct_DisplayValue_ImplementsInterface(t *testing.T) {
	var _ interface{ DisplayValue(string) (string, error) } = Product{}
}
