package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	"medina-consultancy-api/pkg/response"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const CreditsPerSearch = 10

type CityRequest struct {
	Search string `json:"search"`
	City   string `json:"city"`
}

type PlaceDetails struct {
	Name                 string `json:"name"`
	FormattedAddress     string `json:"formatted_address"`
	Email                string `json:"email"`
	FormattedPhoneNumber string `json:"formatted_phone_number"`
	Website              string `json:"website"`
	URL                  string `json:"url"` // google maps url
}

func FindLocationsBasedOnAddress(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "User not found")
		return
	}

	if user.Credits < CreditsPerSearch {
		response.SendGinResponse(c, http.StatusPaymentRequired, gin.H{
			"credits_required":  CreditsPerSearch,
			"credits_available": user.Credits,
		}, nil, "Insufficient credits. Please purchase more credits to continue.")
		return
	}

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

	user.Credits -= CreditsPerSearch
	if err := database.DB.Save(&user).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to debit credits")
		return
	}

	log.Printf("Debited %d credit(s) from user %d. Remaining: %d", CreditsPerSearch, user.ID, user.Credits)

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

			if region != "" {
				cityQuery = fmt.Sprintf("%s %s", cityReq.City, region)
			}

			localPlaces := make(map[string]PlaceDetails)
			fetchPlacesForQuery(searchQuery, cityQuery, apiKey, localPlaces)

			mutex.Lock()
			for placeID, place := range localPlaces {
				if _, exists := uniquePlaces[placeID]; !exists {
					uniquePlaces[placeID] = place
				}
			}
			mutex.Unlock()
		}(region)
	}

	wg.Wait()

	var search []PlaceDetails
	for _, place := range uniquePlaces {
		search = append(search, place)
	}

	log.Printf("Total de resultados únicos encontrados: %d", len(search))

	response.SendGinResponse(c, http.StatusOK, gin.H{
		"results":           search,
		"total_results":     len(search),
		"credits_used":      CreditsPerSearch,
		"credits_remaining": user.Credits,
	}, nil, "")
}

func fetchPlacesForQuery(search string, city string, apiKey string, uniquePlaces map[string]PlaceDetails) {
	nextPageToken := ""
	baseURL := "https://maps.googleapis.com/maps/api/place/textsearch/json"
	maxPages := 5

	for pageCount := 0; pageCount < maxPages; pageCount++ {
		query := fmt.Sprintf("%s in %s", search, city)
		params := url.Values{}
		params.Add("query", query)
		params.Add("key", apiKey)

		if nextPageToken != "" {
			params.Add("pagetoken", nextPageToken)
		}

		fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

		resp, err := http.Get(fullURL)
		if err != nil {
			log.Printf("Failed to fetch data: %v", err)
			break
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			break
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("API returned status %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
			break
		}

		var placesResponse struct {
			Results []struct {
				PlaceID string `json:"place_id"`
			} `json:"results"`
			NextPageToken string `json:"next_page_token"`
			Status        string `json:"status"`
			ErrorMessage  string `json:"error_message"`
		}

		if err := json.Unmarshal(body, &placesResponse); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			break
		}

		if placesResponse.Status != "OK" && placesResponse.Status != "ZERO_RESULTS" {
			log.Printf("API error - Status: %s, Message: %s", placesResponse.Status, placesResponse.ErrorMessage)
			break
		}

		log.Printf("Found %d results for query: %s", len(placesResponse.Results), query)

		var detailsWg sync.WaitGroup
		var detailsMutex sync.Mutex

		for _, result := range placesResponse.Results {
			if _, exists := uniquePlaces[result.PlaceID]; exists {
				continue
			}

			detailsWg.Add(1)
			go func(placeID string) {
				defer detailsWg.Done()

				detailsParams := url.Values{}
				detailsParams.Add("place_id", placeID)
				detailsParams.Add("fields", "name,formatted_address,formatted_phone_number,website,url")
				detailsParams.Add("key", apiKey)

				detailsURL := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/details/json?%s", detailsParams.Encode())

				detailsResp, err := http.Get(detailsURL)
				if err != nil {
					return
				}
				defer detailsResp.Body.Close()

				detailsBody, err := io.ReadAll(detailsResp.Body)
				if err != nil {
					return
				}

				if detailsResp.StatusCode != http.StatusOK {
					return
				}

				var detailsResponse struct {
					Result PlaceDetails `json:"result"`
					Status string       `json:"status"`
				}

				if err := json.Unmarshal(detailsBody, &detailsResponse); err != nil {
					return
				}

				if detailsResponse.Status == "OK" {
					detailsMutex.Lock()
					uniquePlaces[placeID] = detailsResponse.Result
					detailsMutex.Unlock()
				}
			}(result.PlaceID)
		}

		detailsWg.Wait()

		if placesResponse.NextPageToken == "" {
			break
		}

		nextPageToken = placesResponse.NextPageToken
		time.Sleep(2 * time.Second)
	}
}
