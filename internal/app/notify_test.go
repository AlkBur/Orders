package app

import (
	"context"
	"encoding/json"
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

// TestNotify_ReceiptStatus проверяет текст уведомления об изменении статуса:
// отдельный status-шаблон с полем «Статус» и инициатором события.
func TestNotify_ReceiptStatus(t *testing.T) {
	t.Run("submit", func(t *testing.T) {
		db := testutil.NewTestDB(t, NewSchema())
		orgID, _ := insertOrg(t, db, "NotifyStatusOrg", "kns")
		app := &App{receipts: receipts.NewStore(db), log: NewLogger(false)}
		notifier := newRecordingNotifier()
		app.notifier = notifier

		authorID := insertTestUser(t, db, "status-author")
		rec := saveAppReceipt(t, app, orgID, "NST001", "", false)
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
		if !strings.Contains(msg.Title, "статус "+receipts.StatusSent) {
			t.Fatalf("title = %q, want status", msg.Title)
		}
		if !strings.Contains(msg.Message, "Статус: "+receipts.StatusSent) {
			t.Fatalf("message = %q, want status field", msg.Message)
		}
		if !strings.Contains(msg.Message, "Пользователь: operator") {
			t.Fatalf("message = %q, want actor", msg.Message)
		}
		if strings.Contains(msg.Message, "status-author") {
			t.Fatalf("author substituted for actor: %q", msg.Message)
		}
	})

	t.Run("mark_deleted", func(t *testing.T) {
		db := testutil.NewTestDB(t, NewSchema())
		orgID, _ := insertOrg(t, db, "NotifyCancelOrg", "knc")
		app := &App{receipts: receipts.NewStore(db), log: NewLogger(false)}
		notifier := newRecordingNotifier()
		app.notifier = notifier

		authorID := insertTestUser(t, db, "cancel-author")
		rec := saveAppReceipt(t, app, orgID, "NST002", receipts.StatusCreated, false)
		if _, err := db.Exec(`UPDATE receipts SET user_id = ? WHERE id = ?`, authorID, rec.ID); err != nil {
			t.Fatal(err)
		}
		idStr := strconv.FormatInt(rec.ID, 10)
		admin := users.Identity{ID: 7, Login: "admin", IsAdmin: true}

		w := httptest.NewRecorder()
		app.ReceiptMarkDeleted(w, markUserRequest(t, idStr, admin))
		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d: %s", w.Code, w.Body.String())
		}

		msg := waitMessage(t, notifier.msgs)
		if !strings.Contains(msg.Title, "статус "+receipts.StatusCancelled) {
			t.Fatalf("title = %q, want cancelled status", msg.Title)
		}
		if !strings.Contains(msg.Message, "Статус: "+receipts.StatusCancelled) {
			t.Fatalf("message = %q, want status field", msg.Message)
		}
		if !strings.Contains(msg.Message, "Пользователь: admin") {
			t.Fatalf("message = %q, want actor", msg.Message)
		}

		// Повторная пометка — no-op (ErrNotFound), уведомления нет.
		w = httptest.NewRecorder()
		app.ReceiptMarkDeleted(w, markUserRequest(t, idStr, admin))
		if w.Code != http.StatusNotFound {
			t.Fatalf("repeat: expected 404, got %d", w.Code)
		}
		assertNoMessage(t, notifier.msgs)
	})

	t.Run("api_patch", func(t *testing.T) {
		app, orgID, orgUUID := setupSyncApp(t)
		app.log = NewLogger(false)
		notifier := newRecordingNotifier()
		app.notifier = notifier

		rec := insertQueuedReceipt(t, app, orgID)
		uuid := "ext-status-1"
		if _, err := app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: rec.ID, UUID: &uuid}}); err != nil {
			t.Fatal(err)
		}

		body, _ := json.Marshal(map[string]string{"status": receipts.StatusAccepted})
		path := "/api/integration/organizations/" + orgUUID + "/receipts/" + uuid

		w := httptest.NewRecorder()
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(
			w, syncRequest(t, http.MethodPut, path, orgUUID, uuid, "k1", body))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		msg := waitMessage(t, notifier.msgs)
		if !strings.Contains(msg.Message, "Статус: "+receipts.StatusAccepted) {
			t.Fatalf("message = %q, want status field", msg.Message)
		}
		if !strings.Contains(msg.Message, "Пользователь: "+notifyIntegrationActor) {
			t.Fatalf("message = %q, want integration actor", msg.Message)
		}

		// Повтор того же статуса — уведомления нет.
		w = httptest.NewRecorder()
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(
			w, syncRequest(t, http.MethodPut, path, orgUUID, uuid, "k1", body))
		if w.Code != http.StatusOK {
			t.Fatalf("repeat: expected 200, got %d", w.Code)
		}
		assertNoMessage(t, notifier.msgs)
	})

	t.Run("api_sync", func(t *testing.T) {
		app, orgID, orgUUID := setupSyncApp(t)
		app.log = NewLogger(false)
		notifier := newRecordingNotifier()
		app.notifier = notifier

		rec := insertQueuedReceipt(t, app, orgID)
		payload, _ := json.Marshal([]map[string]any{
			{"id": rec.ID, "uuid": "ext-sync-1", "status": receipts.StatusAccepted},
		})
		path := "/api/integration/organizations/" + orgUUID + "/receipts"

		w := httptest.NewRecorder()
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(
			w, syncRequest(t, http.MethodPut, path, orgUUID, "", "k1", payload))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		msg := waitMessage(t, notifier.msgs)
		if !strings.Contains(msg.Message, "Статус: "+receipts.StatusAccepted) {
			t.Fatalf("message = %q, want status field", msg.Message)
		}
		if !strings.Contains(msg.Message, "Пользователь: "+notifyIntegrationActor) {
			t.Fatalf("message = %q, want integration actor", msg.Message)
		}
	})

	t.Run("api_sync_duplicate_id", func(t *testing.T) {
		app, orgID, orgUUID := setupSyncApp(t)
		app.log = NewLogger(false)
		notifier := newRecordingNotifier()
		app.notifier = notifier

		rec := insertQueuedReceipt(t, app, orgID)
		payload, _ := json.Marshal([]map[string]any{
			{"id": rec.ID, "uuid": "ext-sync-dup", "status": receipts.StatusAccepted},
			{"id": rec.ID, "uuid": "ext-sync-dup", "status": receipts.StatusProcessed},
		})
		path := "/api/integration/organizations/" + orgUUID + "/receipts"

		w := httptest.NewRecorder()
		app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(
			w, syncRequest(t, http.MethodPut, path, orgUUID, "", "k1", payload))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Один receipt в запросе дважды — ровно одно уведомление, последний статус.
		msg := waitMessage(t, notifier.msgs)
		if !strings.Contains(msg.Message, "Статус: "+receipts.StatusProcessed) {
			t.Fatalf("message = %q, want last status", msg.Message)
		}
		assertNoMessage(t, notifier.msgs)
	})
}

