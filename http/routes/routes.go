package routes

import (
	authRoutes "medina-consultancy-api/http/routes/auth"
	consultancyRoutes "medina-consultancy-api/http/routes/consultancy"

	"github.com/gin-gonic/gin"
)

func HandleRequest(r *gin.Engine) {
	generalPath := r.Group("/api/v1")
	{
		authRoutes.RegisterAuthRoutes(generalPath)
	}

	consultancyPath := r.Group("/consultancy")
	{
		consultancyRoutes.RegisterConsultancyRoutes(consultancyPath)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
