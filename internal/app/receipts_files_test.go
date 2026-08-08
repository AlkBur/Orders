package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/testutil"

	"github.com/go-chi/chi/v5"
)

// pdfBody — минимальное тело PDF: сигнатура %PDF- обязательна,
// остальное содержимое не проверяется.
var pdfBody = []byte("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF")

func TestReceiptFilesAPI_UploadInsert(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)

	w := httptest.NewRecorder()
	r := fileUploadRequest(t, orgUUID, receiptUUID, "file-uuid-1", "download.pdf", pdfBody)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result fileSyncResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Inserted != 1 || result.Updated != 0 {
		t.Fatalf("expected inserted=1 updated=0, got %+v", result)
	}

	doc, err := app.receipts.GetByExternal(context.Background(), receiptUUID)
	if err != nil {
		t.Fatal(err)
	}
	files, err := app.receiptFiles.ListByReceipt(context.Background(), doc.Receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].UUID != "file-uuid-1" || files[0].FileName != "download.pdf" {
		t.Fatalf("unexpected file metadata: %+v", files[0])
	}
	if files[0].MimeType != "application/pdf" {
		t.Fatalf("expected mime application/pdf, got %q", files[0].MimeType)
	}
	if files[0].FileSize != int64(len(pdfBody)) {
		t.Fatalf("expected size %d, got %d", len(pdfBody), files[0].FileSize)
	}
}

func TestReceiptFilesAPI_UploadUpdate(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)

	uploadFile(t, app, orgUUID, receiptUUID, "file-uuid-1", "original.pdf", pdfBody)

	replacement := []byte("%PDF-1.4\nreplacement body")
	uploadFile(t, app, orgUUID, receiptUUID, "file-uuid-1", "replaced.pdf", replacement)

	doc, _ := app.receipts.GetByExternal(context.Background(), receiptUUID)
	files, err := app.receiptFiles.ListByReceipt(context.Background(), doc.Receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file after upsert, got %d", len(files))
	}
	if files[0].FileName != "replaced.pdf" {
		t.Fatalf("expected replaced filename, got %q", files[0].FileName)
	}
	if files[0].FileSize != int64(len(replacement)) {
		t.Fatalf("expected replaced size %d, got %d", len(replacement), files[0].FileSize)
	}

	got, err := app.receiptFiles.GetByID(context.Background(), doc.Receipt.ID, files[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Data, replacement) {
		t.Fatal("expected replaced file content")
	}
}

