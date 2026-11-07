package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type WebhookEventType string

const (
	WebhookEmailSent            WebhookEventType = "email.sent"
	WebhookEmailDelivered       WebhookEventType = "email.delivered"
	WebhookEmailOpened          WebhookEventType = "email.opened"
	WebhookEmailClicked         WebhookEventType = "email.clicked"
	WebhookEmailBounced         WebhookEventType = "email.bounced"
	WebhookEmailComplained      WebhookEventType = "email.complained"
	WebhookEmailDeliveryDelayed WebhookEventType = "email.delivery_delayed"
	WebhookEmailRejected        WebhookEventType = "email.rejected"
	WebhookEmailFailed          WebhookEventType = "email.failed"
	WebhookEmailUnsubscribed    WebhookEventType = "email.unsubscribed"
	WebhookContactCreated       WebhookEventType = "contact.created"
	WebhookContactUpdated       WebhookEventType = "contact.updated"
	WebhookContactDeleted       WebhookEventType = "contact.deleted"
)

type EndpointStatus string

const (
	EndpointEnabled  EndpointStatus = "enabled"
	EndpointDisabled EndpointStatus = "disabled"
)

type CreateWebhookParams struct {
	Name   string             `json:"name"`
	URL    string             `json:"url"`
	Events []WebhookEventType `json:"events"`
}

type UpdateWebhookParams struct {
	Name   Nullable[string]   `json:"name,omitzero"`
	URL    Nullable[string]   `json:"url,omitzero"`
	Events []WebhookEventType `json:"events,omitzero"`
	Status EndpointStatus     `json:"status,omitempty"`
}

type WebhookEndpoint struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	URL            string             `json:"url"`
	Events         []WebhookEventType `json:"events"`
	Status         EndpointStatus     `json:"status"`
	DisabledReason *string            `json:"disabled_reason"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type CreatedWebhookEndpoint struct {
	WebhookEndpoint
	Secret string `json:"secret"`
}

type WebhookEndpointListItem struct {
	WebhookEndpoint
	SuccessRate    *float64   `json:"success_rate"`
	LastDeliveryAt *time.Time `json:"last_delivery_at"`
}

type WebhookDelivery struct {
	DeliveryID    string     `json:"delivery_id"`
	EventID       string     `json:"event_id"`
	EventType     string     `json:"event_type"`
	EndpointID    string     `json:"endpoint_id"`
	EmailID       string     `json:"email_id"`
	Status        int        `json:"status"`
	Attempt       int        `json:"attempt"`
	DurationMs    int        `json:"duration_ms"`
	CreatedAt     time.Time  `json:"created_at"`
	LastAttemptAt time.Time  `json:"last_attempt_at"`
	NextAttemptAt *time.Time `json:"next_attempt_at"`
}

type WebhookEndpointDetail struct {
	WebhookEndpoint
	Deliveries24h    int               `json:"deliveries_24h"`
	SuccessRate      *float64          `json:"success_rate"`
	AvgDurationMs    *int              `json:"avg_duration_ms"`
	Retrying         int               `json:"retrying"`
	RecentDeliveries []WebhookDelivery `json:"recent_deliveries"`
}

type WebhookEndpointPage struct {
	Data       []WebhookEndpointListItem `json:"data"`
	NextCursor *string                   `json:"next_cursor"`
	PrevCursor *string                   `json:"prev_cursor"`
}

type WebhookService struct {
	client *Client
}

func (s *WebhookService) List(ctx context.Context, params PageParams, opts ...RequestOption) (*WebhookEndpointPage, error) {
	query := url.Values{}
	setPage(query, params.After, params.Before, params.PerPage)

	return call[WebhookEndpointPage](ctx, s.client, http.MethodGet, withQuery("/v1/webhooks", query), nil, opts)
}

func (s *WebhookService) Create(ctx context.Context, params CreateWebhookParams, opts ...RequestOption) (*CreatedWebhookEndpoint, error) {
	return call[CreatedWebhookEndpoint](ctx, s.client, http.MethodPost, "/v1/webhooks", params, opts)
}

func (s *WebhookService) Get(ctx context.Context, endpointID string, opts ...RequestOption) (*WebhookEndpointDetail, error) {
	return call[WebhookEndpointDetail](ctx, s.client, http.MethodGet, buildPath("webhooks", endpointID), nil, opts)
}

func (s *WebhookService) Update(ctx context.Context, endpointID string, params UpdateWebhookParams, opts ...RequestOption) (*WebhookEndpoint, error) {
	return call[WebhookEndpoint](ctx, s.client, http.MethodPatch, buildPath("webhooks", endpointID), params, opts)
}

func (s *WebhookService) Delete(ctx context.Context, endpointID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("webhooks", endpointID), nil, opts)
}
