package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/verno/gotradex/pkg/auth"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	UserIDKey           = "userID"
	EmailKey            = "email"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
				"code":  "INVALID_AUTH_FORMAT",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := m.jwtManager.ValidateToken(tokenString)
		if err != nil {
			var code string
			if err == auth.ErrExpiredToken {
				code = "TOKEN_EXPIRED"
			} else {
				code = "INVALID_TOKEN"
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
				"code":  code,
			})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Next()
	}
}
