package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type CityRequest struct {
	Search string `json:"search"`
	City   string `json:"city"`
}

// type LocationRequest struct {
// 	Latitude  float64 `json:"latitude"`
// 	Longitude float64 `json:"longitude"`
// }

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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file.")
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

	var search []PlaceDetails
	nextPageToken := ""
	baseURL := "https://maps.googleapis.com/maps/api/place/textsearch/json"

	for {
		url := fmt.Sprintf("%s?query=%s+in+%s&key=%s", baseURL, cityReq.Search, cityReq.City, apiKey)
		if nextPageToken != "" {
			url = fmt.Sprintf("%s&pagetoken=%s", url, nextPageToken)
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to fetch data from Google Places API: %v", err)
			break
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			break
		}

		var placesResponse struct {
			Results []struct {
				PlaceID string `json:"place_id"`
			} `json:"results"`
			NextPageToken string `json:"next_page_token"`
		}

		if err := json.Unmarshal(body, &placesResponse); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			break
		}

		for _, result := range placesResponse.Results {
			detailsURL := fmt.Sprintf(
				"https://maps.googleapis.com/maps/api/place/details/json?place_id=%s&fields=name,formatted_address,formatted_phone_number,geometry/location&key=%s",
				result.PlaceID, apiKey,
			)

			detailsResp, err := http.Get(detailsURL)
			if err != nil {
				log.Printf("Failed to fetch details for place_id %s: %v", result.PlaceID, err)
				continue
			}
			defer detailsResp.Body.Close()

			if detailsResp.StatusCode != http.StatusOK {
				log.Printf("Details request failed: %d - %s", detailsResp.StatusCode, detailsResp.Status)
				continue
			}

			detailsBody, err := io.ReadAll(detailsResp.Body)
			if err != nil {
				log.Printf("Failed to read details response: %v", err)
				continue
			}

			var detailsResponse struct {
				Result PlaceDetails `json:"result"`
			}

			if err := json.Unmarshal(detailsBody, &detailsResponse); err != nil {
				log.Printf("Failed to parse details JSON: %v", err)
				continue
			}

			search = append(search, detailsResponse.Result)
		}

		if placesResponse.NextPageToken == "" {
			break
		}

		nextPageToken = placesResponse.NextPageToken
		time.Sleep(3 * time.Second)
	}

	c.JSON(http.StatusOK, search)
}
