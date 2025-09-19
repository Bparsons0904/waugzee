package handlers

import (
	"context"
	"strconv"
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/services"
	"waugzee/internal/types"

	"github.com/gofiber/fiber/v2"
)

type DiscogsHandler struct {
	Handler
	app                  app.App
	orchestrationService services.DiscogsOrchestrationService
	rateLimitService     services.DiscogsRateLimitService
}

type InitiateSyncRequest struct {
	SyncType  string `json:"syncType" validate:"required,oneof=collection wantlist"`
	FullSync  bool   `json:"fullSync"`
	PageLimit *int   `json:"pageLimit,omitempty"`
}

type InitiateSyncResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

type SyncStatusResponse struct {
	SessionID           string  `json:"sessionId"`
	Status              string  `json:"status"`
	SyncType            string  `json:"syncType"`
	TotalRequests       int     `json:"totalRequests"`
	CompletedRequests   int     `json:"completedRequests"`
	FailedRequests      int     `json:"failedRequests"`
	PercentComplete     float64 `json:"percentComplete"`
	StartedAt           string  `json:"startedAt"`
	EstimatedCompletion *string `json:"estimatedCompletion,omitempty"`
	CurrentAction       string  `json:"currentAction"`
}

type UpdateTokenRequest struct {
	DiscogsToken string `json:"discogsToken" validate:"required"`
}

type UpdateTokenResponse struct {
	Message    string `json:"message"`
	TokenValid bool   `json:"tokenValid"`
}

type RateLimitResponse struct {
	Remaining        int    `json:"remaining"`
	Limit            int    `json:"limit"`
	WindowReset      string `json:"windowReset"`
	RecommendedDelay string `json:"recommendedDelay"`
}

