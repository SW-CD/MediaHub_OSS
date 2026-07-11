package utils

import "mediahub_oss/internal/repository"

type PermissionHolder interface {
	IsGlobalAdmin() bool
	CanView(database repository.ULID) bool
	CanCreate(database repository.ULID) bool
	CanEdit(database repository.ULID) bool
	CanDelete(database repository.ULID) bool

	GetUserULID() repository.ULID
	GetAllPermissions() map[repository.ULID]repository.AccessGrant
}

// There are three types of permission holders:
// **GlobalAdmin**: has full access to all databases and actions.
// **APIKeyOfAdmin**: has access limited only by the scope of the API key
// **UserPermissions**: has access limited by the specific database permissions and a potential API key scope

type GlobalAdmin struct {
	UserULID repository.ULID
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

func (g *GlobalAdmin) GetUserULID() repository.ULID {
	return g.UserULID
}

func (g *GlobalAdmin) GetAllPermissions() map[repository.ULID]repository.AccessGrant {
	return map[repository.ULID]repository.AccessGrant{} // Global admin has implicit access to all databases
}

// An API key without admin scope of an admin user
type APIKeyOfAdmin struct {
	UserULID repository.ULID
	Scope    repository.AccessGrant
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

func (a *APIKeyOfAdmin) GetUserULID() repository.ULID {
	return a.UserULID
}

func (a *APIKeyOfAdmin) GetAllPermissions() map[repository.ULID]repository.AccessGrant {
	// TODO query all databases and return a map with the same scope for each database
	return TODO
}

// A non-admin user with specific database permissions
type UserPermissions struct {
	UserULID    repository.ULID
	Scope       repository.AccessGrant
	Permissions map[repository.ULID]repository.AccessGrant
}

func (u *UserPermissions) IsGlobalAdmin() bool {
	return false
}

func (u *UserPermissions) CanView(database repository.ULID) bool {
	const access = repository.AccessView
	if perm, exists := u.Permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanCreate(database repository.ULID) bool {
	const access = repository.AccessCreate
	if perm, exists := u.Permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanEdit(database repository.ULID) bool {
	const access = repository.AccessEdit
	if perm, exists := u.Permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) CanDelete(database repository.ULID) bool {
	const access = repository.AccessDelete
	if perm, exists := u.Permissions[database]; exists {
		return (u.Scope & perm & access) == access
	}
	return false
}

func (u *UserPermissions) GetUserULID() repository.ULID {
	return u.UserULID
}

func (u *UserPermissions) GetAllPermissions() map[repository.ULID]repository.AccessGrant {
	// TODO: Filter permissions by the API key scope
	return TODO
}
