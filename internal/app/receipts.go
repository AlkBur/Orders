package app

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/receipts"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

func (a *App) ReceiptsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	list, err := a.receipts.List(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}

	fields := receipts.Descriptor.ListFields()

	var pageColumns []pages.Column
	for _, f := range fields {
		pageColumns = append(pageColumns, pages.Column{
			Name:  f.GoName,
			Label: f.Label,
		})
	}

	var rows []pages.Row
	for _, rec := range list {
		var item display.Values = *rec

		var cells []string
		for _, f := range fields {
			value, err := item.DisplayValue(f.GoName)
			if err != nil {
				a.InternalError(w, err)
				return
			}
			cells = append(cells, value)
		}
		rows = append(rows, pages.Row{
			Cells: cells,
			ID:    strconv.FormatInt(rec.ID, 10),
			URL:   rec.URL(),
		})
	}

	page := pages.ReceiptListPage{
		Title:     "Товарные чеки",
		Columns:   pageColumns,
		Rows:      rows,
		EmptyText: "Нет товарных чеков",
	}

	a.Render(w, "receipts", page)
}

func (a *App) ReceiptCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	doc, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	title := "Товарный чек №" + doc.Receipt.Number
	page := pages.ReceiptCardPage{
		Title:   title,
		Receipt: doc.Receipt,
		Items:   doc.Items,
	}

	a.Render(w, "receipt_card", page)
}

func (a *App) ReceiptSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if id > 0 {
		existing, err := a.receipts.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, receipts.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			a.InternalError(w, err)
			return
		}
		if existing.Receipt.SentAt != nil {
			a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
			return
		}
	}

	dateStr := r.FormValue("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		a.BadRequest(w, "Invalid date")
		return
	}

	rec := &receipts.Receipt{
		ID:             id,
		Number:         r.FormValue("number"),
		Date:           date,
		OrganizationID: parseInt64(r.FormValue("organization_id")),
		UserID:         parseInt64(r.FormValue("user_id")),
		CustomerID:     parseInt64(r.FormValue("customer_id")),
		Total:          parseFloat(r.FormValue("total")),
		Status:         r.FormValue("status"),
		StatusColor:    r.FormValue("status_color"),
	}

	if id == 0 {
		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		rec.ExchangeID = uuid
	}

	if err := a.receipts.Save(r.Context(), &receipts.Document{Receipt: rec}); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/receipts/"+strconv.FormatInt(rec.ID, 10), http.StatusSeeOther)
}

func (a *App) ReceiptDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	existing, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}
	if existing.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	if err := a.receipts.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/receipts", http.StatusSeeOther)
}

func (a *App) ReceiptSubmit(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	doc, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	if doc.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	now := time.Now()
	doc.Receipt.SentAt = &now
	doc.Receipt.UpdatedAt = now

	if err := a.receipts.Save(r.Context(), doc); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/receipts/"+idStr, http.StatusSeeOther)
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
