package app

import (
	"testing"
	"time"

	"Orders/internal/receipts"
)

func TestStatusPresentation_Keys(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		status  string
		created time.Time
		sentAt  *time.Time
		wantD   string
		wantKey StatusKey
	}{
		{name: "created", status: receipts.StatusCreated, created: now, wantD: "Создан", wantKey: StatusKeyCreated},
		{name: "overdue computed from created", status: receipts.StatusCreated, created: now.Add(-25 * time.Hour), wantD: "Просрочен", wantKey: StatusKeyOverdue},
		{name: "sent allowed", status: receipts.StatusSent, created: now, sentAt: &now, wantD: "Отправлен", wantKey: StatusKeySent},
		{name: "accepted", status: receipts.StatusAccepted, created: now, sentAt: &now, wantD: "Принят", wantKey: StatusKeyAccepted},
		{name: "cancelled", status: receipts.StatusCancelled, created: now, sentAt: &now, wantD: "Отменен", wantKey: StatusKeyCancelled},
		{name: "processed", status: receipts.StatusProcessed, created: now, sentAt: &now, wantD: "Обработан", wantKey: StatusKeyProcessed},
		{name: "finished", status: receipts.StatusFinished, created: now, sentAt: &now, wantD: "Завершен", wantKey: StatusKeyFinished},
		{name: "legacy empty with sentAt means sent", status: "", created: now, sentAt: &now, wantD: "Отправлен", wantKey: StatusKeySent},
		{name: "legacy empty without sentAt means created", status: "", created: now, wantD: "Создан", wantKey: StatusKeyCreated},
		{name: "zero createdAt is not overdue", status: receipts.StatusCreated, created: time.Time{}, wantD: "Создан", wantKey: StatusKeyCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusPresentation(tt.status, tt.created, tt.sentAt)
			if got.Display != tt.wantD {
				t.Fatalf("Display = %q, want %q", got.Display, tt.wantD)
			}
			if got.StatusKey != tt.wantKey {
				t.Fatalf("StatusKey = %q, want %q", got.StatusKey, tt.wantKey)
			}
		})
	}
}

func TestStatusPresentation_UnknownStatus(t *testing.T) {
	got := statusPresentation("Неизвестный", time.Now(), nil)
	if got.Display != "Неизвестный" {
		t.Fatalf("Display = %q, want original value", got.Display)
	}
	if got.StatusKey != "" {
		t.Fatalf("StatusKey = %q, want empty for unknown status", got.StatusKey)
	}
}