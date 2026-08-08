package receipts

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"Orders/internal/database"
	"Orders/internal/testutil"
)

func fileSchema() *database.Schema {
	s := database.NewSchema()
	if err := s.Register(FilesTable); err != nil {
		panic(err)
	}
	return s
}

func TestFileStore_UpsertCycle(t *testing.T) {
	filesDB := testutil.NewTestDB(t, fileSchema())
	store := NewFileStore(filesDB)

	// Вставка нового файла.
	inserted, updated, err := store.Upsert(context.Background(), 1, "f-1", "a.pdf", "application/pdf", []byte("%PDF-a"))
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || updated {
		t.Fatalf("expected inserted=true updated=false, got %v %v", inserted, updated)
	}

	files, err := store.ListByReceipt(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].UUID != "f-1" || files[0].FileName != "a.pdf" {
		t.Fatalf("unexpected metadata: %+v", files[0])
	}

	// Повторная загрузка того же uuid — полная замена.
	replacement := []byte("%PDF-replaced")
	inserted, updated, err = store.Upsert(context.Background(), 1, "f-1", "b.pdf", "application/pdf", replacement)
	if err != nil {
		t.Fatal(err)
	}
	if inserted || !updated {
		t.Fatalf("expected inserted=false updated=true, got %v %v", inserted, updated)
	}

	files, _ = store.ListByReceipt(context.Background(), 1)
	if len(files) != 1 {
		t.Fatalf("expected still 1 file, got %d", len(files))
	}
	if files[0].FileName != "b.pdf" {
		t.Fatalf("expected replaced name, got %q", files[0].FileName)
	}

	got, err := store.GetByID(context.Background(), 1, files[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Data, replacement) {
		t.Fatal("expected replaced content")
	}
}

func TestFileStoreGetByID_RequiresReceiptID(t *testing.T) {
	filesDB := testutil.NewTestDB(t, fileSchema())
	store := NewFileStore(filesDB)

	store.Upsert(context.Background(), 1, "f-1", "a.pdf", "application/pdf", []byte("%PDF-x"))
	files, _ := store.ListByReceipt(context.Background(), 1)

	if _, err := store.GetByID(context.Background(), 2, files[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong receipt, got %v", err)
	}
}
