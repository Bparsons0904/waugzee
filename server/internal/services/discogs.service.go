package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"waugzee/internal/logger"
)

type DiscogsService struct {
	client  *http.Client
	baseURL string
	log     logger.Logger
}

type DiscogsIdentityResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type DiscogsFolderItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
	ResourceURL string `json:"resource_url"`
}

type DiscogsFoldersData struct {
	Folders []DiscogsFolderItem `json:"folders"`
}

type DiscogsFoldersResponse struct {
	Data       DiscogsFoldersData `json:"data"`
	Status     int                `json:"status"`
	StatusText string             `json:"statusText"`
}

type DiscogsPagination struct {
	Page    int                    `json:"page"`
	Pages   int                    `json:"pages"`
	Items   int                    `json:"items"`
	PerPage int                    `json:"per_page"`
	URLs    map[string]string      `json:"urls"`
}

type DiscogsFolderReleaseItem struct {
	ID         int64  `json:"id"`
	InstanceID int    `json:"instance_id"`
	FolderID   int    `json:"folder_id"`
	Rating     int    `json:"rating"`
	Notes      string `json:"notes"`
	BasicInformation struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Year    int    `json:"year"`
		Formats []struct {
			Name         string   `json:"name"`
			Qty          string   `json:"qty"`
			Descriptions []string `json:"descriptions"`
		} `json:"formats"`
		Artists []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"artists"`
		Labels []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
		Genres []string `json:"genres"`
		Styles []string `json:"styles"`
		Thumb  string   `json:"thumb"`
	} `json:"basic_information"`
	DateAdded string `json:"date_added"`
}

type DiscogsFolderReleasesData struct {
	Releases   []DiscogsFolderReleaseItem `json:"releases"`
	Pagination DiscogsPagination          `json:"pagination"`
}

type DiscogsFolderReleasesResponse struct {
	Data       DiscogsFolderReleasesData `json:"data"`
	Status     int                       `json:"status"`
	StatusText string                    `json:"statusText"`
}

func NewDiscogsService() *DiscogsService {
	log := logger.New("DiscogsService")
	return &DiscogsService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.discogs.com",
		log:     log,
	}
}

func (d *DiscogsService) GetUserIdentity(token string) (*DiscogsIdentityResponse, error) {
	log := d.log.Function("GetUserIdentity")

	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	req, err := http.NewRequest("GET", d.baseURL+"/oauth/identity", nil)
	if err != nil {
		return nil, log.Err("failed to create request", err)
	}

	req.Header.Set("Authorization", "Discogs token="+token)
	req.Header.Set("User-Agent", "Waugzee/1.0")

	log.Debug("Making request to Discogs identity endpoint")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, log.Err("failed to make request", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		log.Warn("Invalid Discogs token provided")
		return nil, fmt.Errorf("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		_ = log.Error("Discogs API error", "statusCode", resp.StatusCode)
		return nil, fmt.Errorf("discogs API error: %d", resp.StatusCode)
	}

	var identity DiscogsIdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return nil, log.Err("failed to decode response", err)
	}

	if identity.ID == 0 {
		_ = log.Error("Invalid response from Discogs API", "identity", identity)
		return nil, fmt.Errorf("invalid response from Discogs API")
	}

	log.Info("Successfully retrieved Discogs user identity", "userID", identity.ID, "username", identity.Username)
	return &identity, nil
}
