package events

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Event struct {
	Name string
	Data any
	Time time.Time
}

type Handler func(Event)

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID       string
	TenantID string
	Events   chan SSEMessage
	Done     chan struct{}
}

// SSEMessage is the message sent to SSE clients
type SSEMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
	Time  string `json:"time"`
}

var (
	bus      = make(chan Event, 1000)
	handlers = make(map[string][]Handler)
	mu       sync.RWMutex

	// SSE client management
	sseClients   = make(map[string]*SSEClient)
	sseClientsMu sync.RWMutex
)

func On(name string, h Handler) {
	mu.Lock()
	defer mu.Unlock()
	handlers[name] = append(handlers[name], h)
}

func Emit(name string, data any) {
	select {
	case bus <- Event{Name: name, Data: data, Time: time.Now()}:
	default:
		log.Warn().Str("event", name).Msg("Event bus full, dropping event")
	}
}

// RegisterSSEClient adds a new SSE client
func RegisterSSEClient(id, tenantID string) *SSEClient {
	client := &SSEClient{
		ID:       id,
		TenantID: tenantID,
		Events:   make(chan SSEMessage, 100),
		Done:     make(chan struct{}),
	}
	sseClientsMu.Lock()
	sseClients[id] = client
	sseClientsMu.Unlock()
	log.Debug().Str("client_id", id).Str("tenant_id", tenantID).Msg("SSE client registered")
	return client
}

// UnregisterSSEClient removes an SSE client
func UnregisterSSEClient(id string) {
	sseClientsMu.Lock()
	if client, ok := sseClients[id]; ok {
		close(client.Done)
		delete(sseClients, id)
	}
	sseClientsMu.Unlock()
	log.Debug().Str("client_id", id).Msg("SSE client unregistered")
}

// BroadcastSSE sends an event to all SSE clients (optionally filtered by tenant)
func BroadcastSSE(event string, data any, tenantID string) {
	msg := SSEMessage{
		Event: event,
		Data:  data,
		Time:  time.Now().Format(time.RFC3339),
	}

	sseClientsMu.RLock()
	defer sseClientsMu.RUnlock()

	for _, client := range sseClients {
		// Send to all clients if tenantID is empty, otherwise filter by tenant
		if tenantID == "" || client.TenantID == tenantID {
			select {
			case client.Events <- msg:
			default:
				log.Warn().Str("client_id", client.ID).Msg("SSE client buffer full, dropping message")
			}
		}
	}
}

// GetSSEClientCount returns the number of connected SSE clients
func GetSSEClientCount() int {
	sseClientsMu.RLock()
	defer sseClientsMu.RUnlock()
	return len(sseClients)
}

func Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-bus:
				mu.RLock()
				hs := handlers[e.Name]
				mu.RUnlock()
				for _, h := range hs {
					go func(handler Handler) {
						defer func() {
							if r := recover(); r != nil {
								log.Error().Interface("panic", r).Str("event", e.Name).Msg("Event handler panic")
							}
						}()
						handler(e)
					}(h)
				}

				// Broadcast to SSE clients for relevant events
				if shouldBroadcast(e.Name) {
					tenantID := extractTenantID(e.Data)
					BroadcastSSE(e.Name, e.Data, tenantID)
				}
			}
		}
	}()
}

// shouldBroadcast determines if an event should be sent to SSE clients
func shouldBroadcast(eventName string) bool {
	broadcastEvents := map[string]bool{
		"datasource.scan.started":             true,
		"datasource.scan.progress":            true,
		"datasource.scan.completed":           true,
		"datasource.scan.failed":              true,
		"datasource.created":                  true,
		"datasource.updated":                  true,
		"datasource.deleted":                  true,
		"job.started":                         true,
		"job.completed":                       true,
		"job.failed":                          true,
		"classification.started":              true,
		"classification.progress":             true,
		"classification.completed":            true,
		"classification.failed":               true,
		"classification.queued":               true,
		"policy.violated":                     true,
		"notification.created":                true,
		"rot.scan.started":                    true,
		"rot.scan.progress":                   true,
		"rot.scan.completed":                  true,
		"rot.scan.failed":                     true,
		"scan.failed":                         true,
		"compliance.assessment.completed":     true,
		"compliance.assessment.failed":        true,
		"report.completed":                    true,
		"document.extracted":                  true,
		"document.extraction.failed":          true,
	}
	return broadcastEvents[eventName]
}

// extractTenantID tries to extract tenant_id from event data
func extractTenantID(data any) string {
	if data == nil {
		return ""
	}

	// Try to marshal and unmarshal to get tenant_id
	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	var m map[string]any
	if err := json.Unmarshal(bytes, &m); err != nil {
		return ""
	}

	if tid, ok := m["tenant_id"].(string); ok {
		return tid
	}
	return ""
}

func init() {
	On("datasource.created", func(e Event) {
		log.Info().Str("event", e.Name).Interface("data", e.Data).Msg("DataSource created")
	})
	On("classification.completed", func(e Event) {
		log.Info().Str("event", e.Name).Msg("Classification completed")
	})
	On("policy.violated", func(e Event) {
		log.Warn().Str("event", e.Name).Interface("data", e.Data).Msg("Policy violated")
	})
	On("gate.query", func(e Event) {
		log.Debug().Str("event", e.Name).Msg("Gate query processed")
	})
}