// TestNotify_ReceiptAction проверяет текст уведомления об изменении действия:
// отдельный action-шаблон с полем «Действие» и инициатором события.
func TestNotify_ReceiptAction(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	orgID, _ := insertOrg(t, db, "NotifyActionOrg", "kna")
	app := &App{receipts: receipts.NewStore(db), log: NewLogger(false)}
	notifier := newRecordingNotifier()
	app.notifier = notifier

	authorID := insertTestUser(t, db, "action-author")
	rec := saveSyncedAppReceipt(t, app, orgID, "NACT001")
	if _, err := db.Exec(`UPDATE receipts SET user_id = ? WHERE id = ?`, authorID, rec.ID); err != nil {
		t.Fatal(err)
	}
	idStr := strconv.FormatInt(rec.ID, 10)
	operator := users.Identity{ID: 7, Login: "operator"}

	// Установка действия.
	w := httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("set: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	msg := waitMessage(t, notifier.msgs)
	if !strings.Contains(msg.Title, "действие "+receipts.ActionDelete) {
		t.Fatalf("set: title = %q, want action", msg.Title)
	}
	if !strings.Contains(msg.Message, "Действие: "+receipts.ActionDelete) {
		t.Fatalf("set: message = %q, want action field", msg.Message)
	}
	if !strings.Contains(msg.Message, "Пользователь: operator") {
		t.Fatalf("set: message = %q, want actor", msg.Message)
	}
	if strings.Contains(msg.Message, "action-author") {
		t.Fatalf("set: author substituted for actor: %q", msg.Message)
	}

	// Повтор того же действия — no-op.
	w = httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, receipts.ActionDelete), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("repeat: expected 303, got %d", w.Code)
	}
	assertNoMessage(t, notifier.msgs)

	// Снятие действия.
	w = httptest.NewRecorder()
	app.ReceiptActionSave(w, withUserIdentity(actionRequest(t, http.MethodPost, idStr, ""), operator))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("clear: expected 303, got %d", w.Code)
	}
	msg = waitMessage(t, notifier.msgs)
	if !strings.Contains(msg.Title, "действие отменено") {
		t.Fatalf("clear: title = %q, want cancel", msg.Title)
	}
	if !strings.Contains(msg.Message, "Действие: отменено") {
		t.Fatalf("clear: message = %q, want cancel field", msg.Message)
	}
	if !strings.Contains(msg.Message, "Пользователь: operator") {
		t.Fatalf("clear: message = %q, want actor", msg.Message)
	}
}

