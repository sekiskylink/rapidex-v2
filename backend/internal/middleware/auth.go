package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/auth"
	"basepro/backend/internal/rbac"
	"github.com/gin-gonic/gin"
)

type PermissionOption func(*permissionRequirement)

type permissionRequirement struct {
	moduleScope *string
}

func WithModule(scope string) PermissionOption {
	trimmed := strings.TrimSpace(scope)
	return func(req *permissionRequirement) {
		if trimmed == "" {
			return
		}
		req.moduleScope = &trimmed
	}
}

func JWTAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			apperror.Write(c, apperror.Unauthorized("Missing authorization token"))
			c.Abort()
			return
		}

		claims, err := jwtManager.ParseAccessToken(parts[1])
		if err != nil {
			if err == auth.ErrTokenExpired {
				apperror.Write(c, apperror.Expired("Access token expired"))
			} else {
				apperror.Write(c, apperror.Unauthorized("Invalid access token"))
			}
			c.Abort()
			return
		}

		principal := auth.Principal{
			Type:             "user",
			ID:               strconv.FormatInt(claims.UserID, 10),
			UserID:           claims.UserID,
			Username:         claims.Username,
			Permissions:      []string{},
			Roles:            []string{},
			PermissionGrants: []auth.PermissionGrant{},
		}

		c.Set(auth.ClaimsContextKey, claims)
		c.Set(auth.PrincipalContextKey, principal)
		c.Next()
	}
}

func APITokenAuth(service *auth.Service, headerName string, allowBearer bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader(headerName))
		if token == "" && allowBearer {
			parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				candidate := strings.TrimSpace(parts[1])
				if strings.Count(candidate, ".") != 2 {
					token = candidate
				}
			}
		}

		if token == "" {
			c.Next()
			return
		}

		principal, err := service.AuthenticateAPIToken(c.Request.Context(), token, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			apperror.Write(c, err)
			c.Abort()
			return
		}

		c.Set(auth.PrincipalContextKey, principal)
		c.Next()
	}
}

func ResolveJWTPrincipal(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(auth.PrincipalContextKey); exists {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.Next()
			return
		}

		candidate := strings.TrimSpace(parts[1])
		if strings.Count(candidate, ".") != 2 {
			c.Next()
			return
		}

		claims, err := jwtManager.ParseAccessToken(candidate)
		if err != nil {
			if err == auth.ErrTokenExpired {
				apperror.Write(c, apperror.Expired("Access token expired"))
			} else {
				apperror.Write(c, apperror.Unauthorized("Invalid access token"))
			}
			c.Abort()
			return
		}

		principal := auth.Principal{
			Type:             "user",
			ID:               strconv.FormatInt(claims.UserID, 10),
			UserID:           claims.UserID,
			Username:         claims.Username,
			Permissions:      []string{},
			Roles:            []string{},
			PermissionGrants: []auth.PermissionGrant{},
		}
		c.Set(auth.ClaimsContextKey, claims)
		c.Set(auth.PrincipalContextKey, principal)
		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := PrincipalFromContext(c); !ok {
			apperror.Write(c, apperror.Unauthorized("Unauthorized"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireJWTUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFromContext(c)
		if !ok {
			apperror.Write(c, apperror.Unauthorized("Unauthorized"))
			c.Abort()
			return
		}
		if principal.Type != "user" {
			apperror.Write(c, apperror.Forbidden("Forbidden"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequirePermission(rbacService *rbac.Service, permission string, opts ...PermissionOption) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFromContext(c)
		if !ok {
			apperror.Write(c, apperror.Unauthorized("Unauthorized"))
			c.Abort()
			return
		}

		req := permissionRequirement{}
		for _, opt := range opts {
			opt(&req)
		}

		switch principal.Type {
		case "api_token":
			if hasPermissionGrant(principal.PermissionGrants, permission, req.moduleScope) {
				c.Next()
				return
			}
			apperror.Write(c, apperror.Forbidden("Forbidden"))
			c.Abort()
			return
		case "user":
			if rbacService == nil {
				apperror.Write(c, fmt.Errorf("missing rbac service"))
				c.Abort()
				return
			}

			hasPerm, err := rbacService.HasPermission(c.Request.Context(), principal.UserID, permission, req.moduleScope)
			if err != nil {
				apperror.Write(c, err)
				c.Abort()
				return
			}
			if !hasPerm {
				apperror.Write(c, apperror.Forbidden("Forbidden"))
				c.Abort()
				return
			}

			if err := enrichUserPrincipal(c, rbacService, principal); err != nil {
				apperror.Write(c, err)
				c.Abort()
				return
			}

			c.Next()
			return
		default:
			apperror.Write(c, apperror.Unauthorized("Unauthorized"))
			c.Abort()
			return
		}
	}
}

func ClaimsFromContext(c *gin.Context) (auth.Claims, bool) {
	value, ok := c.Get(auth.ClaimsContextKey)
	if !ok {
		return auth.Claims{}, false
	}
	claims, ok := value.(auth.Claims)
	if !ok {
		return auth.Claims{}, false
	}
	return claims, true
}

func PrincipalFromContext(c *gin.Context) (auth.Principal, bool) {
	value, ok := c.Get(auth.PrincipalContextKey)
	if !ok {
		return auth.Principal{}, false
	}
	principal, ok := value.(auth.Principal)
	return principal, ok
}

func hasPermissionGrant(grants []auth.PermissionGrant, permission string, moduleScope *string) bool {
	for _, candidate := range grants {
		if candidate.Permission != permission {
			continue
		}
		if moduleScope == nil {
			return true
		}
		if candidate.ModuleScope != nil && strings.EqualFold(strings.TrimSpace(*candidate.ModuleScope), strings.TrimSpace(*moduleScope)) {
			return true
		}
	}
	return false
}

func enrichUserPrincipal(c *gin.Context, service *rbac.Service, principal auth.Principal) error {
	if principal.Type != "user" {
		return nil
	}
	roles, err := service.RoleNamesForUser(c.Request.Context(), principal.UserID)
	if err != nil {
		return err
	}
	permissions, err := service.GetUserPermissions(c.Request.Context(), principal.UserID)
	if err != nil {
		return err
	}
	principal.Roles = roles
	principal.Permissions = make([]string, 0, len(permissions))
	principal.PermissionGrants = make([]auth.PermissionGrant, 0, len(permissions))
	for _, permission := range permissions {
		principal.Permissions = append(principal.Permissions, permission.Name)
		principal.PermissionGrants = append(principal.PermissionGrants, auth.PermissionGrant{
			Permission:  permission.Name,
			ModuleScope: permission.ModuleScope,
		})
	}
	c.Set(auth.PrincipalContextKey, principal)
	return nil
}

// Ensure middleware helpers can be wrapped with context cancellation in tests.
func contextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
