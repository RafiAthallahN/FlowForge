package controllers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flow-forger/flow-forger/backend/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// StepEvent represents a real-time status change for a workflow step.
type StepEvent struct {
	RunID         string `json:"run_id"`
	StepID        string `json:"step_id"`
	Status        string `json:"status"`
	LogLine       string `json:"log_line,omitempty"`
	Duration      int64  `json:"duration_ms,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
	SuggestedFix  string `json:"suggested_fix,omitempty"`
}

// EventHub manages SSE client connections grouped by tenant ID.
type EventHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan StepEvent]struct{} // tenantID -> set of channels
}

func NewEventHub() *EventHub {
	return &EventHub{
		clients: make(map[string]map[chan StepEvent]struct{}),
	}
}

func (h *EventHub) Subscribe(tenantID string) chan StepEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan StepEvent, 64)
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[chan StepEvent]struct{})
	}
	h.clients[tenantID][ch] = struct{}{}
	return ch
}

func (h *EventHub) Unsubscribe(tenantID string, ch chan StepEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.clients[tenantID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(h.clients, tenantID)
		}
	}
}

// Publish sends a StepEvent to all subscribers of the given tenant.
func (h *EventHub) Publish(tenantID string, event StepEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if subs, ok := h.clients[tenantID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
				// Drop event if client buffer is full (non-blocking)
			}
		}
	}
}

type SSEController struct {
	hub *EventHub
}

func NewSSEController(hub *EventHub) *SSEController {
	return &SSEController{hub: hub}
}

// Stream handles the SSE endpoint. It keeps the connection open and streams
// step status events as they happen for the authenticated tenant.
func (ctrl *SSEController) Stream(c *fiber.Ctx) error {
	tenantID, _ := c.UserContext().Value(domain.ContextKeyTenantID).(string)
	if tenantID == "" {
		tenantID, _ = c.UserContext().Value("tenant_id").(string)
	}
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant context required"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ch := ctrl.hub.Subscribe(tenantID)
		defer ctrl.hub.Unsubscribe(tenantID, ch)

		// Send initial keepalive
		fmt.Fprintf(w, ": connected\n\n")
		w.Flush()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				if err := w.Flush(); err != nil {
					return // Client disconnected
				}
			case <-ticker.C:
				// Send keepalive comment to prevent connection timeout
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	// Tell fasthttp to hijack the connection for streaming
	c.Context().Response.Header.Set("Content-Type", "text/event-stream")
	c.Context().Response.SetStatusCode(fasthttp.StatusOK)

	return nil
}
