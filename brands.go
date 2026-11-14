package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type CreateBrandParams struct {
	Name string `json:"name"`
}

type UpdateBrandParams struct {
	Name string `json:"name"`
}

type Brand struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	LogoURL        *string   `json:"logo_url"`
	IsDefault      bool      `json:"is_default"`
	DefaultTopicID *string   `json:"default_topic_id"`
	TopicCount     int64     `json:"topic_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BrandPage struct {
	Data       []Brand `json:"data"`
	NextCursor *string `json:"next_cursor"`
	PrevCursor *string `json:"prev_cursor"`
}

type BrandService struct {
	client *Client
}

func (s *BrandService) List(ctx context.Context, params PageParams, opts ...RequestOption) (*BrandPage, error) {
	query := url.Values{}
	setPage(query, params.After, params.Before, params.PerPage)

	return call[BrandPage](ctx, s.client, http.MethodGet, withQuery("/v1/brands", query), nil, opts)
}

func (s *BrandService) Create(ctx context.Context, params CreateBrandParams, opts ...RequestOption) (*Brand, error) {
	return call[Brand](ctx, s.client, http.MethodPost, "/v1/brands", params, opts)
}

func (s *BrandService) Get(ctx context.Context, brandID string, opts ...RequestOption) (*Brand, error) {
	return call[Brand](ctx, s.client, http.MethodGet, buildPath("brands", brandID), nil, opts)
}

func (s *BrandService) Update(ctx context.Context, brandID string, params UpdateBrandParams, opts ...RequestOption) (*Brand, error) {
	return call[Brand](ctx, s.client, http.MethodPatch, buildPath("brands", brandID), params, opts)
}

func (s *BrandService) Delete(ctx context.Context, brandID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("brands", brandID), nil, opts)
}
