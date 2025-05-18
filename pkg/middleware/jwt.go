package middleware

import (
	"strings"

	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/opd/internal/user/usersrv"
	"github.com/Abraxas-365/opd/pkg/jwt"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// JWTAuthMiddleware handles JWT authentication for the application
type JWTAuthMiddleware struct {
	jwtService *jwt.JWTService
	userSrv    *usersrv.Service
}

// NewJWTAuthMiddleware creates a new JWT auth middleware
func NewJWTAuthMiddleware(jwtService *jwt.JWTService, userSrv *usersrv.Service) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		jwtService: jwtService,
		userSrv:    userSrv,
	}
}

// ExtractBearerToken extracts the token from the Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.ErrUnauthorized("Missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.ErrUnauthorized("Invalid token format")
	}

	return parts[1], nil
}

// AuthMiddleware attaches user data to the request context if a valid token is provided
func (m *JWTAuthMiddleware) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString, err := ExtractBearerToken(c.Get("Authorization"))
		if err != nil {
			// No token, continue without authenticated user
			return c.Next()
		}

		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			// Invalid token, continue without authenticated user
			return c.Next()
		}

		// Set user info in context
		c.Locals("userID", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("isAdmin", claims.IsAdmin)
		c.Locals("role", claims.Role)
		c.Locals("subscriptionType", claims.SubscriptionType)
		c.Locals("authenticated", true)

		return c.Next()
	}
}

// RequireAuth ensures the request is authenticated
func (m *JWTAuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authenticated := c.Locals("authenticated")
		if authenticated == nil || authenticated.(bool) == false {
			return errors.ErrUnauthorized("Authentication required")
		}
		return c.Next()
	}
}

// RequireAdmin ensures the user has admin privileges
func (m *JWTAuthMiddleware) RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authenticated := c.Locals("authenticated")
		if authenticated == nil || authenticated.(bool) == false {
			return errors.ErrUnauthorized("Authentication required")
		}

		isAdmin := c.Locals("isAdmin")
		role := c.Locals("role")

		if (isAdmin == nil || isAdmin.(bool) == false) &&
			(role == nil || role.(user.Role) != user.RoleAdmin) {
			return errors.ErrForbidden("Admin access required")
		}

		return c.Next()
	}
}

// RequireRole ensures the user has a specific role
func (m *JWTAuthMiddleware) RequireRole(role user.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authenticated := c.Locals("authenticated")
		if authenticated == nil || authenticated.(bool) == false {
			return errors.ErrUnauthorized("Authentication required")
		}

		userRole := c.Locals("role")
		if userRole == nil || userRole.(user.Role) != role {
			return errors.ErrForbidden("Insufficient permissions")
		}

		return c.Next()
	}
}

// RequireMinimumSubscription ensures the user has at least the specified subscription level
func (m *JWTAuthMiddleware) RequireMinimumSubscription(level user.SubscriptionType) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authenticated := c.Locals("authenticated")
		if authenticated == nil || authenticated.(bool) == false {
			return errors.ErrUnauthorized("Authentication required")
		}

		userRole := c.Locals("role")
		if userRole == nil || userRole.(user.Role) != user.RoleLinkoUser {
			return errors.ErrForbidden("Linko user access required")
		}

		subscription := c.Locals("subscriptionType")
		if subscription == nil {
			return errors.ErrForbidden("No subscription found")
		}

		userSub := subscription.(user.SubscriptionType)
		switch level {
		case user.FreeTier:
			// All subscription types have access to free tier
			return c.Next()
		case user.LinkoPlus:
			if userSub == user.LinkoPlus || userSub == user.LinkoVIP {
				return c.Next()
			}
		case user.LinkoVIP:
			if userSub == user.LinkoVIP {
				return c.Next()
			}
		}

		return errors.ErrForbidden("Subscription level not sufficient")
	}
}

// GetAuthenticatedUser retrieves the user from the request context
func (m *JWTAuthMiddleware) GetAuthenticatedUser(c *fiber.Ctx) (*user.User, error) {
	userID := c.Locals("userID")
	if userID == nil {
		return nil, errors.ErrUnauthorized("No authenticated user")
	}

	return m.userSrv.GetUser(c.Context(), userID.(string))
}
