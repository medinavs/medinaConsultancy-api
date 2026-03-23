package middleware

import (
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	"medina-consultancy-api/pkg/jwt"
	"medina-consultancy-api/pkg/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func IntegrationAuthMiddleware() gin.HandlerFunc {
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

		rawToken := bearerToken[1]

		claims, err := jwt.ValidateIntegrationToken(rawToken)
		if err != nil {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Invalid or expired integration token")
			c.Abort()
			return
		}

		var subscription models.Subscription
		if err := database.DB.Where("id = ? AND user_id = ?", claims.SubscriptionID, claims.UserID).First(&subscription).Error; err != nil {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Subscription not found")
			c.Abort()
			return
		}

		if subscription.Status != "active" {
			response.SendGinResponse(c, http.StatusForbidden, nil, nil, "Subscription is not active")
			c.Abort()
			return
		}

		if subscription.IntegrationToken != rawToken {
			response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Token has been revoked")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("subscriptionID", claims.SubscriptionID)
		c.Next()
	}
}
