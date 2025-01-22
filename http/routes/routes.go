package routes

import (
	"log"

	consultancyRoutes "medina-consultancy-api/http/routes/consultancy"

	"github.com/gin-gonic/gin"
)

func HandleRequest(r *gin.Engine) {

	consultancyPath := r.Group("/consultancy")
	{
		consultancyRoutes.RegisterConsultancyRoutes(consultancyPath)
	}

	log.Fatal(r.Run(":8080"))
}
