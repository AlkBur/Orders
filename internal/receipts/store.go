package receipts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"Orders/internal/common"
	"Orders/internal/database/search"
	"Orders/internal/entity"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Receipt {
	return &Receipt{}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Document, error) {
	r, err := scanReceipt(s.db.QueryRowContext(ctx, `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			r.total, r.sent_at, r.status,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		WHERE r.id = ?
		  AND r.deleted_at IS NULL
	`, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	items, err := s.listItems(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Document{Receipt: r, Items: items}, nil
}

// GetByExternal находит документ по внешнему UUID чека (receipts.uuid,
// который присваивает внешняя система). Используется в Integration API.
func (s *Store) GetByExternal(ctx context.Context, externalUUID string) (*Document, error) {
	r, err := scanReceipt(s.db.QueryRowContext(ctx, `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			r.total, r.sent_at, r.status,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		WHERE r.uuid = ?
		  AND r.deleted_at IS NULL
	`, externalUUID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	items, err := s.listItems(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	return &Document{Receipt: r, Items: items}, nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM receipts WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

// Filter — условия расширенного отбора списка чеков.
// Нулевые значения полей не фильтруют.
//
// DateFrom/DateTo — границы периода в формате YYYY-MM-DD (колонка date —
// TEXT фиксированной ширины, сравнение строк включительно).
// AmountFrom/AmountTo — границы суммы (колонка total, REAL).
// OrganizationID / CustomerID — точный отбор по организации и контрагенту.
// Status — отображаемый статус: один из Status* или OverdueStatus;
// пустое значение — без фильтра по статусу. «Просрочен» — производное
// состояние, в базе не существует: вычисляется как StatusCreated,
// у которого с момента создания прошло не менее 24 часов (тот же
// механизм, что и statusPresentation в приложении).
// ActionSetFrom/ActionSetTo — границы даты установки действия
// (receipt_actions.action_set_at). Значения — даты YYYY-MM-DD.
type Filter struct {
	DateFrom       string
	DateTo         string
	AmountFrom     *float64
	AmountTo       *float64
	OrganizationID int64
	CustomerID     int64
	Status         string
	ActionSetFrom  string
	ActionSetTo    string
}

// Cursor — позиция keyset-пагинации списка чеков. Так как id —
// автоинкрементный идентификатор и монотонно растёт с созданием
// документа, он достаточен как признак порядка: следующая порция
// начинается строго после (ID).
type Cursor struct {
	ID int64
}

// ListOptions управляет выборкой списка чеков.
type ListOptions struct {
	Query  string
	Limit  int
	After  *Cursor
	Filter Filter
}

// ListPage — результат постраничной выборки списка чеков.
//
// HasMore сообщает, есть ли документы после текущей порции (определяется
// по факту получения limit+1 строки, без отдельного COUNT).
// Next — позиция следующей порции; не nil только при HasMore.
type ListPage struct {
	Items   []*Receipt
	HasMore bool
	Next    *Cursor
}

// receiptSearchColumns — поисковые колонки списка чеков.
// COALESCE нужен: организации и клиенты подключаются через LEFT JOIN.
var receiptSearchColumns = []search.MappedColumn{
	{Field: entity.FieldNameNumber, Expression: "r.number"},
	{Field: entity.FieldNameOrganizationName, Expression: "COALESCE(o.name, '')"},
	{Field: entity.FieldNameCustomerName, Expression: "COALESCE(c.name, '')"},
}

func (s *Store) searchableColumns() []search.MappedColumn {
	return receiptSearchColumns
}

// List возвращает все чеки без ограничения порции. visibleFields — поля,
// отображаемые в списке: поиск выполняется только по ним.
func (s *Store) List(ctx context.Context, opts ListOptions, visibleFields []entity.FieldName) ([]*Receipt, error) {
	page, err := s.listPage(ctx, opts, visibleFields, 0)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// ListPage возвращает одну порцию списка чеков (keyset-пагинация).
//
// Порядок — id DESC (id растёт вместе с созданием документа). Если задан
// opts.After, возвращаются только документы строго после позиции курсора.
// Поиск и фильтры применяются до LIMIT; дополнительных данных (fileCounts
// и т.п.) метод не загружает — это ответственность приложения для текущей
// порции.
//
// limit должен быть >= 1; HasMore определяется по факту получения limit+1
// строки (без отдельного COUNT).
func (s *Store) ListPage(ctx context.Context, opts ListOptions, visibleFields []entity.FieldName) (*ListPage, error) {
	return s.listPage(ctx, opts, visibleFields, opts.Limit)
}

func (s *Store) listPage(ctx context.Context, opts ListOptions, visibleFields []entity.FieldName, limit int) (*ListPage, error) {
	query := `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			r.total, r.sent_at, r.status,
			COALESCE(ra.action, '') AS action, ra.action_received_at,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		LEFT JOIN receipt_actions ra ON ra.receipt_id = r.id
	`

	searchWhere, searchArgs := search.BuildWhere(
		search.VisibleColumns(s.searchableColumns(), visibleFields),
		search.NormalizeQuery(opts.Query),
	)
	filterWhere, filterArgs := buildFilterWhere(opts.Filter)

	// Помеченные на удаление документы исключаются из списка безусловно.
	// Порядок кондиций определяет порядок плейсхолдеров: поиск, фильтр,
	// затем курсор.
	conds := []string{`r.deleted_at IS NULL`}
	if searchWhere != "" {
		conds = append(conds, searchWhere)
	}
	if filterWhere != "" {
		conds = append(conds, filterWhere)
	}
	if opts.After != nil {
		conds = append(conds, `r.id < ?`)
		filterArgs = append(filterArgs, opts.After.ID)
	}
	query += ` WHERE ` + strings.Join(conds, ` AND `)
	args := append(searchArgs, filterArgs...)

	query += ` ORDER BY r.id DESC`

	fetch := limit
	if fetch > 0 {
		// Фетчим на одну строку больше, чтобы узнать о наличии следующей
		// порции без отдельного запроса COUNT.
		fetch++
		query += ` LIMIT ?`
		args = append(args, fetch)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Receipt
	for rows.Next() {
		r, err := scanReceiptWithAction(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []*Receipt{}
	}

	page := &ListPage{Items: list}
	if limit > 0 && len(list) > limit {
		page.Items = list[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.Next = &Cursor{ID: last.ID}
	}
	return page, nil
}

// buildFilterWhere собирает условие WHERE расширенного отбора.
// Чистая функция: возвращает пустую строку без аргументов, если
// ни один параметр не задан. Соединение с поиском по тексту — AND
// (выполняется в List).
func buildFilterWhere(f Filter) (string, []any) {
	var conds []string
	var args []any

	if f.DateFrom != "" {
		conds = append(conds, "r.date >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		conds = append(conds, "r.date <= ?")
		args = append(args, f.DateTo)
	}
	if f.AmountFrom != nil {
		conds = append(conds, "r.total >= ?")
		args = append(args, *f.AmountFrom)
	}
	if f.AmountTo != nil {
		conds = append(conds, "r.total <= ?")
		args = append(args, *f.AmountTo)
	}
	if f.OrganizationID > 0 {
		conds = append(conds, "r.organization_id = ?")
		args = append(args, f.OrganizationID)
	}
	if f.CustomerID > 0 {
		conds = append(conds, "r.customer_id = ?")
		args = append(args, f.CustomerID)
	}

	if cond, cargs := statusFilterWhere(f.Status); cond != "" {
		conds = append(conds, cond)
		args = append(args, cargs...)
	}

	if f.ActionSetFrom != "" {
		conds = append(conds, "ra.action_set_at >= ?")
		args = append(args, f.ActionSetFrom+" 00:00:00")
	}
	if f.ActionSetTo != "" {
		conds = append(conds, "ra.action_set_at < ?")
		args = append(args, actionSetToUpperBound(f.ActionSetTo))
	}

	if len(conds) == 0 {
		return "", nil
	}
	return "(" + strings.Join(conds, " AND ") + ")", args
}

// actionSetToUpperBound возвращает верхнюю исключающую границу даты
// установки действия: следующий день после переданной даты в 00:00:00.
// Индекс-friendly: сравнение по диапазону без SQL-функции date().
func actionSetToUpperBound(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02") + " 00:00:00"
}

// statusFilterWhere возвращает условие отбора по отображаемому статусу.
//
// Зеркалит statusPresentation: пустой статус (legacy-данные) трактуется
// как «Создан» при отсутствии sent_at и как «Отправлен» при наличии.
// «Просрочен» — производное состояние StatusCreated старше 24 часов:
//
//	norm = 'Создан' AND datetime(created_at) <= datetime('now', '-24 hours')
//
// Неизвестные значения игнорируются (условие не добавляется).
func statusFilterWhere(status string) (string, []any) {
	switch status {
	case "":
		return "", nil
	case StatusCreated, OverdueStatus:
		created := receiptStatusNorm() + ` = '` + StatusCreated + `'`
		overdue := `datetime(r.created_at) <= datetime('now', '-24 hours')`
		if status == OverdueStatus {
			return `(` + created + ` AND ` + overdue + `)`, nil
		}
		return `(` + created + ` AND NOT (` + overdue + `))`, nil
	case StatusSent, StatusAccepted, StatusCancelled, StatusProcessed, StatusFinished:
		return receiptStatusNorm() + ` = '` + status + `'`, nil
	default:
		return "", nil
	}
}

// receiptStatusNorm — SQL-выражение нормализации статуса к отображаемому
// значению: пустой статус (legacy) зависит от sent_at.
func receiptStatusNorm() string {
	return `CASE WHEN r.status = '' THEN CASE WHEN r.sent_at IS NULL THEN '` + StatusCreated + `' ELSE '` + StatusSent + `' END ELSE r.status END`
}

func (s *Store) Save(ctx context.Context, doc *Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := doc.Receipt

	if r.ID == 0 {
		// Allocate the number only inside the insert transaction. Opening an
		// editor or returning a validation error must not consume a number.
		if r.Number == "" {
			var next int64
			if err := tx.QueryRowContext(ctx, `
				SELECT COALESCE(MAX(CAST(number AS INTEGER)), 0) + 1
				FROM receipts
				WHERE organization_id = ?
			`, r.OrganizationID).Scan(&next); err != nil {
				return err
			}
			r.Number = fmt.Sprintf("%06d", next)
		}
		uuid, err := common.GenerateUUID()
		if err != nil {
			return err
		}
		r.ExchangeID = uuid

		now := time.Now()
		r.CreatedAt = now
		r.UpdatedAt = now

		var uuidArg any
		if r.UUID != "" {
			uuidArg = r.UUID
		}

		var sentAtArg any
		if r.SentAt != nil {
			sentAtArg = r.SentAt.Format(time.RFC3339)
		}

		result, err := tx.ExecContext(ctx, `
			INSERT INTO receipts (uuid, exchange_id, number, date,
				organization_id, user_id, customer_id, total, sent_at,
				status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuidArg, r.ExchangeID, r.Number, r.Date.Format("2006-01-02"),
			r.OrganizationID, r.UserID, r.CustomerID, r.Total,
			sentAtArg, r.Status,
			r.CreatedAt.Format(time.RFC3339), r.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		r.ID = id
	} else {
		r.UpdatedAt = time.Now()

		var uuidArg any
		if r.UUID != "" {
			uuidArg = r.UUID
		}

		var sentAtArg any
		if r.SentAt != nil {
			sentAtArg = r.SentAt.Format(time.RFC3339)
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE receipts SET
				uuid = ?, number = ?, date = ?,
				organization_id = ?, user_id = ?, customer_id = ?,
				total = ?, sent_at = ?, status = ?,
				updated_at = ?
			WHERE id = ?
		`, uuidArg, r.Number, r.Date.Format("2006-01-02"),
			r.OrganizationID, r.UserID, r.CustomerID, r.Total,
			sentAtArg, r.Status,
			r.UpdatedAt.Format(time.RFC3339), r.ID)
		if err != nil {
			return err
		}
	}

	if err := saveItemsTx(tx, r.ID, doc.Items); err != nil {
		return err
	}

	return tx.Commit()
}

func saveItemsTx(tx *sql.Tx, receiptID int64, items []ReceiptItem) error {
	if _, err := tx.ExecContext(context.Background(),
		`DELETE FROM receipt_items WHERE receipt_id = ?`, receiptID); err != nil {
		return err
	}

	for _, item := range items {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO receipt_items
				(receipt_id, line_num, product_id, unit, quantity, price, amount)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, receiptID, item.LineNum, item.ProductID, item.Unit,
			item.Quantity, item.Price, item.Amount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// FK с ON DELETE CASCADE не задействован (PRAGMA foreign_keys не
	// включается в проекте), поэтому связанное действие удаляется явно.
	if _, err := tx.ExecContext(ctx, `DELETE FROM receipt_actions WHERE receipt_id = ?`, id); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM receipts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// MarkDeleted помечает документ на удаление. В одной транзакции основной
// БД устанавливается deleted_at, статус приводится к StatusCancelled,
// а признак отправки снимается (sent_at = NULL). После commit документ
// не участвует в бизнес-операциях, не попадает в очередь синхронизации
// и не может быть изменён через Integration API. Физическое удаление
// выполняется отдельным этапом и останавливается при ошибке, не откатывая
// пометку.
//
// Транзакция берёт write-lock сразу (BEGIN IMMEDIATE), поэтому проверка
// статуса и пометка атомарны: конкурентное интеграционное обновление не
// может проскочить между чтением и записью и обойти ReceiptDeletable.
//
// Повторная пометка возвращает ErrNotFound: признак deleted_at имеет
// приоритет над проверкой статуса (в том числе для документа со статусом
// StatusCancelled).
func (s *Store) MarkDeleted(ctx context.Context, id int64) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), `ROLLBACK`)

	var status string
	var deletedAt sql.NullTime
	err = conn.QueryRowContext(ctx,
		`SELECT status, deleted_at FROM receipts WHERE id = ?`, id).Scan(&status, &deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if deletedAt.Valid {
		return ErrNotFound
	}
	if !ReceiptDeletable(status) {
		return ErrReceiptNotDeletable
	}

	if _, err := conn.ExecContext(ctx, `
		UPDATE receipts
		SET deleted_at = CURRENT_TIMESTAMP,
		    status = ?,
		    sent_at = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, StatusCancelled, id); err != nil {
		return err
	}

	// Связанное действие исчезает вместе с документом (soft delete):
	// FK-каскад не срабатывает, поэтому удаляем явно в той же транзакции.
	if _, err := conn.ExecContext(ctx, `DELETE FROM receipt_actions WHERE receipt_id = ?`, id); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	return nil
}

func (s *Store) Synchronize(ctx context.Context, updates []ReceiptUpdate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, upd := range updates {
		var currentUUID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT uuid FROM receipts WHERE exchange_id = ? AND deleted_at IS NULL`, upd.ExchangeID).Scan(&currentUUID)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrExchangeIDNotFound
			}
			return err
		}

		if upd.UUID != nil {
			if currentUUID.Valid && *upd.UUID != currentUUID.String {
				return ErrUUIDAlreadyAssigned
			}
		}

		var sets []string
		var args []any

		if upd.UUID != nil {
			sets = append(sets, "uuid = ?")
			args = append(args, *upd.UUID)
		}
		if upd.Status != nil {
			sets = append(sets, "status = ?")
			args = append(args, *upd.Status)
		}

		if len(sets) == 0 {
			continue
		}

		sets = append(sets, "updated_at = ?")
		args = append(args, time.Now().Format(time.RFC3339))
		args = append(args, upd.ExchangeID)

		query := "UPDATE receipts SET "
		for i, set := range sets {
			if i > 0 {
				query += ", "
			}
			query += set
		}
		query += " WHERE exchange_id = ? AND deleted_at IS NULL"

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SyncResult — результат синхронизации чеков через Integration API.
type SyncResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
}

// SyncUpdate — обновление чека по внутреннему ID в рамках организации.
// UUID необязателен: его отсутствие оставляет документ в очереди
// синхронизации. Остальные поля заполняются частично.
type SyncUpdate struct {
	ID     int64
	UUID   *string
	Status *string
}

// ListAvailableForSync возвращает чеки организации, которые пользователь
// опубликовал (SentAt IS NOT NULL) и которые ещё не получили внешний UUID
// от 1С (UUID IS NULL). Это очередь синхронизации для Integration API.
// Возвращает документы с положительными строками.
func (s *Store) ListAvailableForSync(ctx context.Context, orgID int64) ([]*Document, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			COALESCE(c.uuid, '') AS customer_uuid,
			r.total, r.sent_at, r.status,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		WHERE r.organization_id = ?
		  AND r.sent_at IS NOT NULL
		  AND r.uuid IS NULL
		  AND r.deleted_at IS NULL
		ORDER BY r.date DESC, r.id DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		r, err := scanReceiptWithCustomerUUID(rows)
		if err != nil {
			return nil, err
		}
		items, err := s.listItems(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		docs = append(docs, &Document{Receipt: r, Items: items})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if docs == nil {
		docs = []*Document{}
	}
	return docs, nil
}

// SynchronizeByID применяет обновления к чекам по внутреннему ID,
// ограниченным организацией. Частичные обновления: заполняются только
// переданные поля. UUID подчиняется инварианту:
//
//	nil  → можно назначить (документ покидает очередь)
//	X    → повторная передача X (идемпотентно)
//	X≠Y  → ошибка ErrUUIDAlreadyAssigned (uuid неизменяем)
//
// Строка без полей пропускается. Все операции в рамках одного вызова
// атомарны.
func (s *Store) SynchronizeByID(ctx context.Context, orgID int64, updates []SyncUpdate) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	var result SyncResult
	for _, upd := range updates {
		var currentUUID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT uuid FROM receipts WHERE id = ? AND organization_id = ? AND deleted_at IS NULL`,
			upd.ID, orgID).Scan(&currentUUID)
		if err != nil {
			if err == sql.ErrNoRows {
				return SyncResult{}, ErrNotFound
			}
			return SyncResult{}, err
		}

		if upd.UUID != nil && *upd.UUID != "" {
			if currentUUID.Valid && *upd.UUID != currentUUID.String {
				return SyncResult{}, ErrUUIDAlreadyAssigned
			}
		}

		var sets []string
		var args []any

		if upd.UUID != nil {
			if *upd.UUID == "" {
				return SyncResult{}, ErrEmptyUUID
			}
			sets = append(sets, "uuid = ?")
			args = append(args, *upd.UUID)
		}
		if upd.Status != nil {
			sets = append(sets, "status = ?")
			args = append(args, *upd.Status)
		}

		if len(sets) == 0 {
			continue
		}

		sets = append(sets, "updated_at = ?")
		args = append(args, time.Now().Format(time.RFC3339))
		args = append(args, upd.ID, orgID)

		query := "UPDATE receipts SET "
		for i, set := range sets {
			if i > 0 {
				query += ", "
			}
			query += set
		}
		query += " WHERE id = ? AND organization_id = ? AND deleted_at IS NULL"

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return SyncResult{}, err
		}
		result.Updated++
	}

	return result, tx.Commit()
}

// UpdateByExternal обновляет статус чека по внешнему UUID
// (receipts.uuid) в рамках организации. Частичное обновление: заполняются
// только переданные поля, остальные не сбрасываются.
func (s *Store) UpdateByExternal(ctx context.Context, orgID int64, externalUUID string, status *string) error {
	var sets []string
	var args []any

	if status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *status)
	}
	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().Format(time.RFC3339))
	args = append(args, orgID, externalUUID)

	query := "UPDATE receipts SET "
	for i, set := range sets {
		if i > 0 {
			query += ", "
		}
		query += set
	}
	query += " WHERE organization_id = ? AND uuid = ? AND deleted_at IS NULL"

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetAction устанавливает текущее действие документа для 1С. Пустое
// значение означает отмену ранее установленного действия: если запись
// существует и действие непустое, она очищается (action пустой,
// action_set_at=now(), action_received_at=NULL); если записи нет или
// действие уже пустое — ничего не происходит.
// Непустое значение — upsert единственной записи с новым action,
// action_set_at=now() и action_received_at=NULL. Повторный выбор того же
// действия — no-op: запись не меняется (action_set_at и
// action_received_at сохраняются, действие не возвращается в очередь).
func (s *Store) SetAction(ctx context.Context, id int64, action string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	if action == "" {
		_, err := s.db.ExecContext(ctx, `
			UPDATE receipt_actions
			SET action = '', action_set_at = ?, action_received_at = NULL
			WHERE receipt_id = ? AND action <> ''
		`, now, id)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO receipt_actions (receipt_id, action, action_set_at, action_received_at)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(receipt_id) DO UPDATE SET
			action = excluded.action,
			action_set_at = excluded.action_set_at,
			action_received_at = NULL
		WHERE receipt_actions.action <> excluded.action
	`, id, action, now)
	return err
}

// GetAction возвращает текущее действие документа и дату получения.
// action пустой — действия нет; receivedAt nil — ещё не получено 1С.
func (s *Store) GetAction(ctx context.Context, id int64) (string, *time.Time, error) {
	var action sql.NullString
	var receivedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT action, action_received_at
		FROM receipt_actions
		WHERE receipt_id = ?
	`, id).Scan(&action, &receivedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, nil
		}
		return "", nil, err
	}
	var recv *time.Time
	if receivedAt.Valid {
		recv = &receivedAt.Time
	}
	return action.String, recv, nil
}

// Action — текущее действие документа для Integration API.
type Action struct {
	UUID   string `json:"uuid"`
	Number string `json:"number"`
	Date   string `json:"date"`
	Action string `json:"action"`
}

// ListActionsForSync возвращает ожидающие действия организации: все записи
// receipt_actions с action_received_at IS NULL, включая отменённые
// (action пустой), которые передаются в 1С как отмена ранее установленного
// действия.
func (s *Store) ListActionsForSync(ctx context.Context, orgID int64) ([]Action, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.uuid, r.number, r.date, ra.action
		FROM receipts r
		JOIN receipt_actions ra ON ra.receipt_id = r.id
		WHERE r.organization_id = ?
		  AND r.deleted_at IS NULL
		  AND ra.action_received_at IS NULL
		ORDER BY r.id DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var a Action
		if err := rows.Scan(&a.UUID, &a.Number, &a.Date, &a.Action); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if actions == nil {
		actions = []Action{}
	}
	return actions, nil
}

// ConfirmActions подтверждает получение действий 1С по внешнему uuid
// документа: устанавливает action_received_at=now() на текущей записи.
// Идемпотентно: уже полученное или отсутствующее действие пропускается.
// Документ не найден в организации (или удалён) — ErrNotFound. Операция
// атомарна для всего массива.
func (s *Store) ConfirmActions(ctx context.Context, orgID int64, uuids []string) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	var result SyncResult
	for _, uuid := range uuids {
		var receiptID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM receipts
			WHERE organization_id = ? AND uuid = ? AND deleted_at IS NULL
		`, orgID, uuid).Scan(&receiptID)
		if err != nil {
			if err == sql.ErrNoRows {
				return SyncResult{}, ErrNotFound
			}
			return SyncResult{}, err
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE receipt_actions
			SET action_received_at = ?
			WHERE receipt_id = ? AND action_received_at IS NULL
		`, time.Now().UTC().Format("2006-01-02 15:04:05"), receiptID)
		if err != nil {
			return SyncResult{}, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return SyncResult{}, err
		}
		if n > 0 {
			result.Updated++
		}
	}

	return result, tx.Commit()
}

func (s *Store) listItems(ctx context.Context, receiptID int64) ([]ReceiptItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			i.id, i.receipt_id, i.line_num, i.product_id,
			COALESCE(p.uuid, '') AS product_uuid,
			COALESCE(p.name, '') AS product_name,
			i.unit, i.quantity, i.price, i.amount
		FROM receipt_items i
		LEFT JOIN products p ON p.id = i.product_id
		WHERE i.receipt_id = ?
		ORDER BY i.line_num
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ReceiptItem
	for rows.Next() {
		var item ReceiptItem
		if err := rows.Scan(
			&item.ID, &item.ReceiptID, &item.LineNum, &item.ProductID,
			&item.ProductUUID, &item.ProductName, &item.Unit, &item.Quantity, &item.Price, &item.Amount,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if items == nil {
		items = []ReceiptItem{}
	}
	return items, nil
}

func scanReceipt(row interface {
	Scan(dest ...any) error
}) (*Receipt, error) {
	r := &Receipt{}

	var uuid sql.NullString
	var sentAt sql.NullTime
	var dateStr, createdAt, updatedAt string

	err := row.Scan(
		&r.ID, &uuid, &r.ExchangeID, &r.Number, &dateStr,
		&r.OrganizationID, &r.OrganizationName,
		&r.UserID, &r.UserLogin,
		&r.CustomerID, &r.CustomerName,
		&r.Total, &sentAt, &r.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if uuid.Valid {
		r.UUID = uuid.String
	}
	if sentAt.Valid {
		r.SentAt = &sentAt.Time
	}
	if dateStr != "" {
		r.Date, _ = time.Parse("2006-01-02", dateStr[:10])
	}
	r.CreatedAt, _ = parseReceiptTime(createdAt)
	r.UpdatedAt, _ = parseReceiptTime(updatedAt)

	return r, nil
}

// scanReceiptWithAction — как scanReceipt, но дополнительно читает текущее
// действие документа (receipt_actions.action, action_received_at) для списка.
func scanReceiptWithAction(row interface {
	Scan(dest ...any) error
}) (*Receipt, error) {
	r := &Receipt{}

	var uuid sql.NullString
	var sentAt sql.NullTime
	var action sql.NullString
	var actionReceivedAt sql.NullTime
	var dateStr, createdAt, updatedAt string

	err := row.Scan(
		&r.ID, &uuid, &r.ExchangeID, &r.Number, &dateStr,
		&r.OrganizationID, &r.OrganizationName,
		&r.UserID, &r.UserLogin,
		&r.CustomerID, &r.CustomerName,
		&r.Total, &sentAt, &r.Status,
		&action, &actionReceivedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if uuid.Valid {
		r.UUID = uuid.String
	}
	if sentAt.Valid {
		r.SentAt = &sentAt.Time
	}
	if action.Valid {
		r.Action = action.String
	}
	if actionReceivedAt.Valid {
		r.ActionReceivedAt = &actionReceivedAt.Time
	}
	if dateStr != "" {
		r.Date, _ = time.Parse("2006-01-02", dateStr[:10])
	}
	r.CreatedAt, _ = parseReceiptTime(createdAt)
	r.UpdatedAt, _ = parseReceiptTime(updatedAt)

	return r, nil
}

// scanReceiptWithCustomerUUID — как scanReceipt, но дополнительно читает
// внешний UUID клиента (receipts.customer_uuid) для Integration API.
func scanReceiptWithCustomerUUID(row interface {
	Scan(dest ...any) error
}) (*Receipt, error) {
	r := &Receipt{}

	var uuid sql.NullString
	var sentAt sql.NullTime
	var dateStr, createdAt, updatedAt string

	err := row.Scan(
		&r.ID, &uuid, &r.ExchangeID, &r.Number, &dateStr,
		&r.OrganizationID, &r.OrganizationName,
		&r.UserID, &r.UserLogin,
		&r.CustomerID, &r.CustomerName,
		&r.CustomerUUID,
		&r.Total, &sentAt, &r.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if uuid.Valid {
		r.UUID = uuid.String
	}
	if sentAt.Valid {
		r.SentAt = &sentAt.Time
	}
	if dateStr != "" {
		r.Date, _ = time.Parse("2006-01-02", dateStr[:10])
	}
	r.CreatedAt, _ = parseReceiptTime(createdAt)
	r.UpdatedAt, _ = parseReceiptTime(updatedAt)

	return r, nil
}

// parseReceiptTime разбирает время хранения SQLite: RFC3339 (пишется кодом)
// или "2006-01-02 15:04:05" (CURRENT_TIMESTAMP). Ошибка невозможна при
// корректных данных; при неверном формате возвращается нулевое время.
func parseReceiptTime(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", value)
}