// TestNotify_TemplateFields фиксирует, что это два разных текста: status
// содержит поле «Статус», action — поле «Действие».
func TestNotify_TemplateFields(t *testing.T) {
	data := notifyData{
		Number:       "1",
		Date:         "01.01.2026",
		Organization: "ООО Ромашка",
		Customer:     "Иванов",
		Total:        "100",
		User:         "operator",
		Status:       "Принят",
		Action:       "Удалить",
	}

	statusTitle, err := renderNotify(statusTemplates, "title", data)
	if err != nil {
		t.Fatal(err)
	}
	statusBody, err := renderNotify(statusTemplates, "body", data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusTitle, "Принят") {
		t.Fatalf("status title = %q, want status", statusTitle)
	}
	if !strings.Contains(statusBody, "Статус: Принят") {
		t.Fatalf("status body = %q, want status field", statusBody)
	}
	if !strings.Contains(statusBody, "Дата: 01.01.2026") {
		t.Fatalf("status body = %q, want date field", statusBody)
	}
	if strings.Contains(statusBody, "Действие") {
		t.Fatalf("status body must not contain action field: %q", statusBody)
	}

	actionTitle, err := renderNotify(actionTemplates, "title", data)
	if err != nil {
		t.Fatal(err)
	}
	actionBody, err := renderNotify(actionTemplates, "body", data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(actionTitle, "Удалить") {
		t.Fatalf("action title = %q, want action", actionTitle)
	}
	if !strings.Contains(actionBody, "Действие: Удалить") {
		t.Fatalf("action body = %q, want action field", actionBody)
	}
	if !strings.Contains(actionBody, "Дата: 01.01.2026") {
		t.Fatalf("action body = %q, want date field", actionBody)
	}
	if strings.Contains(actionBody, "Статус") {
		t.Fatalf("action body must not contain status field: %q", actionBody)
	}

	// Отмена действия идёт через тот же action-шаблон со значением «отменено».
	data.Action = "отменено"
	cancelBody, err := renderNotify(actionTemplates, "body", data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cancelBody, "Действие: отменено") {
		t.Fatalf("cancel body = %q, want cancel field", cancelBody)
	}
}
