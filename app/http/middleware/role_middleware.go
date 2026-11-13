package middleware

import (
	"slices"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

type RoleMiddleware struct {
	roles []string
}

func NewRoleMiddleware(roles ...string) http.Middleware {
	return &RoleMiddleware{roles: roles}
}

func (m *RoleMiddleware) Signature() string {
	return "role:" + strings.Join(m.roles, "|")
}

// Handle authorizes the authenticated user. Access is granted when the user's
// role is explicitly requested by the route, or when the Casbin policy allows
// the role to perform the current method on the current path.
//
// The role is read from the database instead of the JWT claim so a role change
// made in Panel Developer applies immediately, without waiting for the token to
// expire. A deleted account therefore also loses its admin access at once.
func (m *RoleMiddleware) Handle(ctx http.Context) {
	role := currentDatabaseRole(ctx)
	if role == "" {
		denyAccess(ctx, "Role not found")
		return
	}

	if len(m.roles) > 0 && slices.Contains(m.roles, role) {
		ctx.Request().Next()
		return
	}

	if allowedByCasbin(role, ctx.Request().Path(), ctx.Request().Method()) {
		ctx.Request().Next()
		return
	}

	denyAccess(ctx, "Insufficient permissions")
}

func denyAccess(ctx http.Context, message string) {
	if wantsHTML(ctx) {
		ctx.Response().Redirect(http.StatusFound, "/dasbor?error=admin_only").Abort()
		return
	}

	ctx.Response().Json(http.StatusForbidden, http.Json{"error": message}).Abort()
}

// currentDatabaseRole resolves the caller's current role. AuthMiddleware stores
// the JWT "sub" claim as a number (float64 after JSON decoding), so it is
// normalised here before querying users.
func currentDatabaseRole(ctx http.Context) string {
	id := contextUserIDValue(ctx.Value(contextUserID))
	if id == 0 {
		return ""
	}

	var user models.User
	if err := facades.Orm().Query().
		Select("id", "role").
		Where("id = ?", id).
		FirstOrFail(&user); err != nil {
		return ""
	}

	return user.Role
}

func contextUserIDValue(value any) uint {
	switch v := value.(type) {
	case uint:
		return v
	case int:
		return uint(v)
	case float64:
		return uint(v)
	case string:
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			return uint(parsed)
		}
	}

	return 0
}

func allowedByCasbin(role, path, method string) bool {
	enforcer, err := services.Enforcer()
	if err != nil {
		return false
	}

	allowed, err := enforcer.Enforce(role, path, method)

	return err == nil && allowed
}
