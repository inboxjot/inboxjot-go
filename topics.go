package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type TopicVisibility string

const (
	TopicPublic  TopicVisibility = "public"
	TopicPrivate TopicVisibility = "private"
)

type CreateTopicParams struct {
	Name              string           `json:"name"`
	Description       Nullable[string] `json:"description,omitzero"`
	DefaultSubscribed *bool            `json:"default_subscribed,omitempty"`
	Visibility        TopicVisibility  `json:"visibility,omitempty"`
	DoubleOptIn       *bool            `json:"double_opt_in,omitempty"`
	FormEnabled       *bool            `json:"form_enabled,omitempty"`
	SenderName        Nullable[string] `json:"sender_name,omitzero"`
	SenderEmail       Nullable[string] `json:"sender_email,omitzero"`
	RedirectURL       Nullable[string] `json:"redirect_url,omitzero"`
}

type UpdateTopicParams struct {
	Name              Nullable[string] `json:"name,omitzero"`
	Description       Nullable[string] `json:"description,omitzero"`
	DefaultSubscribed *bool            `json:"default_subscribed,omitempty"`
	Visibility        TopicVisibility  `json:"visibility,omitempty"`
	DoubleOptIn       *bool            `json:"double_opt_in,omitempty"`
	FormEnabled       *bool            `json:"form_enabled,omitempty"`
	SenderName        Nullable[string] `json:"sender_name,omitzero"`
	SenderEmail       Nullable[string] `json:"sender_email,omitzero"`
	RedirectURL       Nullable[string] `json:"redirect_url,omitzero"`
}

type TopicCounts struct {
	Subscribed   int `json:"subscribed"`
	Pending      int `json:"pending"`
	Unsubscribed int `json:"unsubscribed"`
}

type Topic struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description"`
	DefaultSubscribed bool            `json:"default_subscribed"`
	Visibility        TopicVisibility `json:"visibility"`
	DoubleOptIn       bool            `json:"double_opt_in"`
	FormEnabled       bool            `json:"form_enabled"`
	SenderName        *string         `json:"sender_name"`
	SenderEmail       *string         `json:"sender_email"`
	RedirectURL       *string         `json:"redirect_url"`
	IsDefault         bool            `json:"is_default"`
	Counts            TopicCounts     `json:"counts"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type TopicPage struct {
	Data       []Topic `json:"data"`
	NextCursor *string `json:"next_cursor"`
	PrevCursor *string `json:"prev_cursor"`
}

type TopicService struct {
	client *Client
}

func (s *TopicService) List(ctx context.Context, params PageParams, opts ...RequestOption) (*TopicPage, error) {
	query := url.Values{}
	setPage(query, params.After, params.Before, params.PerPage)

	return call[TopicPage](ctx, s.client, http.MethodGet, withQuery("/v1/topics", query), nil, opts)
}

func (s *TopicService) Create(ctx context.Context, params CreateTopicParams, opts ...RequestOption) (*Topic, error) {
	return call[Topic](ctx, s.client, http.MethodPost, "/v1/topics", params, opts)
}

func (s *TopicService) Get(ctx context.Context, topicID string, opts ...RequestOption) (*Topic, error) {
	return call[Topic](ctx, s.client, http.MethodGet, buildPath("topics", topicID), nil, opts)
}

func (s *TopicService) Update(ctx context.Context, topicID string, params UpdateTopicParams, opts ...RequestOption) (*Topic, error) {
	return call[Topic](ctx, s.client, http.MethodPatch, buildPath("topics", topicID), params, opts)
}

func (s *TopicService) Delete(ctx context.Context, topicID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("topics", topicID), nil, opts)
}
