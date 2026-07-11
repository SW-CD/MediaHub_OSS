package repository

// AccessGrant represents bitmasked access rights.
type AccessGrant uint8

const (
	AccessView   AccessGrant = 1 << iota // 1 (0001)
	AccessCreate                         // 2 (0010)
	AccessEdit                           // 4 (0100)
	AccessDelete                         // 8 (1000)
	AccessAdmin                          // 16 (0001 0000)
)

func NewAccessGrant(view, create, edit, delete, admin bool) AccessGrant {
	var grant AccessGrant
	if view {
		grant |= AccessView
	}
	if create {
		grant |= AccessCreate
	}
	if edit {
		grant |= AccessEdit
	}
	if delete {
		grant |= AccessDelete
	}
	if admin {
		grant |= AccessAdmin
	}
	return grant
}

func (ag AccessGrant) HasAccess(required AccessGrant) bool {
	return ag&required == required
}
