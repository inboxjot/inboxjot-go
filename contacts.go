package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type ContactStatus string

const (
	ContactSubscribed    ContactStatus = "subscribed"
	ContactPending       ContactStatus = "pending"
	ContactNotSubscribed ContactStatus = "not_subscribed"
	ContactUnsubscribed  ContactStatus = "unsubscribed"
)

type TopicStatus string

const (
	TopicSubscribed   TopicStatus = "subscribed"
	TopicUnsubscribed TopicStatus = "unsubscribed"
)

type CreateContactParams struct {
	Email      string           `json:"email"`
	FirstName  Nullable[string] `json:"first_name,omitzero"`
	LastName   Nullable[string] `json:"last_name,omitzero"`
	Language   string           `json:"language,omitempty"`
	Notes      Nullable[string] `json:"notes,omitzero"`
	Properties map[string]any   `json:"properties,omitempty"`
	Subscribed *bool            `json:"subscribed,omitempty"`
	SegmentIDs []string         `json:"segment_ids,omitzero"`
}

type UpdateContactParams struct {
	FirstName  Nullable[string] `json:"first_name,omitzero"`
	LastName   Nullable[string] `json:"last_name,omitzero"`
	Language   string           `json:"language,omitempty"`
	Notes      Nullable[string] `json:"notes,omitzero"`
	Properties map[string]any   `json:"properties,omitempty"`
	Subscribed *bool            `json:"subscribed,omitempty"`
	SegmentIDs []string         `json:"segment_ids,omitzero"`
	Topics     []TopicChoice    `json:"topics,omitzero"`
}

type TopicChoice struct {
	ID     string      `json:"id"`
	Status TopicStatus `json:"status"`
}

type SavedContact struct {
	Contact
	Created bool `json:"created"`
}

type SegmentRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ContactTopic struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Subscribed bool           `json:"subscribed"`
	Status     *ContactStatus `json:"status"`
}

type Contact struct {
	ID         string         `json:"id"`
	Email      string         `json:"email"`
	FirstName  *string        `json:"first_name"`
	LastName   *string        `json:"last_name"`
	Status     ContactStatus  `json:"status"`
	Language   string         `json:"language"`
	Properties map[string]any `json:"properties"`
	Segments   []SegmentRef   `json:"segments"`
	Topics     []ContactTopic `json:"topics"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ErasureStatus string

const ErasureQueued ErasureStatus = "queued"

type EraseContactParams struct {
	Email string `json:"email"`
}

type EraseContactResponse struct {
	Status ErasureStatus `json:"status"`
}

type ListContactsParams struct {
	Sort      string // "added", "contact" or "last_activity"
	Direction string // "asc" or "desc"
	Status    ContactStatus
	SegmentID string
	Query     string
	After     string
	Before    string
	PerPage   int
}

type LastActivity struct {
	Type string `json:"type"`
	At   string `json:"at"`
}

type ContactListItem struct {
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	FirstName    *string        `json:"first_name"`
	LastName     *string        `json:"last_name"`
	Properties   map[string]any `json:"properties"`
	Status       ContactStatus  `json:"status"`
	IsSuppressed bool           `json:"is_suppressed"`
	Segments     []SegmentRef   `json:"segments"`
	LastActivity *LastActivity  `json:"last_activity"`
	CreatedAt    string         `json:"created_at"`
}

type ContactPage struct {
	Data       []ContactListItem `json:"data"`
	Total      int64             `json:"total"`
	NextCursor *string           `json:"next_cursor"`
	PrevCursor *string           `json:"prev_cursor"`
}

type ContactService struct {
	client *Client
}

func (s *ContactService) Create(ctx context.Context, params CreateContactParams, opts ...RequestOption) (*SavedContact, error) {
	return call[SavedContact](ctx, s.client, http.MethodPost, "/v1/contacts", params, opts)
}

func (s *ContactService) List(ctx context.Context, params ListContactsParams, opts ...RequestOption) (*ContactPage, error) {
	query := url.Values{}
	setQuery(query, "sort", params.Sort)
	setQuery(query, "direction", params.Direction)
	setQuery(query, "status", string(params.Status))
	setQuery(query, "segment_id", params.SegmentID)
	setQuery(query, "query", params.Query)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[ContactPage](ctx, s.client, http.MethodGet, withQuery("/v1/contacts", query), nil, opts)
}

func (s *ContactService) Get(ctx context.Context, contactIDOrEmail string, opts ...RequestOption) (*Contact, error) {
	return call[Contact](ctx, s.client, http.MethodGet, buildPath("contacts", contactIDOrEmail), nil, opts)
}

func (s *ContactService) Update(ctx context.Context, contactIDOrEmail string, params UpdateContactParams, opts ...RequestOption) (*Contact, error) {
	return call[Contact](ctx, s.client, http.MethodPatch, buildPath("contacts", contactIDOrEmail), params, opts)
}

func (s *ContactService) Delete(ctx context.Context, contactIDOrEmail string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("contacts", contactIDOrEmail), nil, opts)
}

func (s *ContactService) Erase(ctx context.Context, params EraseContactParams, opts ...RequestOption) (*EraseContactResponse, error) {
	return call[EraseContactResponse](ctx, s.client, http.MethodPost, "/v1/contacts/erase", params, opts)
}
