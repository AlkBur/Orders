package users

import (
	"context"
	"testing"
)

func TestNormalizeLogin(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Admin", "admin"},
		{"ADMIN", "admin"},
		{"  Admin  ", "admin"},
		{"Alexander", "alexander"},
		{"Ivan.Petrov", "ivan.petrov"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		got := NormalizeLogin(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeLogin(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNewIdentity(t *testing.T) {
	u := &User{
		ID:           42,
		UUID:         "u-1",
		Login:        "Alexander",
		PasswordHash: "hash",
		IsAdmin:      true,
	}

	id := NewIdentity(u)

	if id.ID != 42 {
		t.Fatalf("expected ID 42, got %d", id.ID)
	}
	if id.UUID != "u-1" {
		t.Fatalf("expected UUID u-1, got %s", id.UUID)
	}
	if id.Login != "Alexander" {
		t.Fatalf("expected Login Alexander, got %s", id.Login)
	}
	if id.NormalizedLogin != "alexander" {
		t.Fatalf("expected NormalizedLogin alexander, got %s", id.NormalizedLogin)
	}
	if id.PasswordHash != "hash" {
		t.Fatalf("expected PasswordHash hash, got %s", id.PasswordHash)
	}
	if !id.IsAdmin {
		t.Fatal("expected IsAdmin true")
	}
}

func TestNeedsPasswordSetup(t *testing.T) {
	var id Identity
	if !id.NeedsPasswordSetup() {
		t.Fatal("expected NeedsPasswordSetup true for empty hash")
	}
	id.PasswordHash = "hash"
	if id.NeedsPasswordSetup() {
		t.Fatal("expected NeedsPasswordSetup false for non-empty hash")
	}
}

func TestIdentityService_AddAndGet(t *testing.T) {
	svc := NewIdentityService()

	user := &User{ID: 1, UUID: "u-1", Login: "Admin", IsAdmin: true}
	svc.Add(user)

	id, ok := svc.GetByID(1)
	if !ok {
		t.Fatal("expected to find identity by ID")
	}
	if id.Login != "Admin" {
		t.Fatalf("expected Login Admin, got %s", id.Login)
	}
	if id.NormalizedLogin != "admin" {
		t.Fatalf("expected NormalizedLogin admin, got %s", id.NormalizedLogin)
	}
	if !id.IsAdmin {
		t.Fatal("expected IsAdmin true")
	}

	id2, ok := svc.GetByLogin("admin")
	if !ok {
		t.Fatal("expected to find identity by login")
	}
	if id2.ID != 1 {
		t.Fatalf("expected ID 1, got %d", id2.ID)
	}
}

func TestIdentityService_GetByLogin_Normalized(t *testing.T) {
	svc := NewIdentityService()

	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Alexander"})

	_, ok := svc.GetByLogin("Alexander")
	if !ok {
		t.Fatal("expected to find by original case")
	}

	_, ok = svc.GetByLogin("ALEXANDER")
	if !ok {
		t.Fatal("expected to find by upper case")
	}

	_, ok = svc.GetByLogin("  alexander  ")
	if !ok {
		t.Fatal("expected to find with spaces")
	}
}

func TestIdentityService_Update(t *testing.T) {
	svc := NewIdentityService()

	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Admin", IsAdmin: false})
	svc.Update(&User{ID: 1, UUID: "u-1", Login: "Admin", IsAdmin: true})

	id, ok := svc.GetByID(1)
	if !ok {
		t.Fatal("expected to find after update")
	}
	if !id.IsAdmin {
		t.Fatal("expected IsAdmin true after update")
	}
}

func TestIdentityService_Update_LoginChange(t *testing.T) {
	svc := NewIdentityService()

	svc.Add(&User{ID: 1, UUID: "u-1", Login: "OldLogin", PasswordHash: "hash"})

	svc.Update(&User{ID: 1, UUID: "u-1", Login: "NewLogin", PasswordHash: "hash"})

	_, ok := svc.GetByLogin("oldlogin")
	if ok {
		t.Fatal("expected old login to be removed")
	}

	id, ok := svc.GetByLogin("newlogin")
	if !ok {
		t.Fatal("expected to find by new login")
	}
	if id.Login != "NewLogin" {
		t.Fatalf("expected Login NewLogin, got %s", id.Login)
	}
}

func TestIdentityService_Remove(t *testing.T) {
	svc := NewIdentityService()

	svc.Add(&User{ID: 1, UUID: "u-1", Login: "User1"})
	svc.Add(&User{ID: 2, UUID: "u-2", Login: "User2"})

	svc.Remove(1)

	_, ok := svc.GetByID(1)
	if ok {
		t.Fatal("expected ID 1 to be removed")
	}

	_, ok = svc.GetByLogin("user1")
	if ok {
		t.Fatal("expected user1 to be removed from byLogin")
	}

	_, ok = svc.GetByID(2)
	if !ok {
		t.Fatal("expected ID 2 to remain")
	}
}

func TestIdentityService_Remove_NonExistent(t *testing.T) {
	svc := NewIdentityService()

	svc.Remove(999)
}

func TestIdentityService_IsLoginTaken(t *testing.T) {
	svc := NewIdentityService()

	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Existing"})

	if !svc.IsLoginTaken("existing", 2) {
		t.Fatal("expected existing login to be taken for different user")
	}

	if svc.IsLoginTaken("existing", 1) {
		t.Fatal("expected existing login NOT to be taken for same user")
	}

	if svc.IsLoginTaken("unknown", 0) {
		t.Fatal("expected unknown login not to be taken")
	}

	if svc.IsLoginTaken("", 0) {
		t.Fatal("expected empty login not to be taken")
	}
}

func TestIdentityService_GetByID_NotFound(t *testing.T) {
	svc := NewIdentityService()

	_, ok := svc.GetByID(999)
	if ok {
		t.Fatal("expected not found for non-existent ID")
	}
}

func TestIdentityService_GetByLogin_NotFound(t *testing.T) {
	svc := NewIdentityService()

	_, ok := svc.GetByLogin("nobody")
	if ok {
		t.Fatal("expected not found for non-existent login")
	}
}

func TestIdentityService_AtomicLoad(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	store.Create(&User{UUID: "load-1", Login: "Alpha", IsAdmin: true})
	store.Create(&User{UUID: "load-2", Login: "Beta", IsAdmin: false})

	svc := NewIdentityService()

	if err := svc.Load(context.Background(), store); err != nil {
		t.Fatal(err)
	}

	id, ok := svc.GetByID(1)
	if !ok {
		t.Fatal("expected to find user 1 after Load")
	}
	if id.Login != "Alpha" {
		t.Fatalf("expected Login Alpha, got %s", id.Login)
	}
	if !id.IsAdmin {
		t.Fatal("expected IsAdmin true")
	}

	id, ok = svc.GetByLogin("beta")
	if !ok {
		t.Fatal("expected to find beta after Load")
	}
	if id.Login != "Beta" {
		t.Fatalf("expected Login Beta, got %s", id.Login)
	}
	if id.IsAdmin {
		t.Fatal("expected IsAdmin false")
	}
}

func TestIdentityService_LoadReplacesExisting(t *testing.T) {
	db := testDB(t)
	store := NewStore(db)

	store.Create(&User{UUID: "load-rep-1", Login: "OldOne"})

	svc := NewIdentityService()
	svc.Add(&User{ID: 999, UUID: "ghost", Login: "Ghost"})

	if err := svc.Load(context.Background(), store); err != nil {
		t.Fatal(err)
	}

	_, ok := svc.GetByLogin("ghost")
	if ok {
		t.Fatal("expected ghost to be removed after Load")
	}

	_, ok = svc.GetByLogin("oldone")
	if !ok {
		t.Fatal("expected oldone to exist after Load")
	}
}

func TestIdentityService_IsLastAdministrator(t *testing.T) {
	svc := NewIdentityService()
	svc.Add(&User{ID: 1, UUID: "u-1", Login: "OnlyAdmin", IsAdmin: true})

	if !svc.IsLastAdministrator(1) {
		t.Fatal("expected single admin to be last administrator")
	}

	if svc.IsLastAdministrator(999) {
		t.Fatal("expected non-existent ID not to be last administrator")
	}
}

func TestIdentityService_IsLastAdministrator_NonAdmin(t *testing.T) {
	svc := NewIdentityService()
	svc.Add(&User{ID: 1, UUID: "u-1", Login: "User"})

	if svc.IsLastAdministrator(1) {
		t.Fatal("expected non-admin not to be last administrator")
	}
}

func TestIdentityService_IsLastAdministrator_TwoAdmins(t *testing.T) {
	svc := NewIdentityService()
	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Admin1", IsAdmin: true})
	svc.Add(&User{ID: 2, UUID: "u-2", Login: "Admin2", IsAdmin: true})

	if svc.IsLastAdministrator(1) {
		t.Fatal("expected admin1 not to be last when there are two admins")
	}
	if svc.IsLastAdministrator(2) {
		t.Fatal("expected admin2 not to be last when there are two admins")
	}
}

func TestIdentityService_IsLastAdministrator_AfterRemoval(t *testing.T) {
	svc := NewIdentityService()
	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Admin1", IsAdmin: true})
	svc.Add(&User{ID: 2, UUID: "u-2", Login: "Admin2", IsAdmin: true})

	svc.Remove(2)

	if !svc.IsLastAdministrator(1) {
		t.Fatal("expected admin1 to be last after admin2 removed")
	}
}

func TestIdentityService_IsLastAdministrator_AfterRoleRevoke(t *testing.T) {
	svc := NewIdentityService()
	svc.Add(&User{ID: 1, UUID: "u-1", Login: "Admin1", IsAdmin: true})
	svc.Add(&User{ID: 2, UUID: "u-2", Login: "Admin2", IsAdmin: true})

	svc.Update(&User{ID: 2, UUID: "u-2", Login: "Admin2", IsAdmin: false})

	if !svc.IsLastAdministrator(1) {
		t.Fatal("expected admin1 to be last after admin2 role revoked")
	}
	if svc.IsLastAdministrator(2) {
		t.Fatal("expected former admin2 not to be last administrator")
	}
}
