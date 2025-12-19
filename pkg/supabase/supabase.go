package supabase

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	URL    string
	APIKey string
	Bucket string
}

func NewClient() (*Client, error) {
	url := os.Getenv("SUPABASE_URL")
	apiKey := os.Getenv("SUPABASE_SERVICE_KEY")
	bucket := os.Getenv("SUPABASE_BUCKET")

	if url == "" || apiKey == "" || bucket == "" {
		return nil, fmt.Errorf("SUPABASE_URL, SUPABASE_SERVICE_KEY, and SUPABASE_BUCKET must be set")
	}

	return &Client{
		URL:    url,
		APIKey: apiKey,
		Bucket: bucket,
	}, nil
}

func (c *Client) UploadFile(fileName string, data []byte, contentType string) (string, error) {
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.URL, c.Bucket, fileName)

	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.URL, c.Bucket, fileName)
	return publicURL, nil
}

func (c *Client) GetSignedURL(fileName string) (string, error) {
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.URL, c.Bucket, fileName)
	return publicURL, nil
}

func (c *Client) DownloadFile(fileName string) ([]byte, error) {
	downloadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.URL, c.Bucket, fileName)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
