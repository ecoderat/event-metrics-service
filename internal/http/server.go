package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	routes "event-metrics-service/internal"
	"event-metrics-service/internal/config"
	"event-metrics-service/internal/controller"
)

// Server wraps the Fiber application setup.
type Server struct {
	app *fiber.App
}

// NewServer configures routes and middleware.
func NewServer(appCfg *config.Config, eventController controller.EventController) *Server {
	fiberCfg := fiber.Config{
		DisableStartupMessage: true,
		Prefork:               appCfg.FiberPrefork,
	}
	app := fiber.New(fiberCfg)
	// app.Use(logger.New())
	app.Use(recover.New())

	routes.Register(app, eventController)

	return &Server{app: app}
}

// Listen runs the server on provided addr.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}
