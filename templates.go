package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type CreateTemplateParams struct {
	Name      string           `json:"name"`
	From      Nullable[string] `json:"from,omitzero"`
	ReplyTo   Nullable[string] `json:"reply_to,omitzero"`
	Subject   Nullable[string] `json:"subject,omitzero"`
	Preheader Nullable[string] `json:"preheader,omitzero"`
	HTML      Nullable[string] `json:"html,omitzero"`
	Content   map[string]any   `json:"content,omitempty"`
	StarterID string           `json:"starter_id,omitempty"`
}

type UpdateTemplateParams struct {
	Name      Nullable[string] `json:"name,omitzero"`
	From      Nullable[string] `json:"from,omitzero"`
	ReplyTo   Nullable[string] `json:"reply_to,omitzero"`
	Subject   Nullable[string] `json:"subject,omitzero"`
	Preheader Nullable[string] `json:"preheader,omitzero"`
	HTML      Nullable[string] `json:"html,omitzero"`
	Content   map[string]any   `json:"content,omitempty"`
}

type ListTemplatesParams struct {
	Query     string
	Sort      string // "updated" or "name"
	Direction string // "asc" or "desc"
	After     string
	Before    string
	PerPage   int
}

type TemplateDetail struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	From                  string    `json:"from"`
	ReplyTo               *string   `json:"reply_to"`
	Subject               string    `json:"subject"`
	Preheader             string    `json:"preheader"`
	HTML                  string    `json:"html"`
	Text                  string    `json:"text"`
	PublishedVersionID    *string   `json:"published_version_id"`
	HasUnpublishedChanges bool      `json:"has_unpublished_changes"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type TemplateStats struct {
	Sent    int64 `json:"sent"`
	Opened  int64 `json:"opened"`
	Clicked int64 `json:"clicked"`
}

type UsedCampaign struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UsedIn struct {
	Count int           `json:"count"`
	First *UsedCampaign `json:"first"`
}

type TemplateListItem struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	From      string        `json:"from"`
	Subject   string        `json:"subject"`
	Preheader string        `json:"preheader"`
	Stats     TemplateStats `json:"stats"`
	UsedIn    UsedIn        `json:"used_in"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type TemplatePage struct {
	Data       []TemplateListItem `json:"data"`
	NextCursor *string            `json:"next_cursor"`
	PrevCursor *string            `json:"prev_cursor"`
}

type TemplateService struct {
	client *Client
}

func (s *TemplateService) List(ctx context.Context, params ListTemplatesParams, opts ...RequestOption) (*TemplatePage, error) {
	query := url.Values{}
	setQuery(query, "query", params.Query)
	setQuery(query, "sort", params.Sort)
	setQuery(query, "direction", params.Direction)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[TemplatePage](ctx, s.client, http.MethodGet, withQuery("/v1/templates", query), nil, opts)
}

func (s *TemplateService) Create(ctx context.Context, params CreateTemplateParams, opts ...RequestOption) (*TemplateDetail, error) {
	return call[TemplateDetail](ctx, s.client, http.MethodPost, "/v1/templates", params, opts)
}

func (s *TemplateService) Get(ctx context.Context, templateID string, opts ...RequestOption) (*TemplateDetail, error) {
	return call[TemplateDetail](ctx, s.client, http.MethodGet, buildPath("templates", templateID), nil, opts)
}

func (s *TemplateService) Update(ctx context.Context, templateID string, params UpdateTemplateParams, opts ...RequestOption) (*TemplateDetail, error) {
	return call[TemplateDetail](ctx, s.client, http.MethodPatch, buildPath("templates", templateID), params, opts)
}

func (s *TemplateService) Delete(ctx context.Context, templateID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("templates", templateID), nil, opts)
}

func (s *TemplateService) Publish(ctx context.Context, templateID string, opts ...RequestOption) (*TemplateDetail, error) {
	return call[TemplateDetail](ctx, s.client, http.MethodPost, buildPath("templates", templateID, "publish"), nil, opts)
}

func (s *TemplateService) ListVersions(ctx context.Context, templateID string, params ListVersionsParams, opts ...RequestOption) (*DocumentVersionPage, error) {
	path := withQuery(buildPath("templates", templateID, "versions"), buildVersionsQuery(params))

	return call[DocumentVersionPage](ctx, s.client, http.MethodGet, path, nil, opts)
}

func (s *TemplateService) CreateVersion(ctx context.Context, templateID string, params CreateVersionParams, opts ...RequestOption) (*DocumentVersion, error) {
	return call[DocumentVersion](ctx, s.client, http.MethodPost, buildPath("templates", templateID, "versions"), params, opts)
}

func (s *TemplateService) GetVersion(ctx context.Context, templateID, versionID string, opts ...RequestOption) (*DocumentVersionDetail, error) {
	path := buildPath("templates", templateID, "versions", versionID)

	return call[DocumentVersionDetail](ctx, s.client, http.MethodGet, path, nil, opts)
}

func (s *TemplateService) UpdateVersion(ctx context.Context, templateID, versionID string, params UpdateVersionParams, opts ...RequestOption) (*DocumentVersion, error) {
	path := buildPath("templates", templateID, "versions", versionID)

	return call[DocumentVersion](ctx, s.client, http.MethodPatch, path, params, opts)
}

func (s *TemplateService) RestoreVersion(ctx context.Context, templateID, versionID string, opts ...RequestOption) (*TemplateDetail, error) {
	path := buildPath("templates", templateID, "versions", versionID, "restore")

	return call[TemplateDetail](ctx, s.client, http.MethodPost, path, nil, opts)
}
