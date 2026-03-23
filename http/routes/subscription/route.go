package subscription

import (
	"medina-consultancy-api/http/controllers"
	middleware "medina-consultancy-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterSubscriptionRoutes(r *gin.RouterGroup) {
	r.Use(middleware.ContentTypeMiddleware())
	r.Use(middleware.AuthMiddleware())

	r.POST("/create", controllers.CreateSubscription)
	r.GET("/status", controllers.GetSubscriptionStatus)
	r.POST("/cancel", controllers.CancelSubscription)
	r.GET("/invoices", controllers.GetInvoices)
	r.POST("/regenerate-token", controllers.RegenerateToken)
}
