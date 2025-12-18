package middleware

import (
	"medina-consultancy-api/pkg/jwt"
	"medina-consultancy-api/pkg/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Authorization header is required")
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Invalid authorization header format")
			c.Abort()
			return
		}

		claims, err := jwt.ValidateToken(bearerToken[1])
		if err != nil {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
