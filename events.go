package inboxjot

import (
	"context"
	"net/http"
)

type SendEventParams struct {
	Name  string         `json:"name"`
	Email string         `json:"email"`
	Data  map[string]any `json:"data,omitempty"`
}

type SendEventResponse struct {
	ContactID string `json:"contact_id"`
}

type EventService struct {
	client *Client
}

func (s *EventService) Send(ctx context.Context, params SendEventParams, opts ...RequestOption) (*SendEventResponse, error) {
	return call[SendEventResponse](ctx, s.client, http.MethodPost, "/v1/events", params, opts)
}
