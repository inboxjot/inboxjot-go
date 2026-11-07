package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignScheduled CampaignStatus = "scheduled"
	CampaignSending   CampaignStatus = "sending"
	CampaignPaused    CampaignStatus = "paused"
	CampaignFailed    CampaignStatus = "failed"
	CampaignSent      CampaignStatus = "sent"
	CampaignCanceled  CampaignStatus = "canceled"
)

type CampaignPauseReason string

const (
	PauseManual           CampaignPauseReason = "manual"
	PauseDailyLimit       CampaignPauseReason = "daily_limit"
	PauseWarmup           CampaignPauseReason = "warmup"
	PauseSendingPaused    CampaignPauseReason = "sending_paused"
	PauseDomainUnverified CampaignPauseReason = "domain_unverified"
	PauseNoPlan           CampaignPauseReason = "no_plan"
	PausePlanLimit        CampaignPauseReason = "plan_limit"
)

type CampaignParams struct {
	Name              Nullable[string] `json:"name,omitzero"`
	From              Nullable[string] `json:"from,omitzero"`
	ReplyTo           Nullable[string] `json:"reply_to,omitzero"`
	Subject           Nullable[string] `json:"subject,omitzero"`
	Preheader         Nullable[string] `json:"preheader,omitzero"`
	HTML              Nullable[string] `json:"html,omitzero"`
	Content           map[string]any   `json:"content,omitempty"`
	TemplateID        Nullable[string] `json:"template_id,omitzero"`
	IncludeSegmentIDs []string         `json:"include_segment_ids,omitzero"`
	ExcludeSegmentIDs []string         `json:"exclude_segment_ids,omitzero"`
	TopicID           Nullable[string] `json:"topic_id,omitzero"`
}

type ListCampaignsParams struct {
	Query     string
	Status    []CampaignStatus
	Sort      string // "sent" or "name"
	Direction string // "asc" or "desc"
	After     string
	Before    string
	PerPage   int
}

type SendCampaignParams struct {
	ScheduledAt time.Time `json:"scheduled_at,omitzero"`
}

type SendTestCampaignParams struct {
	To []string `json:"to"`
}

type Named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Campaign struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Status            CampaignStatus       `json:"status"`
	From              string               `json:"from"`
	ReplyTo           *string              `json:"reply_to"`
	Subject           string               `json:"subject"`
	Preheader         string               `json:"preheader"`
	HTML              *string              `json:"html"`
	Text              *string              `json:"text"`
	TemplateID        *string              `json:"template_id"`
	Template          *Named               `json:"template"`
	IncludeSegmentIDs []string             `json:"include_segment_ids"`
	IncludeSegments   []Named              `json:"include_segments"`
	ExcludeSegmentIDs []string             `json:"exclude_segment_ids"`
	ExcludeSegments   []Named              `json:"exclude_segments"`
	TopicID           string               `json:"topic_id"`
	Topic             *Named               `json:"topic"`
	AudienceCount     *int                 `json:"audience_count"`
	QueuedCount       int                  `json:"queued_count"`
	SkippedCount      int                  `json:"skipped_count"`
	PausedReason      *CampaignPauseReason `json:"paused_reason"`
	FailureReason     *string              `json:"failure_reason"`
	ScheduledAt       *time.Time           `json:"scheduled_at"`
	StartedAt         *time.Time           `json:"started_at"`
	CompletedAt       *time.Time           `json:"completed_at"`
	WarmupNextLimit   *int64               `json:"warmup_next_limit"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type CampaignSummary struct {
	Delivered int `json:"delivered"`
	Opened    int `json:"opened"`
	Clicked   int `json:"clicked"`
}

type CampaignListItem struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Status          CampaignStatus       `json:"status"`
	Subject         string               `json:"subject"`
	IncludeSegments []Named              `json:"include_segments"`
	ExcludeSegments []Named              `json:"exclude_segments"`
	Topic           *Named               `json:"topic"`
	AudienceCount   *int                 `json:"audience_count"`
	QueuedCount     int                  `json:"queued_count"`
	SkippedCount    int                  `json:"skipped_count"`
	PausedReason    *CampaignPauseReason `json:"paused_reason"`
	ScheduledAt     *time.Time           `json:"scheduled_at"`
	StartedAt       *time.Time           `json:"started_at"`
	CompletedAt     *time.Time           `json:"completed_at"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Stats           *CampaignSummary     `json:"stats"`
}

