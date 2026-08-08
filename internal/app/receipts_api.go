package app

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"Orders/internal/receipts"

	"github.com/go-chi/chi/v5"
)

// maxFileUploadBytes — максимальный размер всего multipart-запроса (30 MiB).
// Это предел всей загрузки, а не отдельного файла PDF.
const maxFileUploadBytes = 30 << 20

// pdfMagic — сигнатура PDF-документа (%PDF-).
var pdfMagic = []byte("%PDF-")

// HandlePutReceiptFile прикрепляет файл (PDF) к документу через
// Integration API. Идемпотентен по (receipt_id, uuid): повторная загрузка
// того же uuid заменяет содержимое и метаданные файла полностью.
func (a *App) HandlePutReceiptFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileUploadBytes)
	defer r.Body.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		a.BadRequest(w, "Content-Type must be multipart/form-data")
		return
	}

	orgUUID := chi.URLParam(r, "oid")
	receiptUUID := chi.URLParam(r, "ruuid")

	org, err := a.organizations.GetByUUID(r.Context(), orgUUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	doc, err := a.receipts.GetByExternal(r.Context(), receiptUUID)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}
	if doc.Receipt.OrganizationID != org.ID {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		a.BadRequest(w, "Invalid multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	fileUUID := strings.TrimSpace(r.FormValue("uuid"))
	if fileUUID == "" {
		a.BadRequest(w, "uuid is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		a.BadRequest(w, "exactly one file part named 'file' is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	if !isValidPDF(header, data) {
		a.BadRequest(w, "only PDF files are accepted")
		return
	}

	inserted, updated, err := a.receiptFiles.Upsert(r.Context(),
		doc.Receipt.ID, fileUUID, header.Filename, "application/pdf", data)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileSyncResult{Inserted: toBoolInt(inserted), Updated: toBoolInt(updated)})
}

// isValidPDF проверяет оба признака PDF: MIME-тип части должен быть
// application/pdf, а содержимое начинаться с сигнатуры %PDF-.
func isValidPDF(h *multipart.FileHeader, data []byte) bool {
	return h.Header.Get("Content-Type") == "application/pdf" &&
		len(data) >= len(pdfMagic) &&
		string(data[:len(pdfMagic)]) == string(pdfMagic)
}

type fileSyncResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
}

func toBoolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
