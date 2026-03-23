// filepath: /C:/Users/muril/medinaConsultancy-api/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"medina-consultancy-api/database"
	"medina-consultancy-api/http/routes"
	"medina-consultancy-api/pkg/billing"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambdaV2

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	routes.HandleRequest(r)

	for _, route := range r.Routes() {
		log.Printf("Registered route: %s %s", route.Method, route.Path)
	}

	return r
}

func init() {
	database.ConnectWithDatabase()
	r := setupRouter()
	ginLambda = ginadapter.NewV2(r)
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	log.Printf("Received request: %s %s", req.RequestContext.HTTP.Method, req.RawPath)
	return ginLambda.ProxyWithContext(ctx, req)
}

func BillingHandler(ctx context.Context) error {
	log.Println("Starting monthly billing process...")
	return billing.ProcessMonthlyBilling()
}

func main() {
	fmt.Println("Iniciando projeto MedinaConsultancy...")

	mode := os.Getenv("HANDLER_MODE")

	switch mode {
	case "billing":
		lambda.Start(BillingHandler)
	case "local":
		r := setupRouter()
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Local server running on http://localhost:%s", port)
		if err := r.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	case "billing-local":
		log.Println("Running billing locally...")
		if err := billing.ProcessMonthlyBilling(); err != nil {
			log.Fatalf("Billing failed: %v", err)
		}
		log.Println("Billing completed successfully.")
	default:
		lambda.Start(Handler)
	}
}
