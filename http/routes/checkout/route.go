package checkout

import (
	"medina-consultancy-api/http/controllers"
	middleware "medina-consultancy-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterCheckoutRoutes(r *gin.RouterGroup) {
	r.Use(middleware.ContentTypeMiddleware())

	r.GET("/packages", controllers.GetCreditPackages)
	r.GET("/packages/:id", controllers.GetCreditPackageByID)

	r.POST("/create", middleware.AuthMiddleware(), controllers.CreateCheckout)
	r.GET("/orders", middleware.AuthMiddleware(), controllers.GetUserOrders)
	r.GET("/orders/:id", middleware.AuthMiddleware(), controllers.GetOrderStatus)
	r.GET("/orders/:id/check", middleware.AuthMiddleware(), controllers.CheckPaymentStatus) // polling endpoint
	r.GET("/credits", middleware.AuthMiddleware(), controllers.GetUserCredits)
}
