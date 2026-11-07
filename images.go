package inboxjot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ImageType string

const (
	ImageTypePNG  ImageType = "png"
	ImageTypeJPEG ImageType = "jpeg"
	ImageTypeGIF  ImageType = "gif"
)

type UploadImageParams struct {
	File     []byte
	FileName string
}

type ListImagesParams struct {
	Query     string
	Type      ImageType
	Direction string // "asc" or "desc"
	After     string
	Before    string
	PerPage   int
}

type Image struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	URL         *string   `json:"url"`
	ContentType string    `json:"content_type"`
	SizeBytes   int       `json:"size_bytes"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	CreatorName *string   `json:"creator_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type ImagePage struct {
	Data       []Image `json:"data"`
	Total      int64   `json:"total"`
	NextCursor *string `json:"next_cursor"`
	PrevCursor *string `json:"prev_cursor"`
}

type imageUploadTarget struct {
	UploadURL string `json:"upload_url"`
	Key       string `json:"key"`
}

type ImageService struct {
	client *Client
}

func (s *ImageService) List(ctx context.Context, params ListImagesParams, opts ...RequestOption) (*ImagePage, error) {
	query := url.Values{}
	setQuery(query, "query", params.Query)
	setQuery(query, "type", string(params.Type))
	setQuery(query, "direction", params.Direction)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[ImagePage](ctx, s.client, http.MethodGet, withQuery("/v1/images", query), nil, opts)
}

func (s *ImageService) Upload(ctx context.Context, params UploadImageParams, opts ...RequestOption) (*Image, error) {
	var config requestConfig
	for _, opt := range opts {
		opt(&config)
	}
	if config.idempotencyKey != "" {
		return nil, errors.New("inboxjot: image uploads do not support an idempotency key")
	}

	contentType, err := getImageContentType(params.File)
	if err != nil {
		return nil, err
	}

	presign := struct {
		ContentType string `json:"content_type"`
		Size        int    `json:"size"`
	}{contentType, len(params.File)}
	target, err := call[imageUploadTarget](ctx, s.client, http.MethodPost, "/v1/images/presign", presign, opts)
	if err != nil {
		return nil, err
	}

	err = s.client.put(ctx, target.UploadURL, params.File, contentType)
	if err != nil {
		return nil, err
	}

	create := struct {
		Key      string `json:"key"`
		FileName string `json:"file_name,omitempty"`
	}{target.Key, params.FileName}

	return call[Image](ctx, s.client, http.MethodPost, "/v1/images", create, opts)
}

func (s *ImageService) Get(ctx context.Context, imageID string, opts ...RequestOption) (*Image, error) {
	return call[Image](ctx, s.client, http.MethodGet, buildPath("images", imageID), nil, opts)
}

func (s *ImageService) Delete(ctx context.Context, imageID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("images", imageID), nil, opts)
}

func getImageContentType(file []byte) (string, error) {
	switch {
	case bytes.HasPrefix(file, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png", nil
	case bytes.HasPrefix(file, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", nil
	case bytes.HasPrefix(file, []byte{0x47, 0x49, 0x46, 0x38}):
		return "image/gif", nil
	}

	return "", errors.New("inboxjot: the image must be a PNG, JPEG or GIF file")
}

func (c *Client) put(ctx context.Context, uploadURL string, file []byte, contentType string) error {
	attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPut, uploadURL, bytes.NewReader(file))
	if err != nil {
		return fmt.Errorf("inboxjot: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return wrapConnectionError(ctx, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return wrapConnectionError(ctx, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := readError(resp, data)
		apiErr.Message = fmt.Sprintf("the image upload failed with status %d", resp.StatusCode)

		return apiErr
	}

	return nil
}
