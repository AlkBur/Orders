package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

// setupSyncApp создаёт App с организацией и клиентом, пригодный
// для проверки очереди синхронизации чеков.
func setupSyncApp(t *testing.T) (*App, int64, string) {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	orgID, orgUUID := insertOrg(t, db, "SyncOrg", "k1")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		orgKeys:       map[string]string{orgUUID: "k1"},
	}

	c := app.customers.New()
	c.OrganizationID = orgID
	c.UUID = "sync-cust"
	c.Name = "ООО Клиент"
	if err := app.customers.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	return app, orgID, orgUUID
}

// insertQueuedReceipt создаёт опубликованный чек без внешнего UUID
// (т.е. ожидающий синхронизации) и возвращает его.
func insertQueuedReceipt(t *testing.T, app *App, orgID int64) *receipts.Receipt {
	t.Helper()
	now := time.Now()
	rec := &receipts.Receipt{
		Number:         "Q" + uniqueSuffix(),
		Date:           now,
		OrganizationID: orgID,
		UserID:         1,
		CustomerID:     1,
		Total:          123.45,
		SentAt:         &now,
		Status:         "",
	}
	if err := app.receipts.Save(context.Background(), &receipts.Document{Receipt: rec}); err != nil {
		t.Fatal(err)
	}
	return rec
}

// syncRequest строит запрос с уже установленными URL-параметрами.
// ruuid передаётся явно; для маршрутов без него — пустая строка.
func syncRequest(t *testing.T, method, path, orgUUID, ruuid, apiKey string, body []byte) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		r.Header.Set("X-API-Key", apiKey)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	if ruuid != "" {
		rctx.URLParams.Add("ruuid", ruuid)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSyncAPI_GetQueue(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	sent := insertQueuedReceipt(t, app, orgID)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptsQueue)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got []ReceiptQueueItem
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 queue item, got %d", len(got))
	}
	if got[0].ID != sent.ID {
		t.Fatalf("expected ID %d, got %d", sent.ID, got[0].ID)
	}
	if got[0].Number != sent.Number {
		t.Fatalf("expected Number %s, got %s", sent.Number, got[0].Number)
	}
	if got[0].CustomerUUID != "sync-cust" {
		t.Fatalf("expected CustomerUUID sync-cust, got %s", got[0].CustomerUUID)
	}
	if got[0].CustomerName != "ООО Клиент" {
		t.Fatalf("expected CustomerName ООО Клиент, got %s", got[0].CustomerName)
	}
	if got[0].Total != 123.45 {
		t.Fatalf("expected Total 123.45, got %f", got[0].Total)
	}
}

func TestSyncAPI_GetQueue_Empty(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptsQueue)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []ReceiptQueueItem
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected empty array, got %#v", got)
	}
}

func TestSyncAPI_SyncAssign(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	sent := insertQueuedReceipt(t, app, orgID)

	status := "Принят"
	uuid := "1c-assigned"
	body, _ := json.Marshal([]receiptSyncRequest{{
		ID: sent.ID, UUID: &uuid, Status: &status,
	}})

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res receipts.SyncResult
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated 1, got %+v", res)
	}

	doc, err := app.receipts.GetByID(context.Background(), sent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.UUID != uuid {
		t.Fatalf("expected UUID %s, got %s", uuid, doc.Receipt.UUID)
	}
	if doc.Receipt.Status != status {
		t.Fatalf("expected Status %s, got %s", status, doc.Receipt.Status)
	}

	remaining, err := app.receipts.ListAvailableForSync(context.Background(), orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty queue after assign, got %d", len(remaining))
	}
}

func TestSyncAPI_SyncConflict(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	sent := insertQueuedReceipt(t, app, orgID)

	uuid1 := "1c-a"
	uuid2 := "1c-b"
	app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: sent.ID, UUID: &uuid1}})

	body, _ := json.Marshal([]receiptSyncRequest{{
		ID: sent.ID, UUID: &uuid2,
	}})

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_SyncNotFound(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)
	status := "Принят"
	body, _ := json.Marshal([]receiptSyncRequest{{
		ID: 999999, Status: &status,
	}})

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_SyncBadJSON(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", []byte(`{not json`))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_SyncMissingAPIKey(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptsQueue)).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSyncAPI_UpdateStatus(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	sent := insertQueuedReceipt(t, app, orgID)
	uuid := "1c-status"
	app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: sent.ID, UUID: &uuid}})

	status := "Принят"
	body, _ := json.Marshal(receiptStatusRequest{Status: &status})

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/"+uuid, orgUUID, uuid, "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	doc, err := app.receipts.GetByID(context.Background(), sent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Receipt.Status != status {
		t.Fatalf("expected Status %s, got %s", status, doc.Receipt.Status)
	}
}

func TestSyncAPI_UpdateStatus_NotFound(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)
	status := "Принят"
	body, _ := json.Marshal(receiptStatusRequest{Status: &status})

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/no-such", orgUUID, "no-such", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_UpdateStatus_MissingFields(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/x", orgUUID, "x", "k1", []byte(`{}`))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// syncReceiptWithUUID создаёт опубликованный чек и назначает ему внешний
// UUID напрямую через хранилище, возвращая uuid.
func syncReceiptWithUUID(t *testing.T, app *App, orgID int64) string {
	t.Helper()
	rec := insertQueuedReceipt(t, app, orgID)
	uuid := "1c-" + uniqueSuffix()
	if _, err := app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: rec.ID, UUID: &uuid}}); err != nil {
		t.Fatal(err)
	}
	return uuid
}

