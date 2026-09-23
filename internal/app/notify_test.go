package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"Orders/internal/gotify"
	"Orders/internal/receipts"
	"Orders/internal/testutil"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

// recordingNotifier — fake gotifySender, складывающий сообщения в канал.
type recordingNotifier struct {
	msgs chan gotify.Message
}

func newRecordingNotifier() *recordingNotifier {
	return &recordingNotifier{msgs: make(chan gotify.Message, 8)}
}

func (n *recordingNotifier) Send(_ context.Context, msg gotify.Message) error {
	n.msgs <- msg
	return nil
}

func waitMessage(t *testing.T, ch <-chan gotify.Message) gotify.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
		return gotify.Message{}
	}
}

func assertNoMessage(t *testing.T, ch <-chan gotify.Message) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("unexpected notification: %+v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

// submitReceiptRequest собирает POST /receipts/{id}/send с URL-параметром id.
func submitReceiptRequest(t *testing.T, id int64) *http.Request {
	t.Helper()
	idStr := strconv.FormatInt(id, 10)
	r := httptest.NewRequest(http.MethodPost, "/receipts/"+idStr+"/send", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestNotify_ReceiptSubmit(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "NotifySendOrg", "knotify-send")
	app := &App{receipts: receipts.NewStore(db), log: NewLogger(false)}
	notifier := newRecordingNotifier()
	app.notifier = notifier

	authorID := insertTestUser(t, db, "doc-author")
	rec := saveAppReceipt(t, app, orgID, "NTF-SEND", receipts.StatusCreated, false)
	if _, err := db.Exec(`UPDATE receipts SET user_id = ? WHERE id = ?`, authorID, rec.ID); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := withUserIdentity(submitReceiptRequest(t, rec.ID), users.Identity{ID: 7, Login: "operator"})
	app.ReceiptSubmit(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
	}

	msg := waitMessage(t, notifier.msgs)
	if !strings.Contains(msg.Title, "отправлен в бухгалтерию") {
		t.Fatalf("unexpected title: %q", msg.Title)
	}
	if !strings.Contains(msg.Title, rec.Number) {
		t.Fatalf("title %q does not contain number %q", msg.Title, rec.Number)
	}
	if !strings.Contains(msg.Message, "Пользователь: operator") {
		t.Fatalf("expected actor in message, got %q", msg.Message)
	}
	if strings.Contains(msg.Message, "Пользователь: doc-author") {
		t.Fatalf("document author substituted for actor: %q", msg.Message)
	}
}

func TestNotify_ReceiptActionSave(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "NotifyActOrg", "knotify-act")
	app := &App{receipts: receipts.NewStore(db), log: NewLogger(false)}
	notifier := newRecordingNotifier()
	app.notifier = notifier

	authorID := insertTestUser(t, db, "act-author")
	rec := saveSyncedAppReceipt(t, app, orgID, "NTF001")
	if _, err := db.Exec(`UPDATE receipts SET user_id = ? WHERE id = ?`, authorID, rec.ID); err != nil {
		t.Fatal(err)
	}
	idStr := strconv.FormatInt(rec.ID, 10)
	operator := users.Identity{ID: 7, Login: "operator"}

	// Установка действия — уведомление с инициатором, а не автором.
	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	msg := waitMessage(t, notifier.msgs)
	if !strings.Contains(msg.Title, "действие "+receipts.ActionDelete) {
		t.Fatalf("set: unexpected title %q", msg.Title)
	}
	if !strings.Contains(msg.Message, "Действие: "+receipts.ActionDelete) {
		t.Fatalf("set: unexpected message %q", msg.Message)
	}
	if !strings.Contains(msg.Message, "Пользователь: operator") {
		t.Fatalf("set: expected actor in message, got %q", msg.Message)
	}
	if strings.Contains(msg.Message, "Пользователь: act-author") {
		t.Fatalf("set: document author substituted for actor: %q", msg.Message)
	}

	// Повтор того же действия — no-op, уведомления нет.
	w = httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("repeat: expected 303, got %d", w.Code)
	}
	assertNoMessage(t, notifier.msgs)

	// Очистка действия — уведомление об отмене с инициатором.
	w = httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, ""), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("clear: expected 303, got %d", w.Code)
	}
	msg = waitMessage(t, notifier.msgs)
	if !strings.Contains(msg.Title, "действие отменено") {
		t.Fatalf("clear: unexpected title %q", msg.Title)
	}
	if !strings.Contains(msg.Message, "Действие: отменено") {
		t.Fatalf("clear: unexpected message %q", msg.Message)
	}
	if !strings.Contains(msg.Message, "Пользователь: operator") {
		t.Fatalf("clear: expected actor in message, got %q", msg.Message)
	}
}
