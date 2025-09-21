package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
)

type DiscogsHandler struct {
	Handler
	app app.App
}

type InitiateSyncRequest struct {
	SyncType  string `json:"syncType"            validate:"required,oneof=collection wantlist"`
	FullSync  bool   `json:"fullSync"`
	PageLimit *int   `json:"pageLimit,omitempty"`
}

type InitiateSyncResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func NewDiscogsHandler(app app.App, router fiber.Router) *DiscogsHandler {
	log := logger.New("handlers").File("discogs_handler")
	return &DiscogsHandler{
		app: app,
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

	// Only sync initiation - everything else handled via WebSocket or other services
	protected.Post("/syncCollection", h.InitiateCollectionSync)
}

func (h *DiscogsHandler) InitiateCollectionSync(c *fiber.Ctx) error {
	// log := h.log.Function("InitiateCollectionSync")

	// user := middleware.GetUser(c)
	// if user == nil {
	// 	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
	// 		"error": "Authentication required",
	// 	})
	// }
	//
	// var req InitiateSyncRequest
	// if err := c.BodyParser(&req); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "Invalid request body",
	// 	})
	// }
	//
	// // Basic validation
	// if req.SyncType == "" {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "syncType is required",
	// 	})
	// }
	//
	// if req.SyncType != "collection" && req.SyncType != "wantlist" {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "syncType must be 'collection' or 'wantlist'",
	// 	})
	// }
	//
	// // Convert sync type
	// var syncType models.SyncType
	// switch req.SyncType {
	// case "collection":
	// 	syncType = models.SyncTypeCollection
	// case "wantlist":
	// 	syncType = models.SyncTypeWantlist
	// default:
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "Invalid sync type",
	// 	})
	// }
	//
	// // Initiate sync
	// syncSession, err := h.orchestrationService.InitiateCollectionSync(
	// 	c.Context(),
	// 	user.ID,
	// 	syncType,
	// 	req.FullSync,
	// 	req.PageLimit,
	// )
	// if err != nil {
	// 	_ = log.Error("Failed to initiate collection sync", "error", err, "userID", user.ID)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"error": err.Error(),
	// 	})
	// }
	//
	// response := InitiateSyncResponse{
	// 	SessionID: syncSession.SessionID,
	// 	Status:    string(syncSession.Status),
	// 	Message:   "Collection sync started. Progress will be sent via WebSocket.",
	// }
	//
	// log.Info("Collection sync initiated", "sessionID", syncSession.SessionID, "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}
