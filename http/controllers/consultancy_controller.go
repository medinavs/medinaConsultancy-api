package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CityRequest struct {
	Search string `json:"search"`
	City   string `json:"city"`
}

type PlaceDetails struct {
	Name                 string `json:"name"`
	FormattedAddress     string `json:"formatted_address"`
	Email                string `json:"email"`
	FormattedPhoneNumber string `json:"formatted_phone_number"`
	Geometry             struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
}

func FindLocationsBasedOnAddress(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	var cityReq CityRequest
	if err := c.ShouldBindJSON(&cityReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		log.Printf("API key is missing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API key is missing"})
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
	c.JSON(http.StatusOK, search)
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
				detailsParams.Add("fields", "name,formatted_address,formatted_phone_number,geometry/location")
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
