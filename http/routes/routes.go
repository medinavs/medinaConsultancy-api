package routes

import (
	consultancyRoutes "medina-consultancy-api/http/routes/consultancy"

	"github.com/gin-gonic/gin"
)

func HandleRequest(r *gin.Engine) {

	consultancyPath := r.Group("/consultancy")
	{
		consultancyRoutes.RegisterConsultancyRoutes(consultancyPath)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