func putReceiptStatus(t *testing.T, app *App, orgUUID, ruuid string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/"+ruuid, orgUUID, ruuid, "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(w, r)
	return w
}

// status_color убран из контракта: цвет вычисляется приложением из статуса.
// Отправка поля обязана завершаться 400 (DisallowUnknownFields).
func TestSyncAPI_RejectsStatusColor(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	uuid := syncReceiptWithUUID(t, app, orgID)

	color := "#00ff00"

	// PATCH /receipts/{ruuid} со status_color — 400.
	body, _ := json.Marshal(map[string]any{"status_color": color})
	if w := putReceiptStatus(t, app, orgUUID, uuid, body); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for status_color in status update, got %d: %s", w.Code, w.Body.String())
	}

	// PUT /receipts (sync) со status_color — 400.
	rec := insertQueuedReceipt(t, app, orgID)
	body, _ = json.Marshal([]map[string]any{{"id": rec.ID, "status_color": color}})
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleSyncReceipts)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for status_color in sync, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_OtherOrgNotFound(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	org1, org1UUID := insertOrg(t, db, "Org1", "k1")
	_, org2UUID := insertOrg(t, db, "Org2", "k2")

	app := &App{
		customers:     customers.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		orgKeys:       map[string]string{org1UUID: "k1", org2UUID: "k2"},
	}
	sent := insertQueuedReceipt(t, app, org1)

	// Авторизация ключом Org2, чек принадлежит Org1 — чужая очередь.
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+org2UUID+"/receipts", org2UUID, "", "k2", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptsQueue)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []ReceiptQueueItem
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 0 {
		t.Fatalf("expected empty queue for org2, got %d", len(got))
	}

	// Обновление чужого чека по uuid — 404.
	status := "Принят"
	body, _ := json.Marshal(receiptStatusRequest{Status: &status})
	uuid := "1c-other"
	app.receipts.SynchronizeByID(context.Background(), org1, []receipts.SyncUpdate{{ID: sent.ID, UUID: &uuid}})

	w2 := httptest.NewRecorder()
	r2 := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+org2UUID+"/receipts/"+uuid, org2UUID, uuid, "k2", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleUpdateReceiptStatus)).ServeHTTP(w2, r2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign org, got %d: %s", w2.Code, w2.Body.String())
	}
}

// actionReceipt создаёт синхронизированный чек (uuid назначен) с текущим
// действием и возвращает его.
func actionReceipt(t *testing.T, app *App, orgID int64, action string) *receipts.Receipt {
	t.Helper()
	rec := insertQueuedReceipt(t, app, orgID)
	uuid := "1c-act-" + uniqueSuffix()
	if _, err := app.receipts.SynchronizeByID(context.Background(), orgID, []receipts.SyncUpdate{{ID: rec.ID, UUID: &uuid}}); err != nil {
		t.Fatal(err)
	}
	rec.UUID = uuid
	if err := app.receipts.SetAction(context.Background(), rec.ID, action); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestSyncAPI_GetActions(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	rec := actionReceipt(t, app, orgID, receipts.ActionChange)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got []receipts.Action
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 action, got %d", len(got))
	}
	if got[0].UUID != rec.UUID || got[0].Action != receipts.ActionChange {
		t.Fatalf("unexpected action item: %+v", got[0])
	}
}

func TestSyncAPI_GetActions_Empty(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []receipts.Action
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected empty array, got %#v", got)
	}
}

func TestSyncAPI_ConfirmActions(t *testing.T) {
	app, orgID, orgUUID := setupSyncApp(t)
	rec := actionReceipt(t, app, orgID, receipts.ActionDelete)

	body, _ := json.Marshal([]receiptActionConfirm{{UUID: rec.UUID}})
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleConfirmReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res receipts.SyncResult
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated 1, got %+v", res)
	}

	// Действие больше не в очереди (дата получения установлена).
	w2 := httptest.NewRecorder()
	r2 := syncRequest(t, http.MethodGet, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", nil)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleGetReceiptActions)).ServeHTTP(w2, r2)
	var remaining []receipts.Action
	json.NewDecoder(w2.Body).Decode(&remaining)
	if len(remaining) != 0 {
		t.Fatalf("expected empty actions after confirm, got %d", len(remaining))
	}
}

func TestSyncAPI_ConfirmActions_NotFound(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	body, _ := json.Marshal([]receiptActionConfirm{{UUID: "missing"}})
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleConfirmReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_ConfirmActions_BadJSON(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", []byte(`{not json`))
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleConfirmReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncAPI_ConfirmActions_EmptyUUID(t *testing.T) {
	app, _, orgUUID := setupSyncApp(t)

	body, _ := json.Marshal([]receiptActionConfirm{{UUID: ""}})
	w := httptest.NewRecorder()
	r := syncRequest(t, http.MethodPut, "/api/integration/organizations/"+orgUUID+"/receipts/actions", orgUUID, "", "k1", body)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandleConfirmReceiptActions)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
