package routes

import (
	authRoutes "medina-consultancy-api/http/routes/auth"
	checkoutRoutes "medina-consultancy-api/http/routes/checkout"
	consultancyRoutes "medina-consultancy-api/http/routes/consultancy"
	integrationRoutes "medina-consultancy-api/http/routes/integration"
	subscriptionRoutes "medina-consultancy-api/http/routes/subscription"

	"github.com/gin-gonic/gin"
)

func HandleRequest(r *gin.Engine) {
	generalPath := r.Group("/api/v1")
	{
		authRoutes.RegisterAuthRoutes(generalPath)
	}

	consultancyPath := r.Group("/api/v1/consultancy")
	{
		consultancyRoutes.RegisterConsultancyRoutes(consultancyPath)
	}

	checkoutPath := r.Group("/api/v1/checkout")
	{
		checkoutRoutes.RegisterCheckoutRoutes(checkoutPath)
	}

	subscriptionPath := r.Group("/api/v1/subscription")
	{
		subscriptionRoutes.RegisterSubscriptionRoutes(subscriptionPath)
	}

	integrationPath := r.Group("/api/v1/integration")
	{
		integrationRoutes.RegisterIntegrationRoutes(integrationPath)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
