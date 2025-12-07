package controller

import (
	"event-metrics-service/internal/model"
	"event-metrics-service/internal/service"

	"github.com/gofiber/fiber/v2"
)

type EventController interface {
	CreateEvent(c *fiber.Ctx) error
	GetMetrics(c *fiber.Ctx) error
}

// EventHandler exposes HTTP handlers for ingestion endpoints.
type eventController struct {
	eventService service.EventService
}

// NewEventController builds an EventController.
func NewEventController(svc service.EventService) EventController {
	return &eventController{eventService: svc}
}

// CreateEvent accepts single event payloads.
func (h *eventController) CreateEvent(c *fiber.Ctx) error {
	var req model.EventRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json payload")
	}

	event, err := h.eventService.BuildEvent(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_, err = h.eventService.ProcessEvent(c.Context(), event)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to process event")
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "accepted", "event": event})
}

// GetMetrics returns aggregated metrics for events.
func (h *eventController) GetMetrics(c *fiber.Ctx) error {
	return nil
}
