package auth

import (
	"medina-consultancy-api/http/controllers"
	middleware "medina-consultancy-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.RouterGroup) {
	r.Use(middleware.ContentTypeMiddleware())

	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

	r.GET("/profile", middleware.AuthMiddleware(), controllers.GetProfile)
}
