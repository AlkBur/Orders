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
	desc := Register[testEntity](
		PrimaryKey("ID"),
	)

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
	d1 := Register[testEntity](PrimaryKey("ID"))
	d2 := Register[testEntity]()

	if d1 != d2 {
		t.Fatal("expected the same pointer for cached descriptor")
	}
}

func TestListFields(t *testing.T) {
	desc := Register[testEntity](PrimaryKey("ID"))
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
	desc := Register[testEntity](PrimaryKey("ID"))
	fields := desc.SearchFields()

	if len(fields) != 1 {
		t.Fatalf("expected 1 search field, got %d", len(fields))
	}

	if fields[0].GoName != "Name" {
		t.Fatal("expected Name as search field")
	}
}

func TestRegister_NoDBTags(t *testing.T) {
	type withIgnored struct {
		ID   string `db:"id" label:"ID" order:"10"`
		Name string // no tag: should be ignored
	}

	desc := Register[withIgnored](PrimaryKey("ID"))

	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}

	if len(desc.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(desc.Fields))
	}

	if desc.Fields[0].GoName != "ID" {
		t.Fatal("expected ID field")
	}
}

func TestRegister_ReadOnly(t *testing.T) {
	type withReadOnly struct {
		Name string `db:"name" label:"Name" order:"10" list:"true"`
		Org  string `readonly:"true" label:"Organization" order:"20" list:"true"`
	}

	desc := Register[withReadOnly](
		PrimaryKey("Name"),
	)

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

	Register[dupOrder](PrimaryKey("A"))
}

type keyTestEntity struct {
	ID   string `db:"id" label:"ID" order:"10" list:"true"`
	UUID string `db:"uuid" label:"UUID" order:"20" list:"true"`
}

func TestRegister_Keys(t *testing.T) {
	desc := Register[keyTestEntity](
		PrimaryKey("ID"),
		ExternalKey("UUID"),
	)

	if desc.PrimaryKey() == nil {
		t.Fatal("expected PrimaryKey")
	}
	if desc.PrimaryKey().Kind != KeyPrimary {
		t.Fatal("expected KeyPrimary kind")
	}
	if len(desc.PrimaryKey().Fields) != 1 {
		t.Fatal("expected 1 field in PrimaryKey")
	}
	if desc.PrimaryKey().Fields[0].GoName != "ID" {
		t.Fatal("expected PrimaryKey field ID")
	}

	if desc.ExternalKey() == nil {
		t.Fatal("expected ExternalKey")
	}
	if desc.ExternalKey().Kind != KeyExternal {
		t.Fatal("expected KeyExternal kind")
	}
	if len(desc.ExternalKey().Fields) != 1 {
		t.Fatal("expected 1 field in ExternalKey")
	}
	if desc.ExternalKey().Fields[0].GoName != "UUID" {
		t.Fatal("expected ExternalKey field UUID")
	}
}

func TestRegister_CompositeExternalKey(t *testing.T) {
	type compositeEntity struct {
		ID             string `db:"id" label:"ID" order:"10"`
		OrganizationID string `db:"organization_id" label:"Org" order:"20"`
		UUID           string `db:"uuid" label:"UUID" order:"30"`
	}

	desc := Register[compositeEntity](
		PrimaryKey("ID"),
		ExternalKey("OrganizationID", "UUID"),
	)

	if desc.ExternalKey() == nil {
		t.Fatal("expected ExternalKey")
	}
	if !desc.ExternalKey().IsComposite() {
		t.Fatal("expected composite ExternalKey")
	}
	if len(desc.ExternalKey().Fields) != 2 {
		t.Fatal("expected 2 fields in ExternalKey")
	}
	if desc.ExternalKey().Fields[0].GoName != "OrganizationID" {
		t.Fatal("expected first field OrganizationID")
	}
	if desc.ExternalKey().Fields[1].GoName != "UUID" {
		t.Fatal("expected second field UUID")
	}
}

func TestRegister_PanicNoPrimaryKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing PrimaryKey")
		}
	}()

	type noPK struct {
		UUID string `db:"uuid" label:"UUID" order:"10"`
	}

	Register[noPK]()
}

func TestRegister_MultipleExternalKeys(t *testing.T) {
	type multiKey struct {
		ID   string `db:"id" label:"ID" order:"10"`
		UUID string `db:"uuid" label:"UUID" order:"20"`
		Code string `db:"code" label:"Code" order:"30"`
	}

	desc := Register[multiKey](
		PrimaryKey("ID"),
		ExternalKey("UUID"),
		ExternalKey("Code"),
	)

	if desc == nil {
		t.Fatal("expected non-nil descriptor")
	}

	if len(desc.Keys) != 3 {
		t.Fatalf("expected 3 keys (1 PK + 2 external), got %d", len(desc.Keys))
	}

	if desc.PrimaryKey() == nil {
		t.Fatal("expected PrimaryKey")
	}
	if desc.PrimaryKey().Fields[0].GoName != "ID" {
		t.Fatal("expected PrimaryKey field ID")
	}

	if desc.ExternalKey() == nil {
		t.Fatal("expected ExternalKey")
	}
}

func TestRegister_PanicUnknownField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown field")
		}
	}()

	type simple struct {
		ID string `db:"id" label:"ID" order:"10"`
	}

	Register[simple](
		PrimaryKey("ID"),
		ExternalKey("Unknown"),
	)
}
