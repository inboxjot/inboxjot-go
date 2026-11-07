package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type SegmentType string

const (
	SegmentStatic  SegmentType = "static"
	SegmentDynamic SegmentType = "dynamic"
)

type SegmentMatch string

const (
	SegmentMatchAll SegmentMatch = "all"
	SegmentMatchAny SegmentMatch = "any"
)

type SegmentOperator string

const (
	OperatorIs          SegmentOperator = "is"
	OperatorIsNot       SegmentOperator = "is_not"
	OperatorContains    SegmentOperator = "contains"
	OperatorNotContains SegmentOperator = "not_contains"
	OperatorStartsWith  SegmentOperator = "starts_with"
	OperatorEndsWith    SegmentOperator = "ends_with"
	OperatorIsSet       SegmentOperator = "is_set"
	OperatorIsNotSet    SegmentOperator = "is_not_set"
	OperatorEquals      SegmentOperator = "equals"
	OperatorNotEquals   SegmentOperator = "not_equals"
	OperatorGreaterThan SegmentOperator = "greater_than"
	OperatorLessThan    SegmentOperator = "less_than"
	OperatorWithin      SegmentOperator = "within"
	OperatorNotWithin   SegmentOperator = "not_within"
	OperatorBefore      SegmentOperator = "before"
	OperatorAfter       SegmentOperator = "after"
	OperatorIn          SegmentOperator = "in"
	OperatorNotIn       SegmentOperator = "not_in"
	OperatorAtLeast     SegmentOperator = "at_least"
	OperatorNever       SegmentOperator = "never"
)

type RuleWindow struct {
	Count float64 `json:"count"`
	Days  float64 `json:"days"`
}

type SegmentRule struct {
	Field    string          `json:"field"`
	Operator SegmentOperator `json:"operator"`
	Value    any             `json:"value,omitempty"`
}

type SegmentConditions struct {
	Match SegmentMatch  `json:"match"`
	Rules []SegmentRule `json:"rules"`
}

type CreateSegmentParams struct {
	Name        string             `json:"name"`
	Description Nullable[string]   `json:"description,omitzero"`
	Type        SegmentType        `json:"type"`
	Conditions  *SegmentConditions `json:"conditions,omitempty"`
}

type UpdateSegmentParams struct {
	Name        Nullable[string]   `json:"name,omitzero"`
	Description Nullable[string]   `json:"description,omitzero"`
	Conditions  *SegmentConditions `json:"conditions,omitempty"`
}

type ListSegmentsParams struct {
	Type      SegmentType
	Query     string
	Sort      string // "created", "name" or "contacts"
	Direction string // "asc" or "desc"
	After     string
	Before    string
	PerPage   int
}

type ListSegmentMembersParams struct {
	Query   string
	After   string
	Before  string
	PerPage int
}

type AddSegmentMembersParams struct {
	ContactIDs []string `json:"contact_ids"`
}

type RemoveSegmentMembersParams struct {
	ContactIDs []string `json:"contact_ids"`
}

type Segment struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  *string            `json:"description"`
	Type         SegmentType        `json:"type"`
	Conditions   *SegmentConditions `json:"conditions"`
	ContactCount *int               `json:"contact_count"`
	CountedAt    *time.Time         `json:"counted_at"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type SegmentListItem struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  *string     `json:"description"`
	Type         SegmentType `json:"type"`
	ContactCount *int        `json:"contact_count"`
	CountedAt    *time.Time  `json:"counted_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type SegmentPage struct {
	Data       []SegmentListItem `json:"data"`
	Total      int64             `json:"total"`
	NextCursor *string           `json:"next_cursor"`
	PrevCursor *string           `json:"prev_cursor"`
}

type SegmentMember struct {
	ID        string        `json:"id"`
	Email     string        `json:"email"`
	FirstName *string       `json:"first_name"`
	LastName  *string       `json:"last_name"`
	Status    ContactStatus `json:"status"`
	JoinedAt  time.Time     `json:"joined_at"`
}

type SegmentMemberPage struct {
	Data       []SegmentMember `json:"data"`
	NextCursor *string         `json:"next_cursor"`
	PrevCursor *string         `json:"prev_cursor"`
}

type AddedMembers struct {
	Added int64 `json:"added"`
}

type RemovedMembers struct {
	Removed int64 `json:"removed"`
}

type SegmentService struct {
	client *Client
}

func (s *SegmentService) List(ctx context.Context, params ListSegmentsParams, opts ...RequestOption) (*SegmentPage, error) {
	query := url.Values{}
	setQuery(query, "type", string(params.Type))
	setQuery(query, "query", params.Query)
	setQuery(query, "sort", params.Sort)
	setQuery(query, "direction", params.Direction)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[SegmentPage](ctx, s.client, http.MethodGet, withQuery("/v1/segments", query), nil, opts)
}

func (s *SegmentService) Create(ctx context.Context, params CreateSegmentParams, opts ...RequestOption) (*Segment, error) {
	return call[Segment](ctx, s.client, http.MethodPost, "/v1/segments", params, opts)
}

func (s *SegmentService) Get(ctx context.Context, segmentID string, opts ...RequestOption) (*Segment, error) {
	return call[Segment](ctx, s.client, http.MethodGet, buildPath("segments", segmentID), nil, opts)
}

func (s *SegmentService) Update(ctx context.Context, segmentID string, params UpdateSegmentParams, opts ...RequestOption) (*Segment, error) {
	return call[Segment](ctx, s.client, http.MethodPatch, buildPath("segments", segmentID), params, opts)
}

func (s *SegmentService) Delete(ctx context.Context, segmentID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("segments", segmentID), nil, opts)
}

func (s *SegmentService) ListMembers(ctx context.Context, segmentID string, params ListSegmentMembersParams, opts ...RequestOption) (*SegmentMemberPage, error) {
	query := url.Values{}
	setQuery(query, "query", params.Query)
	setPage(query, params.After, params.Before, params.PerPage)

	path := withQuery(buildPath("segments", segmentID, "members"), query)

	return call[SegmentMemberPage](ctx, s.client, http.MethodGet, path, nil, opts)
}

func (s *SegmentService) AddMembers(ctx context.Context, segmentID string, params AddSegmentMembersParams, opts ...RequestOption) (*AddedMembers, error) {
	return call[AddedMembers](ctx, s.client, http.MethodPost, buildPath("segments", segmentID, "members"), params, opts)
}

func (s *SegmentService) RemoveMembers(ctx context.Context, segmentID string, params RemoveSegmentMembersParams, opts ...RequestOption) (*RemovedMembers, error) {
	return call[RemovedMembers](ctx, s.client, http.MethodDelete, buildPath("segments", segmentID, "members"), params, opts)
}
