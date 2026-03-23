package integration

import (
	"medina-consultancy-api/http/controllers"
	middleware "medina-consultancy-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterIntegrationRoutes(r *gin.RouterGroup) {
	r.Use(middleware.ContentTypeMiddleware())
	r.Use(middleware.IntegrationAuthMiddleware())

	r.POST("/search", controllers.IntegrationSearch)
	r.GET("/usage", controllers.GetUsage)
}
