package sqlite

import "strings"

func parseRolesString(rolesStr string) (canView, canCreate, canEdit, canDelete bool) {
	roles := strings.Split(rolesStr, ",")
	for _, r := range roles {
		switch strings.TrimSpace(r) {
		case "CanView":
			canView = true
		case "CanCreate":
			canCreate = true
		case "CanEdit":
			canEdit = true
		case "CanDelete":
			canDelete = true
		}
	}
	return
}

func buildRolesString(canView, canCreate, canEdit, canDelete bool) string {
	var roles []string
	if canView {
		roles = append(roles, "CanView")
	}
	if canCreate {
		roles = append(roles, "CanCreate")
	}
	if canEdit {
		roles = append(roles, "CanEdit")
	}
	if canDelete {
		roles = append(roles, "CanDelete")
	}
	return strings.Join(roles, ",")
}
