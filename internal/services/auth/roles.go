// internal/services/auth/roles.go
package auth

import "mediahub/internal/models"

// getUserRoles retrieves the roles for a given user.
func getUserRoles(user *models.User) []string {
	var roles []string
	if user.CanView {
		roles = append(roles, "CanView")
	}
	if user.CanCreate {
		roles = append(roles, "CanCreate")
	}
	if user.CanEdit {
		roles = append(roles, "CanEdit")
	}
	if user.CanDelete {
		roles = append(roles, "CanDelete")
	}
	if user.IsAdmin {
		roles = append(roles, "IsAdmin")
	}
	return roles
}
