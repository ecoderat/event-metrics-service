package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"event-metrics-service/internal/config"
	"event-metrics-service/internal/controller"
)

// Server wraps the Fiber application setup.
type Server struct {
	app *fiber.App
}

// NewServer configures routes and middleware.
func NewServer(appCfg *config.Config, eventController controller.EventController) *Server {
	isBenchmark := strings.ToLower(appCfg.AppMode) == "benchmark"
	fiberCfg := fiber.Config{
		DisableStartupMessage: true,
		Prefork:               appCfg.FiberPrefork,
	}
	app := fiber.New(fiberCfg)
	if !isBenchmark {
		// app.Use(logger.New())
	}
	app.Use(recover.New())

	app.Post("/events", eventController.CreateEvent)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	return &Server{app: app}
}

// Listen runs the server on provided addr.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}
