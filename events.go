package inboxjot

import (
	"context"
	"net/http"
	"time"
)

type SendEventParams struct {
	Name       string         `json:"name"`
	Email      string         `json:"email,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	OccurredAt time.Time      `json:"occurred_at,omitzero"`
	Data       map[string]any `json:"data,omitempty"`
}

type SendEventResponse struct {
	ContactID string `json:"contact_id"`
}

type BatchEventResponse struct {
	Data []SendEventResponse `json:"data"`
}

type EventService struct {
	client *Client
}

func (s *EventService) Send(ctx context.Context, params SendEventParams, opts ...RequestOption) (*SendEventResponse, error) {
	return call[SendEventResponse](ctx, s.client, http.MethodPost, "/v1/events", params, opts)
}

func (s *EventService) Batch(ctx context.Context, events []SendEventParams, opts ...RequestOption) (*BatchEventResponse, error) {
	return call[BatchEventResponse](ctx, s.client, http.MethodPost, "/v1/events/batch", events, opts)
}