func NewDiscogsHandler(app app.App, router fiber.Router) *DiscogsHandler {
	log := logger.New("handlers").File("discogs_handler")
	return &DiscogsHandler{
		app:                  app,
		orchestrationService: app.DiscogsOrchestrationService,
		rateLimitService:     app.DiscogsRateLimitService,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *DiscogsHandler) Register() {
	discogs := h.router.Group("/discogs")

	// Apply authentication middleware to all routes
	protected := discogs.Group("/", h.middleware.RequireAuth(h.app.ZitadelService))

	// Sync endpoints
	protected.Post("/sync-collection", h.InitiateCollectionSync)
	protected.Get("/sync-status/:sessionId", h.GetSyncStatus)
	protected.Post("/sync-cancel/:sessionId", h.CancelSync)
	protected.Post("/sync-resume/:sessionId", h.ResumeSync)
	protected.Post("/sync-pause/:sessionId", h.PauseSync)

	// Token management
	protected.Put("/token", h.UpdateDiscogsToken)
	protected.Get("/token/validate", h.ValidateDiscogsToken)

	// Rate limit info
	protected.Get("/rate-limit", h.GetRateLimit)

	// Sync history
	protected.Get("/syncs", h.GetSyncHistory)
	protected.Get("/syncs/active", h.GetActiveSyncs)
}

func (h *DiscogsHandler) InitiateCollectionSync(c *fiber.Ctx) error {
	log := h.log.Function("InitiateCollectionSync")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req InitiateSyncRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Basic validation
	if req.SyncType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "syncType is required",
		})
	}

	if req.SyncType != "collection" && req.SyncType != "wantlist" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "syncType must be 'collection' or 'wantlist'",
		})
	}

	// Convert sync type
	var syncType models.SyncType
	switch req.SyncType {
	case "collection":
		syncType = models.SyncTypeCollection
	case "wantlist":
		syncType = models.SyncTypeWantlist
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid sync type",
		})
	}

	// Initiate sync
	syncSession, err := h.orchestrationService.InitiateCollectionSync(
		c.Context(),
		user.ID,
		syncType,
		req.FullSync,
		req.PageLimit,
	)
	if err != nil {
		_ = log.Error("Failed to initiate collection sync", "error", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := InitiateSyncResponse{
		SessionID: syncSession.SessionID,
		Status:    string(syncSession.Status),
		Message:   "Collection sync started. Progress will be sent via WebSocket.",
	}

	log.Info("Collection sync initiated", "sessionID", syncSession.SessionID, "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *DiscogsHandler) GetSyncStatus(c *fiber.Ctx) error {
	log := h.log.Function("GetSyncStatus")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get sync progress
	progress, err := h.orchestrationService.GetSyncProgress(c.Context(), sessionID)
	if err != nil {
		_ = log.Error("Failed to get sync progress", "error", err, "sessionID", sessionID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Sync session not found",
		})
	}

	response := SyncStatusResponse{
		SessionID:         progress.SessionID,
		Status:            progress.Status,
		SyncType:          progress.SyncType,
		TotalRequests:     progress.TotalRequests,
		CompletedRequests: progress.CompletedRequests,
		FailedRequests:    progress.FailedRequests,
		PercentComplete:   progress.PercentComplete,
		StartedAt:         progress.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		CurrentAction:     progress.CurrentAction,
	}

	if progress.EstimatedTimeLeft != nil {
		response.EstimatedCompletion = progress.EstimatedTimeLeft
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *DiscogsHandler) CancelSync(c *fiber.Ctx) error {
	log := h.log.Function("CancelSync")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	if err := h.orchestrationService.CancelSync(c.Context(), sessionID); err != nil {
		_ = log.Error("Failed to cancel sync", "error", err, "sessionID", sessionID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info("Sync cancelled", "sessionID", sessionID, "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sync cancelled successfully",
	})
}

func (h *DiscogsHandler) ResumeSync(c *fiber.Ctx) error {
	log := h.log.Function("ResumeSync")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	if err := h.orchestrationService.ResumeSync(c.Context(), sessionID); err != nil {
		_ = log.Error("Failed to resume sync", "error", err, "sessionID", sessionID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info("Sync resumed", "sessionID", sessionID, "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sync resumed successfully",
	})
}

func (h *DiscogsHandler) PauseSync(c *fiber.Ctx) error {
	log := h.log.Function("PauseSync")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	if err := h.orchestrationService.PauseSync(c.Context(), sessionID); err != nil {
		_ = log.Error("Failed to pause sync", "error", err, "sessionID", sessionID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info("Sync paused", "sessionID", sessionID, "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sync paused successfully",
	})
}

func (h *DiscogsHandler) UpdateDiscogsToken(c *fiber.Ctx) error {
	log := h.log.Function("UpdateDiscogsToken")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req UpdateTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Basic validation
	if req.DiscogsToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "discogsToken is required",
		})
	}

	// Update user's Discogs token
	user.DiscogsToken = &req.DiscogsToken
	if err := h.app.UserRepo.Update(c.Context(), user); err != nil {
		_ = log.Error("Failed to update Discogs token", "error", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update token",
		})
	}

	// Reset rate limit for user
	if err := h.rateLimitService.ResetUserRateLimit(c.Context(), user.ID); err != nil {
		log.Warn("Failed to reset rate limit", "error", err, "userID", user.ID)
	}

	response := UpdateTokenResponse{
		Message:    "Discogs token updated successfully",
		TokenValid: true, // We could validate this by making a test API call
	}

	log.Info("Discogs token updated", "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *DiscogsHandler) ValidateDiscogsToken(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	hasToken := user.DiscogsToken != nil && *user.DiscogsToken != ""

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"hasToken":   hasToken,
		"tokenValid": hasToken, // We could validate this by making a test API call
	})
}

func (h *DiscogsHandler) GetRateLimit(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	rateLimit, err := h.rateLimitService.GetRateLimit(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get rate limit information",
		})
	}

	response := RateLimitResponse{
		Remaining:        rateLimit.Remaining,
		Limit:            rateLimit.Limit,
		WindowReset:      rateLimit.WindowReset.Format("2006-01-02T15:04:05Z07:00"),
		RecommendedDelay: rateLimit.RecommendedDelay.String(),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *DiscogsHandler) GetSyncHistory(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get query parameters for pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// This would need to be implemented in the sync repository
	// For now, return a placeholder response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"syncs": []interface{}{},
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": 0,
		},
	})
}

func (h *DiscogsHandler) GetActiveSyncs(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	activeSyncs, err := h.orchestrationService.GetActiveSyncs(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get active syncs",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"activeSyncs": activeSyncs,
	})
}

// ProcessApiResponse handles API responses from the client
func (h *DiscogsHandler) ProcessApiResponse(ctx context.Context, response *types.ApiResponse) error {
	return h.orchestrationService.ProcessApiResponse(ctx, response.RequestID, response)
}