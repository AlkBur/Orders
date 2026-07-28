package entity

import (
	"testing"
)

type testEntity struct {
	ID     string `db:"id" label:"ID" order:"10" list:"true"`
	Name   string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
	Hidden string `db:"hidden" label:"Hidden" order:"30"`
}

type noDBTag struct {
	Name string
}

func TestRegister(t *testing.T) {
	desc := Register[testEntity]()

	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}

	if len(desc.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(desc.Fields))
	}

	if desc.Fields[0].GoName != "ID" || desc.Fields[1].GoName != "Name" || desc.Fields[2].GoName != "Hidden" {
		t.Fatal("unexpected field order")
	}
}

func TestRegister_ReturnsCached(t *testing.T) {
	d1 := Register[testEntity]()
	d2 := Register[testEntity]()

	if d1 != d2 {
		t.Fatal("expected the same pointer for cached descriptor")
	}
}

func TestListFields(t *testing.T) {
	desc := Register[testEntity]()
	fields := desc.ListFields()

	if len(fields) != 2 {
		t.Fatalf("expected 2 list fields, got %d", len(fields))
	}

	if fields[0].GoName != "ID" || fields[1].GoName != "Name" {
		t.Fatal("unexpected list field order")
	}

	if !fields[0].List || !fields[1].List {
		t.Fatal("expected List=true for both fields")
	}
}

func TestSearchFields(t *testing.T) {
	desc := Register[testEntity]()
	fields := desc.SearchFields()

	if len(fields) != 1 {
		t.Fatalf("expected 1 search field, got %d", len(fields))
	}

	if fields[0].GoName != "Name" {
		t.Fatal("expected Name as search field")
	}
}

func TestRegister_NoDBTags(t *testing.T) {
	desc := Register[noDBTag]()

	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}

	if len(desc.Fields) != 0 {
		t.Fatalf("expected 0 fields for struct without db tags, got %d", len(desc.Fields))
	}
}

func TestRegister_ReadOnly(t *testing.T) {
	type withReadOnly struct {
		Name string `db:"name" label:"Name" order:"10" list:"true"`
		Org  string `readonly:"true" label:"Organization" order:"20" list:"true"`
	}

	desc := Register[withReadOnly]()

	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}

	if len(desc.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(desc.Fields))
	}

	if !desc.Fields[1].ReadOnly {
		t.Fatal("expected ReadOnly=true for Org field")
	}

	if desc.Fields[1].DBName != "" {
		t.Fatalf("expected empty DBName for readonly field, got %q", desc.Fields[1].DBName)
	}

	listFields := desc.ListFields()
	if len(listFields) != 2 {
		t.Fatalf("expected 2 list fields, got %d", len(listFields))
	}
}

func TestRegister_DuplicateOrder(t *testing.T) {
	type dupOrder struct {
		A string `db:"a" label:"A" order:"10"`
		B string `db:"b" label:"B" order:"10"`
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate order")
		}
	}()

	Register[dupOrder]()
}
