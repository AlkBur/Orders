package users

import (
	"context"
	"strings"
	"sync"
)

func NormalizeLogin(login string) string {
	return strings.TrimSpace(strings.ToLower(login))
}

type Identity struct {
	ID              int64
	UUID            string
	Login           string
	NormalizedLogin string
	PasswordHash    string
	IsAdmin         bool
}

func (i Identity) NeedsPasswordSetup() bool {
	return i.PasswordHash == ""
}

func NewIdentity(user *User) Identity {
	return Identity{
		ID:              user.ID,
		UUID:            user.UUID,
		Login:           user.Login,
		NormalizedLogin: NormalizeLogin(user.Login),
		PasswordHash:    user.PasswordHash,
		IsAdmin:         user.IsAdmin,
	}
}

type IdentityService struct {
	mu      sync.RWMutex
	byID    map[int64]Identity
	byLogin map[string]Identity
}

func NewIdentityService() *IdentityService {
	return &IdentityService{
		byID:    make(map[int64]Identity),
		byLogin: make(map[string]Identity),
	}
}

func (s *IdentityService) Load(ctx context.Context, store *Store) error {
	list, err := store.List(ctx, ListOptions{}, nil)
	if err != nil {
		return err
	}

	byID := make(map[int64]Identity, len(list))
	byLogin := make(map[string]Identity, len(list))

	for _, u := range list {
		id := NewIdentity(u)
		byID[id.ID] = id
		byLogin[id.NormalizedLogin] = id
	}

	s.mu.Lock()
	s.byID = byID
	s.byLogin = byLogin
	s.mu.Unlock()

	return nil
}

func (s *IdentityService) GetByID(id int64) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.byID[id]
	return v, ok
}

func (s *IdentityService) GetByLogin(login string) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.byLogin[NormalizeLogin(login)]
	return v, ok
}

func (s *IdentityService) IsLoginTaken(login string, excludeID int64) bool {
	normalized := NormalizeLogin(login)
	if normalized == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.byLogin[normalized]
	if !ok {
		return false
	}
	return v.ID != excludeID
}

func (s *IdentityService) upsert(user *User) {
	id := NewIdentity(user)

	s.mu.Lock()
	s.byID[id.ID] = id
	s.byLogin[id.NormalizedLogin] = id
	s.mu.Unlock()
}

func (s *IdentityService) Add(user *User) {
	s.upsert(user)
}

func (s *IdentityService) Update(user *User) {
	id := NewIdentity(user)

	s.mu.Lock()
	old, exists := s.byID[id.ID]
	if exists && old.NormalizedLogin != id.NormalizedLogin {
		delete(s.byLogin, old.NormalizedLogin)
	}
	s.byID[id.ID] = id
	s.byLogin[id.NormalizedLogin] = id
	s.mu.Unlock()
}

func (s *IdentityService) IsLastAdministrator(id int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	identity, ok := s.byID[id]
	if !ok || !identity.IsAdmin {
		return false
	}

	count := 0
	for _, u := range s.byID {
		if u.IsAdmin {
			count++
		}
	}
	return count == 1
}

func (s *IdentityService) Reload(ctx context.Context, store *Store) error {
	return s.Load(ctx, store)
}

func (s *IdentityService) Remove(id int64) {
	s.mu.RLock()
	v, ok := s.byID[id]
	s.mu.RUnlock()

	if !ok {
		return
	}

	s.mu.Lock()
	delete(s.byID, id)
	delete(s.byLogin, v.NormalizedLogin)
	s.mu.Unlock()
}
