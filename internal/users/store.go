package users

import (
	"context"
	"database/sql"

	"Orders/internal/database/search"
	"Orders/internal/entity"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *User {
	return &User{}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.FindByID(id)
}

func (s *Store) GetByUUID(ctx context.Context, uuid string) (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT id, uuid, login, email, password_hash, is_admin, created_at, updated_at
		FROM users
		WHERE uuid = ?
	`, uuid))
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// ListOptions управляет выборкой списка пользователей.
type ListOptions struct {
	Query   string
	Limit   int
	Offset  int
	OrderBy string
}

// userSearchColumns — поисковые колонки списка пользователей.
var userSearchColumns = []search.MappedColumn{
	{Field: entity.FieldNameLogin, Expression: "login"},
	{Field: entity.FieldNameEmail, Expression: "email"},
}

func (s *Store) searchableColumns() []search.MappedColumn {
	return userSearchColumns
}

// List возвращает пользователей. visibleFields — поля, отображаемые в списке:
// поиск выполняется только по ним.
func (s *Store) List(ctx context.Context, opts ListOptions, visibleFields []entity.FieldName) ([]*User, error) {
	query := `
		SELECT id, uuid, login, email, password_hash, is_admin, created_at, updated_at
		FROM users
	`
	var args []any

	where, whereArgs := search.BuildWhere(
		search.VisibleColumns(s.searchableColumns(), visibleFields),
		search.NormalizeQuery(opts.Query),
	)
	if where != "" {
		query += ` WHERE ` + where
		args = append(args, whereArgs...)
	}
	query += ` ORDER BY login`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.UUID, &u.Login, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.HasPassword = u.PasswordHash != ""
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if users == nil {
		users = []*User{}
	}

	return users, nil
}

// Save inserts or updates a user.
//
// INSERT (ID == 0):
//   - UUID must already be assigned.
//   - Returns ErrEmptyUUID if UUID is empty.
//
// UPDATE (ID > 0):
//   - Updates the existing record by ID.
func (s *Store) Save(ctx context.Context, user *User) error {
	if user.ID == 0 {
		if user.UUID == "" {
			return ErrEmptyUUID
		}

		result, err := s.db.ExecContext(ctx, `
			INSERT INTO users (uuid, login, email, password_hash, is_admin, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, user.UUID, user.Login, user.Email, user.PasswordHash, user.IsAdmin)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		user.ID = id
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET uuid = ?, login = ?, email = ?, password_hash = ?, is_admin = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, user.UUID, user.Login, user.Email, user.PasswordHash, user.IsAdmin, user.ID)
	return err
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
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
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE uuid = ?`, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Create(user *User) error {
	if user.UUID == "" {
		return ErrEmptyUUID
	}

	result, err := s.db.Exec(`
		INSERT INTO users (uuid, login, email, password_hash, is_admin)
		VALUES (?, ?, ?, ?, ?)
	`, user.UUID, user.Login, user.Email, user.PasswordHash, user.IsAdmin)
	if err != nil {
		return err
	}

	user.ID, err = result.LastInsertId()
	return err
}

func (s *Store) Update(user *User) error {
	_, err := s.db.Exec(`
		UPDATE users
		SET uuid = ?, login = ?, email = ?, password_hash = ?, is_admin = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, user.UUID, user.Login, user.Email, user.PasswordHash, user.IsAdmin, user.ID)
	return err
}

func (s *Store) FindByLogin(login string) (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT id, uuid, login, email, password_hash, is_admin, created_at, updated_at
		FROM users
		WHERE login = ?
	`, login))
}

func (s *Store) FindByID(id int64) (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT id, uuid, login, email, password_hash, is_admin, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id))
}

func (s *Store) FindAdmin() (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT id, uuid, login, email, password_hash, is_admin, created_at, updated_at
		FROM users
		WHERE is_admin = 1
		LIMIT 1
	`))
}

func (s *Store) Authenticate(login, password string) (*User, error) {
	user, err := s.FindByLogin(login)
	if err != nil {
		return nil, err
	}

	ok, err := user.VerifyPassword(password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func scanUser(row *sql.Row) (*User, error) {
	user := &User{}

	err := row.Scan(
		&user.ID,
		&user.UUID,
		&user.Login,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	user.HasPassword = user.PasswordHash != ""
	return user, nil
}
