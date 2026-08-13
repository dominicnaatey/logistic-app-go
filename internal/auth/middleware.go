package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"logistic-app-go/pkg/auth"
	"logistic-app-go/pkg/response"
)

// Context keys used to store values on the Gin context.
// Using typed constants avoids collisions with other middleware.
const (
	// ContextKeyClaims is the key under which the JWT claims are stored.
	ContextKeyClaims = "claims"

	// ContextKeyUserID is a convenience key for the user's UUID string.
	ContextKeyUserID = "userID"

	// ContextKeyRole is a convenience key for the user's role string.
	ContextKeyRole = "role"
)

// JWTMiddleware returns a Gin handler that validates the Bearer token
// in the Authorization header and attaches the parsed claims to the context.
//
// Protected routes must be registered under a group that uses this middleware:
//
//	auth := router.Group("/api/v1")
//	auth.Use(middleware.JWTMiddleware(jwtManager))
//	auth.GET("/me", handlers.GetMe)
func JWTMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from "Authorization: Bearer <token>" header
		tokenString, err := extractBearerToken(c)
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}

		// Verify signature and expiry
		claims, err := jwtManager.Verify(tokenString)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// Attach claims and convenience values to context for downstream handlers
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyUserID, claims.UserID.String())
		c.Set(ContextKeyRole, claims.Role)

		c.Next()
	}
}

// RequireRoles returns a Gin handler that allows only users whose role
// matches one of the provided allowedRoles.
// Must be used AFTER JWTMiddleware (it reads the role from context).
//
// Example — admin only:
//
//	adminRoutes.Use(middleware.RequireRoles(user.RoleAdmin))
//
// Example — drivers and owner-operators:
//
//	driverRoutes.Use(middleware.RequireRoles(user.RoleDriver, user.RoleOwnerOperator))
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	// Build a set for O(1) lookup
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			// JWTMiddleware wasn't applied before this — programming error
			response.Error(c, http.StatusInternalServerError, "auth middleware not applied")
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			response.Forbidden(c, "invalid role type in context")
			c.Abort()
			return
		}

		if _, permitted := allowed[roleStr]; !permitted {
			response.Forbidden(c, "you do not have permission to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetClaims is a helper that extracts the typed Claims from the Gin context.
// Returns nil if the middleware was not applied or claims are missing.
func GetClaims(c *gin.Context) *auth.Claims {
	val, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil
	}
	claims, _ := val.(*auth.Claims)
	return claims
}

// extractBearerToken reads and validates the Authorization header format.
// Returns the raw token string or a descriptive error.
func extractBearerToken(c *gin.Context) (string, error) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization header format must be: Bearer <token>")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	return token, nil
}
