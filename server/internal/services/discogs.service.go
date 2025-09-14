package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscogsService struct {
	client  *http.Client
	baseURL string
}

type DiscogsIdentityResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func NewDiscogsService() *DiscogsService {
	return &DiscogsService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.discogs.com",
	}
}

func (d *DiscogsService) ValidateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	req, err := http.NewRequest("GET", d.baseURL+"/oauth/identity", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Discogs token="+token)
	req.Header.Set("User-Agent", "Waugzee/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log warning but don't fail the operation
			// We can't use logger here since this is a simple service
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discogs API error: %d", resp.StatusCode)
	}

	var identity DiscogsIdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if identity.ID == 0 {
		return fmt.Errorf("invalid response from Discogs API")
	}

	return nil
}