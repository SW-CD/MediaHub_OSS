package utils

import (
	"context"
	"mediahub_oss/internal/repository"
)

type PermissionHolder interface {
	IsGlobalAdmin() bool
	CanView(database repository.ULID) bool
	CanCreate(database repository.ULID) bool
	CanEdit(database repository.ULID) bool
	CanDelete(database repository.ULID) bool
	CanAdmin(database repository.ULID) bool

	GetUserULID() repository.ULID
	GetAllPermissions(ctx context.Context) (map[repository.ULID]repository.AccessGrant, error)
}

// There are three types of permission holders:
// **GlobalAdmin**: has full access to all databases and actions.
// **APIKeyOfAdmin**: has access limited only by the scope of the API key
// **UserPermissions**: has access limited by the specific database permissions and a potential API key scope

type GlobalAdmin struct {
	UserULID repository.ULID
	Repo     repository.Repository
}

func (g *GlobalAdmin) IsGlobalAdmin() bool {
	return true
}

func (g *GlobalAdmin) CanView(database repository.ULID) bool {
	return true
}

func (g *GlobalAdmin) CanCreate(database repository.ULID) bool {
	return true
}

func (g *GlobalAdmin) CanEdit(database repository.ULID) bool {
	return true
}

func (g *GlobalAdmin) CanDelete(database repository.ULID) bool {
	return true
}

func (g *GlobalAdmin) CanAdmin(database repository.ULID) bool {
	return true
}

func (g *GlobalAdmin) GetUserULID() repository.ULID {
	return g.UserULID
}

func (g *GlobalAdmin) GetAllPermissions(ctx context.Context) (map[repository.ULID]repository.AccessGrant, error) {
	return map[repository.ULID]repository.AccessGrant{}, nil // Global admin has implicit access to all databases
}

// An API key without admin scope of an admin user
type APIKeyOfAdmin struct {
	UserULID repository.ULID
	Scope    repository.AccessGrant
	Repo     repository.Repository

	loaded    bool
	databases []repository.ULID
}

func (a *APIKeyOfAdmin) IsGlobalAdmin() bool {
	return false
}

func (a *APIKeyOfAdmin) CanView(database repository.ULID) bool {
	return a.Scope.HasAccess(repository.AccessView)
}

func (a *APIKeyOfAdmin) CanCreate(database repository.ULID) bool {
	return a.Scope.HasAccess(repository.AccessCreate)
}

func (a *APIKeyOfAdmin) CanEdit(database repository.ULID) bool {
	return a.Scope.HasAccess(repository.AccessEdit)
}

func (a *APIKeyOfAdmin) CanDelete(database repository.ULID) bool {
	return a.Scope.HasAccess(repository.AccessDelete)
}

func (a *APIKeyOfAdmin) CanAdmin(database repository.ULID) bool {
	return a.Scope.HasAccess(repository.AccessAdmin)
}

func (a *APIKeyOfAdmin) GetUserULID() repository.ULID {
	return a.UserULID
}

func (a *APIKeyOfAdmin) loadDatabases(ctx context.Context) error {
	if a.loaded {
		return nil
	}
	dbs, err := a.Repo.GetDatabases(ctx)
	if err != nil {
		return err
	}
	dbIDs := make([]repository.ULID, len(dbs))
	for i, db := range dbs {
		dbIDs[i] = db.ID
	}
	a.databases = dbIDs
	a.loaded = true
	return nil
}

func (a *APIKeyOfAdmin) GetAllPermissions(ctx context.Context) (map[repository.ULID]repository.AccessGrant, error) {
	if err := a.loadDatabases(ctx); err != nil {
		return nil, err
	}
	perms := make(map[repository.ULID]repository.AccessGrant, len(a.databases))
	for _, dbID := range a.databases {
		perms[dbID] = a.Scope
	}
	return perms, nil
}

// A non-admin user with specific database permissions
type UserPermissions struct {
	UserULID repository.ULID
	Scope    repository.AccessGrant
	Repo     repository.Repository

	loaded      bool
	permissions map[repository.ULID]repository.AccessGrant
}

func (u *UserPermissions) IsGlobalAdmin() bool {
	return false
}

func (u *UserPermissions) loadPermissions(ctx context.Context) {
	if u.loaded {
		return
	}
	perms, err := u.Repo.GetAllUserPermissions(ctx, u.UserULID)
	if err != nil {
		u.permissions = make(map[repository.ULID]repository.AccessGrant)
		u.loaded = true
		return
	}
	permsMap := make(map[repository.ULID]repository.AccessGrant, len(perms))
	for _, p := range perms {
		permsMap[p.DatabaseID] = p.Roles
	}
	u.permissions = permsMap
	u.loaded = true
}

func (u *UserPermissions) CanView(database repository.ULID) bool {
	u.loadPermissions(context.Background())
	const access = repository.AccessView
	if perm, exists := u.permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanCreate(database repository.ULID) bool {
	u.loadPermissions(context.Background())
	const access = repository.AccessCreate
	if perm, exists := u.permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanEdit(database repository.ULID) bool {
	u.loadPermissions(context.Background())
	const access = repository.AccessEdit
	if perm, exists := u.permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanDelete(database repository.ULID) bool {
	u.loadPermissions(context.Background())
	const access = repository.AccessDelete
	if perm, exists := u.permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanAdmin(database repository.ULID) bool {
	u.loadPermissions(context.Background())
	const access = repository.AccessAdmin
	if perm, exists := u.permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) GetUserULID() repository.ULID {
	return u.UserULID
}

func (u *UserPermissions) GetAllPermissions(ctx context.Context) (map[repository.ULID]repository.AccessGrant, error) {
	u.loadPermissions(ctx)
	filtered := make(map[repository.ULID]repository.AccessGrant, len(u.permissions))
	for dbID, perm := range u.permissions {
		filtered[dbID] = perm & u.Scope
	}
	return filtered, nil
}