func TestReceiptFilesAPI_NonPDFMIMERejected(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)

	w := httptest.NewRecorder()
	r := fileUploadRequest(t, orgUUID, receiptUUID, "file-uuid-2", "bad.txt", []byte("not pdf"))
	r.MultipartForm = nil // force reparse path; header already wrong
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiptFilesAPI_Errors(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)

	tests := []struct {
		name       string
		org        string
		ruuid      string
		uuid       string
		fileName   string
		body       []byte
		contentTyp string
		wantStatus int
	}{
		{
			name:       "WrongContentType",
			org:        orgUUID,
			ruuid:      receiptUUID,
			uuid:       "f1",
			fileName:   "a.pdf",
			body:       pdfBody,
			contentTyp: "application/json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "MissingReceipt",
			org:        orgUUID,
			ruuid:      "no-such-receipt",
			uuid:       "f1",
			fileName:   "a.pdf",
			body:       pdfBody,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "MissingUUID",
			org:        orgUUID,
			ruuid:      receiptUUID,
			uuid:       "",
			fileName:   "a.pdf",
			body:       pdfBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "MissingFile",
			org:        orgUUID,
			ruuid:      receiptUUID,
			uuid:       "f1",
			fileName:   "",
			body:       pdfBody,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			if tt.uuid != "" {
				mw.WriteField("uuid", tt.uuid)
			}
			if tt.fileName != "" {
				fw, _ := mw.CreateFormFile("file", tt.fileName)
				fw.Write(tt.body)
			}
			mw.Close()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut,
				"/api/integration/organizations/"+tt.org+"/receipts/"+tt.ruuid+"/files",
				&body)
			r.Header.Set("Content-Type", tt.contentTyp)
			if tt.contentTyp == "" {
				r.Header.Set("Content-Type", mw.FormDataContentType())
			}
			r.Header.Set("X-API-Key", "k1")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("oid", tt.org)
			rctx.URLParams.Add("ruuid", tt.ruuid)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestReceiptFilesAPI_ReceiptOfOtherOrgRejected(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	filesDB := testutil.NewTestDB(t, NewFilesSchema())
	org1, org1UUID := insertOrg(t, db, "Org1", "k1")
	_, org2UUID := insertOrg(t, db, "Org2", "k2")

	// Чек принадлежит Org1.
	receiptUUID := insertReceiptForOrg(t, db, org1)

	// Запрос авторизован ключом Org2, чек при этом принадлежит Org1 —
	// чужой документ → 404.
	app := &App{
		receipts:      receipts.NewStore(db),
		receiptFiles:  receipts.NewFileStore(filesDB),
		organizations: organizations.NewStore(db),
		orgKeys:       map[string]string{org1UUID: "k1", org2UUID: "k2"},
	}

	w := httptest.NewRecorder()
	r := fileUploadRequestWithKey(t, org2UUID, receiptUUID, "file-uuid-3", "a.pdf", pdfBody, "k2")
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiptFilesAPI_MissingAPIKey(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("uuid", "f1")
	fw, _ := mw.CreateFormFile("file", "a.pdf")
	fw.Write(pdfBody)
	mw.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut,
		"/api/integration/organizations/"+orgUUID+"/receipts/"+receiptUUID+"/files", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	rctx.URLParams.Add("ruuid", receiptUUID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiptFileContent_ServesPDF(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)
	uploadFile(t, app, orgUUID, receiptUUID, "file-uuid-1", "док.pdf", pdfBody)

	doc, _ := app.receipts.GetByExternal(context.Background(), receiptUUID)
	files, _ := app.receiptFiles.ListByReceipt(context.Background(), doc.Receipt.ID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+strconv.FormatInt(doc.Receipt.ID, 10)+"/files/"+strconv.FormatInt(files[0].ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(doc.Receipt.ID, 10))
	rctx.URLParams.Add("fileID", strconv.FormatInt(files[0].ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ReceiptFileContent(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("expected application/pdf, got %q", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), pdfBody) {
		t.Fatal("expected pdf body")
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "inline") {
		t.Fatalf("expected inline disposition, got %q", cd)
	}
}

func TestReceiptFileContent_UnknownFile404(t *testing.T) {
	app, orgUUID, receiptUUID := setupFilesApp(t)
	uploadFile(t, app, orgUUID, receiptUUID, "file-uuid-1", "a.pdf", pdfBody)

	doc, _ := app.receipts.GetByExternal(context.Background(), receiptUUID)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+strconv.FormatInt(doc.Receipt.ID, 10)+"/files/9999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(doc.Receipt.ID, 10))
	rctx.URLParams.Add("fileID", "9999")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ReceiptFileContent(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReceiptFileContent_CrossReceipt404(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	filesDB := testutil.NewTestDB(t, NewFilesSchema())
	org, orgUUID := insertOrg(t, db, "Org", "k1")
	ruuid1 := insertReceiptForOrg(t, db, org)
	ruuid2 := insertReceiptForOrg(t, db, org)

	app := &App{
		receipts:      receipts.NewStore(db),
		receiptFiles:  receipts.NewFileStore(filesDB),
		organizations: organizations.NewStore(db),
		orgKeys:       map[string]string{orgUUID: "k1"},
	}
	uploadFile(t, app, orgUUID, ruuid1, "file-uuid-1", "a.pdf", pdfBody)

	doc1, _ := app.receipts.GetByExternal(context.Background(), ruuid1)
	doc2, _ := app.receipts.GetByExternal(context.Background(), ruuid2)
	files1, _ := app.receiptFiles.ListByReceipt(context.Background(), doc1.Receipt.ID)

	// Файл из doc1 запрашивается в контексте doc2 — 404.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/receipts/"+strconv.FormatInt(doc2.Receipt.ID, 10)+"/files/"+strconv.FormatInt(files1[0].ID, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(doc2.Receipt.ID, 10))
	rctx.URLParams.Add("fileID", strconv.FormatInt(files1[0].ID, 10))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	app.ReceiptFileContent(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// fileUploadRequest строит multipart PUT-запрос с полем uuid и частью file.
// Часть file имеет Content-Type application/pdf (как от внешней системы).
func fileUploadRequest(t *testing.T, orgUUID, receiptUUID, fileUUID, fileName string, data []byte) *http.Request {
	t.Helper()
	return fileUploadRequestWithKey(t, orgUUID, receiptUUID, fileUUID, fileName, data, "k1")
}

func fileUploadRequestWithKey(t *testing.T, orgUUID, receiptUUID, fileUUID, fileName string, data []byte, apiKey string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("uuid", fileUUID)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	h.Set("Content-Type", "application/pdf")
	fw, _ := mw.CreatePart(h)
	fw.Write(data)
	mw.Close()

	r := httptest.NewRequest(http.MethodPut,
		"/api/integration/organizations/"+orgUUID+"/receipts/"+receiptUUID+"/files", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("X-API-Key", apiKey)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("oid", orgUUID)
	rctx.URLParams.Add("ruuid", receiptUUID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func uploadFile(t *testing.T, app *App, orgUUID, receiptUUID, fileUUID, fileName string, data []byte) {
	t.Helper()
	w := httptest.NewRecorder()
	r := fileUploadRequest(t, orgUUID, receiptUUID, fileUUID, fileName, data)
	app.RequireOrganizationAPIKey(http.HandlerFunc(app.HandlePutReceiptFile)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("upload %s: expected 200, got %d: %s", fileUUID, w.Code, w.Body.String())
	}
}

// setupFilesApp создаёт App с base.db и files.db, организацию и чек.
func setupFilesApp(t *testing.T) (*App, string, string) {
	t.Helper()
	db := testutil.NewTestDB(t, NewSchema())
	filesDB := testutil.NewTestDB(t, NewFilesSchema())
	orgID, orgUUID := insertOrg(t, db, "FilesOrg", "k1")
	receiptUUID := insertReceiptForOrg(t, db, orgID)

	app := &App{
		customers:     customers.NewStore(db),
		organizations: organizations.NewStore(db),
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		receiptFiles:  receipts.NewFileStore(filesDB),
		orgKeys:       map[string]string{orgUUID: "k1"},
	}
	return app, orgUUID, receiptUUID
}

// insertReceiptForOrg создаёт минимальную запись чека и возвращает его
// внешний UUID (receipts.uuid).
func insertReceiptForOrg(t *testing.T, dbt *sql.DB, orgID int64) string {
	t.Helper()
	uuid := "receipt-" + strconv.FormatInt(orgID, 10) + "-" + uniqueSuffix()
	number := "N" + uniqueSuffix()
	_, err := dbt.Exec(`
		INSERT INTO receipts (uuid, exchange_id, number, date, organization_id,
			user_id, customer_id, total, created_at, updated_at)
		VALUES (?, ?, ?, '2026-08-01', ?, 1, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid, "exch-"+uuid, number, orgID)
	if err != nil {
		t.Fatalf("insert receipt: %v", err)
	}
	return uuid
}

var uniqueCounter int

func uniqueSuffix() string {
	uniqueCounter++
	return strconv.Itoa(uniqueCounter)
}
