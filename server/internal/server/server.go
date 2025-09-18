package server

import (
	"fmt"
	"time"
	"waugzee/internal/app"
	"waugzee/internal/handlers"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogs "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/helmet/v2"
)

type AppServer struct {
	FiberApp *fiber.App
	log      logger.Logger
}

func New(app *app.App) (*AppServer, error) {
	log := logger.New("server").Function("New")
	log.Info("Initializing server")

	config := fiber.Config{
		ServerHeader: fmt.Sprintf(
			"APIServer/%s",
			app.Config.GeneralVersion,
		),
		AppName:                  "waugzee_server",
		BodyLimit:                10 * 1024 * 1024,
		ReadBufferSize:           16384,
		WriteBufferSize:          16384,
		StreamRequestBody:        false,
		EnableSplittingOnParsers: true,
		EnableTrustedProxyCheck:  true,
		ReadTimeout:              30 * time.Second,
		WriteTimeout:             30 * time.Second,
		IdleTimeout:              120 * time.Second,
		DisableStartupMessage:    true,
		EnablePrintRoutes:        false,
	}

	if app.Config.Environment == "development" {
		log.Info("Enabling development mode")
		config.DisableStartupMessage = false
		config.EnablePrintRoutes = true
	}

	server := fiber.New(config)

	server.Use(cors.New(cors.Config{
		AllowOrigins:     app.Config.CorsAllowOrigins,
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, withCredentials, X-Response-Type, Upgrade, Connection, X-Client-Type",
		AllowCredentials: true,
		MaxAge:           300,
		ExposeHeaders:    "Upgrade, X-Auth-Token",
	}))

	server.Use(fiberLogs.New())
	server.Use(compress.New())

	// Enhanced security headers
	server.Use(helmet.New(helmet.Config{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
		// CSP will be handled per-route basis for more flexibility
		ContentSecurityPolicy: "",
	}))

	fiberApp := &AppServer{
		FiberApp: server,
		log:      log,
	}

	if err := handlers.Router(server, app); err != nil {
		log.Er("failed to initialize handlers", err)
		return &AppServer{}, log.Err("failed to initialize handlers", err)
	}

	return fiberApp, nil
}

func (s *AppServer) Listen(port int) error {
	log := s.log.Function("Listen")

	if port == 0 {
		return log.Error(
			"Fatal error: invalid port",
			"port", port,
		)
	}

	log.Info("Starting server", "port", port)
	return s.FiberApp.Listen(fmt.Sprintf(":%d", port))
}
