package users

import (
	"database/sql"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create создает пользователя.
func (s *Store) Create(user *User) error {
	result, err := s.db.Exec(`
		INSERT INTO users (
			login,
			password_hash,
			is_admin
		)
		VALUES (?, ?, ?)
	`,
		user.Login,
		user.PasswordHash,
		user.IsAdmin,
	)
	if err != nil {
		return err
	}

	user.ID, err = result.LastInsertId()
	return err
}

// Update сохраняет изменения пользователя.
func (s *Store) Update(user *User) error {
	_, err := s.db.Exec(`
		UPDATE users
		SET
			login = ?,
			password_hash = ?,
			is_admin = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		user.Login,
		user.PasswordHash,
		user.IsAdmin,
		user.ID,
	)

	return err
}

// FindByLogin ищет пользователя по логину.
func (s *Store) FindByLogin(login string) (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT
			id,
			login,
			password_hash,
			is_admin,
			created_at,
			updated_at
		FROM users
		WHERE login = ?
	`, login))
}

// FindAdmin возвращает администратора.
func (s *Store) FindAdmin() (*User, error) {
	return scanUser(s.db.QueryRow(`
		SELECT
			id,
			login,
			password_hash,
			is_admin,
			created_at,
			updated_at
		FROM users
		WHERE is_admin = 1
		LIMIT 1
	`))
}

func scanUser(row *sql.Row) (*User, error) {
	user := &User{}

	err := row.Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
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

func (s *Store) FindByID(id int64) (*User, error) {

	row := s.db.QueryRow(`
		SELECT
			id,
			login,
			password_hash,
			is_admin,
			created_at,
			updated_at
		FROM users
		WHERE id = ?
	`, id)

	return scanUser(row)
}
