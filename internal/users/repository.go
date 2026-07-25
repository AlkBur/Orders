package users

import "database/sql"

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

func (s *Store) Count() (int, error) {
	var count int

	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM users`,
	).Scan(&count)

	return count, err
}

func (s *Store) Create(user *User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (
			login,
			password_hash,
			display_name,
			disabled
		)
		VALUES (?, ?, ?, ?)
	`,
		user.Login,
		user.PasswordHash,
		user.Name,
		user.Disabled,
	)

	return err
}

func (s *Store) FindByLogin(login string) (User, error) {
	var user User

	err := s.db.QueryRow(`
		SELECT
			id,
			login,
			password_hash,
			display_name,
			disabled
		FROM users
		WHERE login = ?
	`, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Name,
		&user.Disabled,
	)

	return user, err
}
