package controllers

import (
	"fmt"
	"log"
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	"medina-consultancy-api/pkg/response"
	"medina-consultancy-api/pkg/supabase"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func IntegrationSearch(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	subscriptionID, _ := c.Get("subscriptionID")

	var cityReq CityRequest
	if err := c.ShouldBindJSON(&cityReq); err != nil {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, "Invalid request body")
		return
	}

	if cityReq.Search == "" || cityReq.City == "" {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, "Search and city fields are required")
		return
	}

	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		log.Printf("API key is missing")
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "API key is missing")
		return
	}

	uniquePlaces := make(map[string]PlaceDetails)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	regions := []string{"", "centro", "norte", "sul", "leste", "oeste"}

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			searchQuery := cityReq.Search
			cityQuery := cityReq.City

			if len(cityReq.Keywords) > 0 {
				searchQuery = fmt.Sprintf("%s %s", searchQuery, strings.Join(cityReq.Keywords, " "))
			}

			if region != "" {
				cityQuery = fmt.Sprintf("%s %s", cityReq.City, region)
			}

			fetchPlacesForQuery(searchQuery, cityQuery, apiKey, cityReq.PlaceType, uniquePlaces, &mutex)
		}(region)
	}

	wg.Wait()

	var search []PlaceDetails
	for _, place := range uniquePlaces {
		search = append(search, place)
	}

	log.Printf("Integration search - Total unique results: %d", len(search))

	searchID := uuid.New().String()
	fileName := fmt.Sprintf("searches/%s.csv", searchID)

	csvData, err := generateCSV(search)
	if err != nil {
		log.Printf("Failed to generate CSV: %v", err)
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to generate CSV")
		return
	}

	supabaseClient, err := supabase.NewClient()
	if err != nil {
		log.Printf("Failed to create Supabase client: %v", err)
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to initialize storage")
		return
	}

	bucketURL, err := supabaseClient.UploadFile(fileName, csvData, "text/csv")
	if err != nil {
		log.Printf("Failed to upload CSV to Supabase: %v", err)
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to save search results")
		return
	}

	billingMonth := time.Now().Format("2006-01")
	integrationQuery := models.IntegrationQuery{
		SubscriptionID: subscriptionID.(uint),
		UserID:         userID.(uint),
		SearchID:       searchID,
		Query:          cityReq.Search,
		City:           cityReq.City,
		Results:        len(search),
		BucketURL:      bucketURL,
		BillingMonth:   billingMonth,
	}

	if err := database.DB.Create(&integrationQuery).Error; err != nil {
		log.Printf("Failed to save integration query record: %v", err)
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to save query record")
		return
	}

	// get current month query count for billing info
	var queryCount int64
	database.DB.Model(&models.IntegrationQuery{}).
		Where("subscription_id = ? AND billing_month = ?", subscriptionID, billingMonth).
		Count(&queryCount)

	unitPrice := CalculateUnitPrice(int(queryCount))

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"search_id":     searchID,
		"results":       search,
		"total_results": len(search),
		"download_url":  bucketURL,
		"billing": gin.H{
			"queries_this_month": queryCount,
			"current_tier_price": fmt.Sprintf("%.2f", unitPrice),
		},
	}, nil, "")
}

func GetUsage(c *gin.Context) {
	subscriptionID, exists := c.Get("subscriptionID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Subscription not found")
		return
	}

	billingMonth := time.Now().Format("2006-01")
	var queryCount int64
	database.DB.Model(&models.IntegrationQuery{}).
		Where("subscription_id = ? AND billing_month = ?", subscriptionID, billingMonth).
		Count(&queryCount)

	unitPrice := CalculateUnitPrice(int(queryCount))
	estimatedTotal := unitPrice * float64(queryCount)

	tier := "0-99"
	if queryCount >= 200 {
		tier = "200+"
	} else if queryCount >= 100 {
		tier = "100-199"
	}

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"billing_month":      billingMonth,
		"queries_this_month": queryCount,
		"current_tier":       tier,
		"unit_price":         fmt.Sprintf("%.2f", unitPrice),
		"estimated_total":    fmt.Sprintf("%.2f", estimatedTotal),
	}, nil, "")
}
