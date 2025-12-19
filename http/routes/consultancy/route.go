package marketsRoutes

import (
	"medina-consultancy-api/http/controllers"
	middleware "medina-consultancy-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterConsultancyRoutes(r *gin.RouterGroup) {
	r.Use(middleware.ContentTypeMiddleware())

	r.GET("/place-types", controllers.GetPlaceTypes)
	r.GET("/keywords", controllers.GetKeywordSuggestions)

	r.POST("/search", middleware.AuthMiddleware(), controllers.FindLocationsBasedOnAddress)
	r.GET("/search/:searchId/csv", middleware.AuthMiddleware(), controllers.DownloadSearchCSV)
	r.GET("/searches", middleware.AuthMiddleware(), controllers.GetUserSearches)
}
