package services

import (
	"encoding/json"
	"net/http"
	"time"
	logger "github.com/Bparsons0904/goLogger"
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
	Page    int               `json:"page"`
	Pages   int               `json:"pages"`
	Items   int               `json:"items"`
	PerPage int               `json:"per_page"`
	URLs    map[string]string `json:"urls"`
}

type DiscogsNote struct {
	FieldID int    `json:"field_id"`
	Value   string `json:"value"`
}

type DiscogsFolderReleaseItem struct {
	ID               int64         `json:"id"`
	InstanceID       int           `json:"instance_id"`
	FolderID         int           `json:"folder_id"`
	Rating           int           `json:"rating"`
	Notes            []DiscogsNote `json:"notes"`
	BasicInformation struct {
		ID          int64  `json:"id"`
		MasterID    int64  `json:"master_id"`
		MasterURL   string `json:"master_url"`
		ResourceURL string `json:"resource_url"`
		Title       string `json:"title"`
		Year        int    `json:"year"`
		Thumb       string `json:"thumb"`
		CoverImage  string `json:"cover_image"`
		Formats     []struct {
			Name         string   `json:"name"`
			Qty          string   `json:"qty"`
			Text         string   `json:"text"`
			Descriptions []string `json:"descriptions"`
		} `json:"formats"`
		Artists []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			ResourceURL string `json:"resource_url"`
		} `json:"artists"`
		Labels []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			ResourceURL string `json:"resource_url"`
		} `json:"labels"`
		Genres []string `json:"genres"`
		Styles []string `json:"styles"`
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
		return nil, log.ErrMsg("token cannot be empty")
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
		return nil, log.ErrMsg("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, log.Error("Discogs API error", "statusCode", resp.StatusCode)
	}

	var identity DiscogsIdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return nil, log.Err("failed to decode response", err)
	}

	if identity.ID == 0 {
		return nil, log.Error("Invalid response from Discogs API", "identity", identity)
	}

	log.Info(
		"Successfully retrieved Discogs user identity",
		"userID",
		identity.ID,
		"username",
		identity.Username,
	)
	return &identity, nil
}
