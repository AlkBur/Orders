package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"

	"Orders/internal/database/search"
	"Orders/internal/entity"
)

// ListOptions управляет выборкой списка организаций.
// OrderBy добавлен заранее, чтобы не менять сигнатуру позже.
type ListOptions struct {
	Query   string
	Limit   int
	Offset  int
	OrderBy string
}

type Store struct {
	db *sql.DB
}

// organizationSearchColumns — поисковые колонки списка организаций.
// Expression возвращает ту же строку, что показывается в списке.
var organizationSearchColumns = []search.MappedColumn{
	{Field: entity.FieldNameName, Expression: "name"},
}

func (s *Store) searchableColumns() []search.MappedColumn {
	return organizationSearchColumns
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Organization {
	return &Organization{Active: true}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Organization, error) {
	o := &Organization{}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, uuid, name, api_key, active, created_at, updated_at
		FROM organizations
		WHERE id = ?
	`, id).Scan(
		&o.ID,
		&o.UUID,
		&o.Name,
		&o.APIKey,
		&o.Active,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Store) GetByUUID(ctx context.Context, uuid string) (*Organization, error) {
	o := &Organization{}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, uuid, name, api_key, active, created_at, updated_at
		FROM organizations
		WHERE uuid = ?
	`, uuid).Scan(
		&o.ID,
		&o.UUID,
		&o.Name,
		&o.APIKey,
		&o.Active,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return o, nil
}

// List возвращает организации. visibleFields — поля, отображаемые в списке
// (entity.FieldName по GoName): поиск выполняется только по ним.
func (s *Store) List(ctx context.Context, opts ListOptions, visibleFields []entity.FieldName) ([]*Organization, error) {
	q := `
		SELECT id, uuid, name, active, created_at, updated_at
		FROM organizations
	`
	var args []any

	where, whereArgs := search.BuildWhere(
		search.VisibleColumns(s.searchableColumns(), visibleFields),
		search.NormalizeQuery(opts.Query),
	)
	if where != "" {
		q += ` WHERE ` + where
		args = append(args, whereArgs...)
	}

	switch opts.OrderBy {
	case "created_at":
		q += ` ORDER BY created_at DESC`
	default:
		q += ` ORDER BY name`
	}

	if opts.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			q += ` OFFSET ?`
			args = append(args, opts.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		o := &Organization{}
		if err := rows.Scan(
			&o.ID,
			&o.UUID,
			&o.Name,
			&o.Active,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if orgs == nil {
		orgs = []*Organization{}
	}

	return orgs, nil
}

// Count возвращает число организаций.
func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// Save inserts or updates an organization.
//
// INSERT (ID == 0):
//   - UUID must already be assigned.
//   - Returns ErrEmptyUUID if UUID is empty.
//
// UPDATE (ID > 0):
//   - Updates the existing record by ID.
//   - Returns ErrNotFound if the record does not exist.
func (s *Store) Save(ctx context.Context, org *Organization) error {
	if org.ID == 0 {
		if org.UUID == "" {
			return ErrEmptyUUID
		}

		if org.APIKey == "" {
			key, err := generateAPIKey()
			if err != nil {
				return err
			}
			org.APIKey = key
		}

		result, err := s.db.ExecContext(ctx, `
			INSERT INTO organizations (uuid, name, api_key, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, org.UUID, org.Name, org.APIKey, org.Active)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		org.ID = id
		return nil
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE organizations
		SET uuid = ?, name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, org.UUID, org.Name, org.Active, org.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteByUUID(ctx context.Context, uuid string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM organizations WHERE uuid = ?`, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) LoadAPIKeys(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, api_key
		FROM organizations
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]string)
	for rows.Next() {
		var uuid, apiKey string
		if err := rows.Scan(&uuid, &apiKey); err != nil {
			return nil, err
		}
		keys[uuid] = apiKey
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ord_" + fmt.Sprintf("%x", b), nil
}
