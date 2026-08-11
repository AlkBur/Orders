package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"Orders/internal/receipts"
)

const (
	receiptListDefaultLimit = 50
	receiptListMinLimit     = 1
	receiptListMaxLimit     = 100
)

// receiptListCursor — внутреннее представление непрозрачного курсора
// keyset-пагинации списка чеков. Клиент не зависит от структуры
// сортировки: на стороне сервера id соответствует ORDER BY id DESC.
type receiptListCursor struct {
	ID int64 `json:"id"`
}

// encodeReceiptCursor превращает курсор хранилища в непрозрачную строку
// base64url(JSON). Ошибка сериализации невозможна для структурного значения.
func encodeReceiptCursor(c receipts.Cursor) string {
	raw, _ := json.Marshal(receiptListCursor{ID: c.ID})
	return base64.RawURLEncoding.EncodeToString(raw)
}

// parseReceiptAfter декодирует непрозрачный курсор after. Отсутствующий
// параметр даёт nil. Присутствующий, но некорректный (повреждённый или
// поддельный) курсор — ошибка: клиент получает 400, а не молчаливую
// перезагрузку первой порции.
func parseReceiptAfter(s string) (*receipts.Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	var c receiptListCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse cursor: %w", err)
	}
	if c.ID <= 0 {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &receipts.Cursor{ID: c.ID}, nil
}

// receiptListLimit разбирает параметр limit. Отсутствующий параметр даёт
// значение по умолчанию. Явно переданное значение вне 1..100 — ошибка
// (400), а не молчаливое ограничение: ошибка клиента не скрывается.
func receiptListLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return receiptListDefaultLimit, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < receiptListMinLimit || v > receiptListMaxLimit {
		return 0, fmt.Errorf(
			"limit must be an integer in %d..%d, got %q",
			receiptListMinLimit, receiptListMaxLimit, raw,
		)
	}
	return v, nil
}

// receiptListLoadMoreURL строит URL следующей порции списка. Все условия
// (текстовый поиск q и расширенные фильтры) повторяются в каждом запросе
// порции; limit и непрозрачный курсор сохраняются. Параметр part=rows
// позволяет обработчику отличить запрос порции от обычного запроса списка.
func (a *App) receiptListLoadMoreURL(query string, filter receiptFilter, limit int, after receipts.Cursor) string {
	q := url.Values{}
	if query != "" {
		q.Set("q", query)
	}
	if filter.dateFrom != "" {
		q.Set("date_from", filter.dateFrom)
	}
	if filter.dateTo != "" {
		q.Set("date_to", filter.dateTo)
	}
	if filter.amountFrom != nil {
		q.Set("amount_from", strconv.FormatFloat(*filter.amountFrom, 'f', -1, 64))
	}
	if filter.amountTo != nil {
		q.Set("amount_to", strconv.FormatFloat(*filter.amountTo, 'f', -1, 64))
	}
	if filter.orgID > 0 {
		q.Set("organization_id", strconv.FormatInt(filter.orgID, 10))
	}
	if filter.custID > 0 {
		q.Set("customer_id", strconv.FormatInt(filter.custID, 10))
	}
	if filter.status != "" {
		q.Set("status", filter.status)
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("part", "rows")
	q.Set("after", encodeReceiptCursor(after))
	return a.URL(RouteReceipts) + "?" + q.Encode()
}