type CampaignPage struct {
	Data       []CampaignListItem `json:"data"`
	NextCursor *string            `json:"next_cursor"`
	PrevCursor *string            `json:"prev_cursor"`
}

type QueuedIDs struct {
	IDs []string `json:"ids"`
}

type CampaignService struct {
	client *Client
}

func (s *CampaignService) List(ctx context.Context, params ListCampaignsParams, opts ...RequestOption) (*CampaignPage, error) {
	query := url.Values{}
	setQuery(query, "query", params.Query)
	setStatuses(query, params.Status)
	setQuery(query, "sort", params.Sort)
	setQuery(query, "direction", params.Direction)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[CampaignPage](ctx, s.client, http.MethodGet, withQuery("/v1/campaigns", query), nil, opts)
}

func (s *CampaignService) Create(ctx context.Context, params CampaignParams, opts ...RequestOption) (*Campaign, error) {
	return call[Campaign](ctx, s.client, http.MethodPost, "/v1/campaigns", params, opts)
}

func (s *CampaignService) Get(ctx context.Context, campaignID string, opts ...RequestOption) (*Campaign, error) {
	return call[Campaign](ctx, s.client, http.MethodGet, buildPath("campaigns", campaignID), nil, opts)
}

func (s *CampaignService) Update(ctx context.Context, campaignID string, params CampaignParams, opts ...RequestOption) (*Campaign, error) {
	return call[Campaign](ctx, s.client, http.MethodPatch, buildPath("campaigns", campaignID), params, opts)
}

func (s *CampaignService) Delete(ctx context.Context, campaignID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("campaigns", campaignID), nil, opts)
}

func (s *CampaignService) Send(ctx context.Context, campaignID string, params SendCampaignParams, opts ...RequestOption) (*Campaign, error) {
	return call[Campaign](ctx, s.client, http.MethodPost, buildPath("campaigns", campaignID, "send"), params, opts)
}

func (s *CampaignService) Cancel(ctx context.Context, campaignID string, opts ...RequestOption) (*Campaign, error) {
	return call[Campaign](ctx, s.client, http.MethodPost, buildPath("campaigns", campaignID, "cancel"), nil, opts)
}

func (s *CampaignService) SendTest(ctx context.Context, campaignID string, params SendTestCampaignParams, opts ...RequestOption) (*QueuedIDs, error) {
	return call[QueuedIDs](ctx, s.client, http.MethodPost, buildPath("campaigns", campaignID, "test"), params, opts)
}

func (s *CampaignService) ListVersions(ctx context.Context, campaignID string, params ListVersionsParams, opts ...RequestOption) (*DocumentVersionPage, error) {
	path := withQuery(buildPath("campaigns", campaignID, "versions"), buildVersionsQuery(params))

	return call[DocumentVersionPage](ctx, s.client, http.MethodGet, path, nil, opts)
}

func (s *CampaignService) CreateVersion(ctx context.Context, campaignID string, params CreateVersionParams, opts ...RequestOption) (*DocumentVersion, error) {
	return call[DocumentVersion](ctx, s.client, http.MethodPost, buildPath("campaigns", campaignID, "versions"), params, opts)
}

func (s *CampaignService) GetVersion(ctx context.Context, campaignID, versionID string, opts ...RequestOption) (*DocumentVersionDetail, error) {
	path := buildPath("campaigns", campaignID, "versions", versionID)

	return call[DocumentVersionDetail](ctx, s.client, http.MethodGet, path, nil, opts)
}

func (s *CampaignService) UpdateVersion(ctx context.Context, campaignID, versionID string, params UpdateVersionParams, opts ...RequestOption) (*DocumentVersion, error) {
	path := buildPath("campaigns", campaignID, "versions", versionID)

	return call[DocumentVersion](ctx, s.client, http.MethodPatch, path, params, opts)
}

func (s *CampaignService) RestoreVersion(ctx context.Context, campaignID, versionID string, opts ...RequestOption) (*Campaign, error) {
	path := buildPath("campaigns", campaignID, "versions", versionID, "restore")

	return call[Campaign](ctx, s.client, http.MethodPost, path, nil, opts)
}
