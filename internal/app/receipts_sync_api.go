package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"Orders/internal/receipts"

	"github.com/go-chi/chi/v5"
)

// ReceiptQueueItem — документ из очереди синхронизации для 1С.
// id — внутренний ID Orders; используется только при первичном обмене.
type ReceiptQueueItem struct {
	ID           int64              `json:"id"`
	Number       string             `json:"number"`
	Date         string             `json:"date"`
	CustomerUUID string             `json:"customer_uuid"`
	CustomerName string             `json:"customer_name"`
	Total        float64            `json:"total"`
	Items        []ReceiptQueueLine `json:"items"`
}

// ReceiptQueueLine — строка документа из очереди синхронизации.
type ReceiptQueueLine struct {
	ProductUUID string  `json:"product_uuid"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
}

// receiptSyncRequest — элемент первичного подтверждения: id (внутренний)
// → uuid (необязательный) + status (частичное обновление).
type receiptSyncRequest struct {
	ID     int64   `json:"id"`
	UUID   *string `json:"uuid"`
	Status *string `json:"status"`
}

// receiptStatusRequest — изменение статуса чека по внешнему UUID.
// Частичная семантика: переданное поле изменяется, остальные не трогаются.
type receiptStatusRequest struct {
	Status *string `json:"status"`
}

// receiptActionConfirm — элемент подтверждения получения действия:
// только внешний uuid документа (подтверждается факт получения информации,
// а не конкретное действие).
type receiptActionConfirm struct {
	UUID string `json:"uuid"`
}

// getOrgFromURL извлекает организацию из {oid} в URL. Используется всеми
// хендлерами Integration API после RequireOrganizationAPIKey.
func (a *App) getOrgFromURL(w http.ResponseWriter, r *http.Request) (int64, bool) {
	org, err := a.organizations.GetByUUID(r.Context(), chi.URLParam(r, "oid"))
	if err != nil {
		http.NotFound(w, r)
		return 0, false
	}
	return org.ID, true
}

// HandleGetReceiptsQueue возвращает очередь синхронизации: опубликованные
// документы организации, которые ещё не получили внешний UUID от 1С.
func (a *App) HandleGetReceiptsQueue(w http.ResponseWriter, r *http.Request) {
	orgID, ok := a.getOrgFromURL(w, r)
	if !ok {
		return
	}

	docs, err := a.receipts.ListAvailableForSync(r.Context(), orgID)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	items := make([]ReceiptQueueItem, 0, len(docs))
	for _, doc := range docs {
		lines := make([]ReceiptQueueLine, 0, len(doc.Items))
		for _, item := range doc.Items {
			lines = append(lines, ReceiptQueueLine{
				ProductUUID: item.ProductUUID,
				ProductName: item.ProductName,
				Unit:        item.Unit,
				Quantity:    item.Quantity,
				Price:       item.Price,
				Amount:      item.Amount,
			})
		}
		items = append(items, ReceiptQueueItem{
			ID:           doc.Receipt.ID,
			Number:       doc.Receipt.Number,
			Date:         doc.Receipt.Date.Format("2006-01-02"),
			CustomerUUID: doc.Receipt.CustomerUUID,
			CustomerName: doc.Receipt.CustomerName,
			Total:        doc.Receipt.Total,
			Items:        lines,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// HandleSyncReceipts применяет первичное подтверждение: по внутреннему id
// документа 1С сообщает назначенный uuid (или его отсутствие) и статус.
// Частичное обновление; строка без полей пропускается.
func (a *App) HandleSyncReceipts(w http.ResponseWriter, r *http.Request) {
	orgID, ok := a.getOrgFromURL(w, r)
	if !ok {
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var requests []receiptSyncRequest
	if err := dec.Decode(&requests); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	updates := make([]receipts.SyncUpdate, 0, len(requests))
	for _, item := range requests {
		if item.ID <= 0 {
			a.BadRequest(w, "id is required")
			return
		}
		if item.Status != nil {
			if *item.Status == "" || !receipts.ValidStatus(*item.Status) {
				a.BadRequest(w, "invalid status")
				return
			}
		}
		updates = append(updates, receipts.SyncUpdate{
			ID:     item.ID,
			UUID:   item.UUID,
			Status: item.Status,
		})
	}

	result, err := a.receipts.SynchronizeByID(r.Context(), orgID, updates)
	if err != nil {
		switch {
		case errors.Is(err, receipts.ErrNotFound):
			http.NotFound(w, r)
		case errors.Is(err, receipts.ErrUUIDAlreadyAssigned):
			http.Error(w, receipts.ErrUUIDAlreadyAssigned.Error(), http.StatusConflict)
		case errors.Is(err, receipts.ErrEmptyUUID):
			a.BadRequest(w, "uuid must not be empty")
		default:
			a.InternalError(w, r, err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleUpdateReceiptStatus изменяет статус чека по внешнему UUID
// (receipts.uuid). PATCH-семантика: непереданный статус не сбрасывается.
func (a *App) HandleUpdateReceiptStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := a.getOrgFromURL(w, r)
	if !ok {
		return
	}

	var req receiptStatusRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	if req.Status == nil {
		a.BadRequest(w, "status is required")
		return
	}
	if *req.Status == "" || !receipts.ValidStatus(*req.Status) {
		a.BadRequest(w, "invalid status")
		return
	}

	ruuid := chi.URLParam(r, "ruuid")
	if err := a.receipts.UpdateByExternal(r.Context(), orgID, ruuid, req.Status); err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipts.SyncResult{Updated: 1})
}

// HandleGetReceiptActions возвращает ожидающие действия организации:
// документы, у которых есть запись в receipt_actions с незаполненной
// датой получения (action_received_at IS NULL).
func (a *App) HandleGetReceiptActions(w http.ResponseWriter, r *http.Request) {
	orgID, ok := a.getOrgFromURL(w, r)
	if !ok {
		return
	}

	actions, err := a.receipts.ListActionsForSync(r.Context(), orgID)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actions)
}

// HandleConfirmReceiptActions подтверждает получение действий 1С по
// внешнему uuid: устанавливает action_received_at=now(). Атомарно для
// всего массива; идемпотентно для уже подтверждённых.
func (a *App) HandleConfirmReceiptActions(w http.ResponseWriter, r *http.Request) {
	orgID, ok := a.getOrgFromURL(w, r)
	if !ok {
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var requests []receiptActionConfirm
	if err := dec.Decode(&requests); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	uuids := make([]string, 0, len(requests))
	for _, item := range requests {
		if item.UUID == "" {
			a.BadRequest(w, "uuid is required")
			return
		}
		uuids = append(uuids, item.UUID)
	}

	result, err := a.receipts.ConfirmActions(r.Context(), orgID, uuids)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
