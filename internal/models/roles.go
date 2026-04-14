package models

// Стандартные роли, попадающие в claim JWT `roles`.
const (
	RoleUser       = "ROLE_USER"
	RoleEditor     = "ROLE_EDITOR"
	RoleSuperAdmin = "ROLE_SUPER_ADMIN"
	AdminRole      = "ROLE_ADMIN"
)

// DefaultUserRoles роль по умолчанию для новой учётной записи.
func DefaultUserRoles() []string {
	return []string{RoleUser}
}
