package sessions

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

var idleTimeout = 8 * time.Hour

type Flash struct {
	Type    string
	Message string
}

type Session struct {
	ID        string
	UserID    *int64
	Flash     *Flash
	Values    map[string]any
	UserAgent string
	CreatedAt time.Time
	LastSeen  time.Time
	ExpiresAt time.Time
}

func (s *Session) SetFlash(ftype, fmsg string) {
	s.Flash = &Flash{Type: ftype, Message: fmsg}
}

func (s *Session) ClearFlash() {
	s.Flash = nil
}

func (s *Session) SetValue(key string, val any) {
	if s.Values == nil {
		s.Values = make(map[string]any)
	}
	s.Values[key] = val
}

func (s *Session) RemoveValue(key string) {
	delete(s.Values, key)
}

type Store struct {
	mu        sync.Mutex
	db        *sql.DB
	opsCount  int64
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func generateID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *Store) Create(userID int64, ua string) (*Session, error) {
	session := &Session{
		ID:        generateID(),
		UserID:    &userID,
		Values:    make(map[string]any),
		UserAgent: ua,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		ExpiresAt: time.Now().Add(idleTimeout),
	}

	if err := s.Save(session); err != nil {
		return nil, err
	}

	s.maybeCleanup()
	return session, nil
}

func (s *Store) Save(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valuesJSON, err := json.Marshal(session.Values)
	if err != nil {
		return err
	}

	flashType := ""
	flashMessage := ""
	if session.Flash != nil {
		flashType = session.Flash.Type
		flashMessage = session.Flash.Message
	}

	_, err = s.db.Exec(`
INSERT INTO sessions (
    id, user_id, flash_type, flash_message, values_json,
    user_agent, created_at, last_seen_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    user_id      = excluded.user_id,
    flash_type   = excluded.flash_type,
    flash_message = excluded.flash_message,
    values_json  = excluded.values_json,
    user_agent   = excluded.user_agent,
    last_seen_at = excluded.last_seen_at,
    expires_at   = excluded.expires_at
`,
		session.ID,
		session.UserID,
		flashType,
		flashMessage,
		string(valuesJSON),
		session.UserAgent,
		session.CreatedAt.Format(time.RFC3339),
		session.LastSeen.Format(time.RFC3339),
		session.ExpiresAt.Format(time.RFC3339),
	)

	return err
}

func (s *Store) FindByID(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRow(`
SELECT
    id, user_id, flash_type, flash_message, values_json,
    user_agent, created_at, last_seen_at, expires_at
FROM sessions
WHERE id = ?
`, id)

	session := &Session{}
	var userID sql.NullInt64
	var flashType, flashMessage, valuesJSON string
	var createdAt, lastSeenAt, expiresAt string

	err := row.Scan(
		&session.ID,
		&userID,
		&flashType,
		&flashMessage,
		&valuesJSON,
		&session.UserAgent,
		&createdAt,
		&lastSeenAt,
		&expiresAt,
	)
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		session.UserID = &userID.Int64
	}

	if flashType != "" || flashMessage != "" {
		session.Flash = &Flash{Type: flashType, Message: flashMessage}
	}

	if valuesJSON != "" && valuesJSON != "{}" {
		json.Unmarshal([]byte(valuesJSON), &session.Values)
	}
	if session.Values == nil {
		session.Values = make(map[string]any)
	}

	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	session.LastSeen, _ = time.Parse(time.RFC3339, lastSeenAt)
	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	return session, nil
}

func (s *Store) Touch(session *Session) {
	now := time.Now()
	if now.Sub(session.LastSeen) < 5*time.Minute {
		return
	}
	session.LastSeen = now
	if now.Add(idleTimeout / 2).After(session.ExpiresAt) {
		session.ExpiresAt = now.Add(idleTimeout)
	}
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteAllByUserID(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Store) maybeCleanup() {
	s.mu.Lock()
	s.opsCount++
	count := s.opsCount
	s.mu.Unlock()

	if count%100 != 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
}
