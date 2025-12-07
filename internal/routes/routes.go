package routes

import (
	"event-metrics-service/internal/controller"

	"github.com/gofiber/fiber/v2"
)

// Register attaches all HTTP routes to the Fiber app.
func Register(app *fiber.App, eventController controller.EventController) {
	app.Post("/events", eventController.CreateEvent)
	app.Get("/metrics", eventController.GetMetrics)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
