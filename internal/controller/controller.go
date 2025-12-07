package controller

import (
	"strconv"
	"time"

	"event-metrics-service/internal/model"
	"event-metrics-service/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
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

	h.eventService.ProcessEvent(c.Context(), event)

	return c.SendStatus(fiber.StatusAccepted)
}

// GetMetrics returns aggregated metrics for events.
func (h *eventController) GetMetrics(c *fiber.Ctx) error {
	filter, err := buildMetricsFilter(c)
	if err != nil {
		return err
	}

	resp, svcErr := h.eventService.GetMetrics(c.Context(), filter)
	if svcErr != nil {
		if _, ok := svcErr.(*service.ValidationError); ok {
			return fiber.NewError(fiber.StatusBadRequest, svcErr.Error())
		}

		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch metrics")
	}

	return c.JSON(resp)
}

func buildMetricsFilter(c *fiber.Ctx) (model.MetricsFilter, error) {
	eventName := utils.Trim(c.Query("event_name"), ' ')
	if eventName == "" {
		return model.MetricsFilter{}, fiber.NewError(fiber.StatusBadRequest, "event_name is required")
	}

	groupBy := utils.Trim(c.Query("group_by", "channel"), ' ')

	var from, to time.Time

	if raw := utils.Trim(c.Query("from"), ' '); raw != "" {
		sec, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			return model.MetricsFilter{}, fiber.NewError(fiber.StatusBadRequest, "invalid from timestamp")
		}
		from = time.Unix(sec, 0).UTC()
	}

	if raw := utils.Trim(c.Query("to"), ' '); raw != "" {
		sec, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			return model.MetricsFilter{}, fiber.NewError(fiber.StatusBadRequest, "invalid to timestamp")
		}
		to = time.Unix(sec, 0).UTC()
	}

	var channel *string
	if raw := utils.Trim(c.Query("channel"), ' '); raw != "" {
		channel = &raw
	}

	return model.MetricsFilter{
		EventName: eventName,
		GroupBy:   groupBy,
		From:      from,
		To:        to,
		Channel:   channel,
	}, nil
}
