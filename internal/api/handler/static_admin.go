package handler

const (
	staticAdminName       = "Local Admin"
	staticAdminEmail      = "admin@local"
	staticAdminGivenName  = "Local"
	staticAdminFamilyName = "Admin"
)

// staticAdminGroups are the hardcoded IDP groups injected when OIDC is disabled.
// Projects whose roleBindings reference any of these groups will be accessible.
var staticAdminGroups = []string{"local-admin"}
