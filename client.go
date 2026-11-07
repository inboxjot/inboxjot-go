package inboxjot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.inboxjot.com"
	modulePath     = "github.com/inboxjot/inboxjot-go"
)

var Version = readVersion()

func readVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return strings.TrimPrefix(dep.Version, "v")
		}
	}

	return "dev"
}

type Client struct {
	Emails    *EmailService
	Contacts  *ContactService
	Events    *EventService
	Campaigns *CampaignService
	Templates *TemplateService
	Domains   *DomainService
	Images    *ImageService
	Segments  *SegmentService
	Topics    *TopicService
	Webhooks  *WebhookService

	// A pointer, so fmt prints an address and never the key.
	apiKey     *string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

func WithMaxRetries(retries int) Option {
	return func(c *Client) {
		c.maxRetries = max(retries, 0)
	}
}

func NewClient(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("INBOXJOT_API_KEY")
	}

	c := &Client{
		apiKey:     &apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
		timeout:    60 * time.Second,
		maxRetries: 2,
	}
	for _, opt := range opts {
		opt(c)
	}

	c.Emails = &EmailService{c}
	c.Contacts = &ContactService{c}
	c.Events = &EventService{c}
	c.Campaigns = &CampaignService{c}
	c.Templates = &TemplateService{c}
	c.Domains = &DomainService{c}
	c.Images = &ImageService{c}
	c.Segments = &SegmentService{c}
	c.Topics = &TopicService{c}
	c.Webhooks = &WebhookService{c}

	return c
}

type RequestOption func(*requestConfig)

type requestConfig struct {
	idempotencyKey string
}

func WithIdempotencyKey(key string) RequestOption {
	return func(config *requestConfig) {
		config.idempotencyKey = key
	}
}

type ErrorName string

const (
	ErrValidation                   ErrorName = "validation_error"
	ErrNotFound                     ErrorName = "not_found"
	ErrUnauthorized                 ErrorName = "unauthorized"
	ErrForbidden                    ErrorName = "forbidden"
	ErrInternalServer               ErrorName = "internal_server_error"
	ErrBadRequest                   ErrorName = "bad_request"
	ErrUserNotVerified              ErrorName = "user_not_verified"
	ErrRateLimitExceeded            ErrorName = "rate_limit_exceeded"
	ErrConflict                     ErrorName = "conflict"
	ErrLimitExceeded                ErrorName = "limit_exceeded"
	ErrReauthenticationRequired     ErrorName = "reauthentication_required"
	ErrInvalidIdempotencyKey        ErrorName = "invalid_idempotency_key"
	ErrConcurrentIdempotentRequests ErrorName = "concurrent_idempotent_requests"
	ErrInvalidIdempotentRequest     ErrorName = "invalid_idempotent_request"
	ErrPayloadTooLarge              ErrorName = "payload_too_large"
	ErrUnsupportedMediaType         ErrorName = "unsupported_media_type"
	ErrServiceUnavailable           ErrorName = "service_unavailable"
	ErrUnknown                      ErrorName = "unknown_error"
)

type ErrorItem struct {
	Message  string `json:"message"`
	Location string `json:"location"`
}

type Error struct {
	Name      ErrorName   `json:"name"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Errors    []ErrorItem `json:"errors"`
	RequestID string      `json:"request_id"`

	retryAfter string
}

func (e *Error) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("inboxjot: %d: %s", e.Status, e.Message)
	}

	return fmt.Sprintf("inboxjot: %s (%d): %s", e.Name, e.Status, e.Message)
}

type ConnectionError struct {
	Err error
}

func (e *ConnectionError) Error() string {
	return "inboxjot: connection error: " + e.Err.Error()
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

type StatusOK struct {
	Status string `json:"status"`
}

type PageParams struct {
	After   string
	Before  string
	PerPage int
}

type Nullable[T any] struct {
	value T
	state uint8
}

func Value[T any](value T) Nullable[T] {
	return Nullable[T]{value: value, state: 1}
}

func Null[T any]() Nullable[T] {
	return Nullable[T]{state: 2}
}

func (n Nullable[T]) IsZero() bool {
	return n.state == 0
}

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.state != 1 {
		return []byte("null"), nil
	}

	return json.Marshal(n.value)
}

func setQuery(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setStatuses[T ~string](query url.Values, statuses []T) {
	if len(statuses) == 0 {
		return
	}

	values := make([]string, len(statuses))
	for i, status := range statuses {
		values[i] = string(status)
	}
	query.Set("status", strings.Join(values, ","))
}

func setPage(query url.Values, after, before string, perPage int) {
	setQuery(query, "after", after)
	setQuery(query, "before", before)
	if perPage > 0 {
		query.Set("per_page", strconv.Itoa(perPage))
	}
}

func buildPath(segments ...string) string {
	var path strings.Builder
	path.WriteString("/v1")
	for _, segment := range segments {
		path.WriteString("/" + url.PathEscape(segment))
	}

	return path.String()
}

func withQuery(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}

	return path + "?" + query.Encode()
}

func call[T any](ctx context.Context, c *Client, method, path string, body any, opts []RequestOption) (*T, error) {
	out := new(T)

	err := c.do(ctx, method, path, body, out, opts)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any, opts []RequestOption) error {
	if *c.apiKey == "" {
		return errors.New("inboxjot: missing API key; pass one to NewClient or set INBOXJOT_API_KEY")
	}

	var payload []byte
	if body != nil {
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)

		err := encoder.Encode(body)
		if err != nil {
			return fmt.Errorf("inboxjot: encode request: %w", err)
		}

		payload = bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
	}

	var config requestConfig
	for _, opt := range opts {
		opt(&config)
	}
	isSafe := method == http.MethodGet || method == http.MethodDelete || config.idempotencyKey != ""

	for attempt := 0; ; attempt++ {
		err := c.send(ctx, method, path, payload, out, config)
		if err == nil || !isRetryable(err, isSafe) {
			return err
		}
		if attempt >= c.maxRetries || ctx.Err() != nil {
			return err
		}

		retryAfter := ""
		var apiErr *Error
		if errors.As(err, &apiErr) {
			retryAfter = apiErr.retryAfter
		}

		delay, ok := getRetryDelay(attempt, retryAfter)
		if !ok {
			return err
		}

		waitErr := wait(ctx, delay)
		if waitErr != nil {
			return waitErr
		}
	}
}

func (c *Client) send(ctx context.Context, method, path string, payload []byte, out any, config requestConfig) error {
	attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(attemptCtx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("inboxjot: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+*c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "inboxjot-go/"+Version)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if config.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", config.idempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return wrapConnectionError(ctx, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return wrapConnectionError(ctx, err)
	}

	if resp.StatusCode >= 400 {
		return readError(resp, data)
	}

	if out == nil {
		return nil
	}

	err = json.Unmarshal(data, out)
	if err != nil {
		return fmt.Errorf("inboxjot: decode response: %w", err)
	}

	return nil
}

func wrapConnectionError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return err
	}

	return &ConnectionError{Err: err}
}

func readError(resp *http.Response, data []byte) *Error {
	apiErr := &Error{}

	decodeErr := json.Unmarshal(data, apiErr)
	if decodeErr != nil || apiErr.Name == "" {
		apiErr = &Error{Message: strings.TrimSpace(string(data))}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
	}

	apiErr.Status = resp.StatusCode
	if apiErr.RequestID == "" {
		apiErr.RequestID = resp.Header.Get("X-Request-Id")
	}
	apiErr.retryAfter = resp.Header.Get("Retry-After")

	return apiErr
}

func isRetryable(err error, isSafe bool) bool {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		isRateLimited := apiErr.Status == http.StatusTooManyRequests
		isConcurrent := apiErr.Name == ErrConcurrentIdempotentRequests
		isServerError := apiErr.Status >= 500

		return isRateLimited || isConcurrent || (isServerError && isSafe)
	}

	var connectionErr *ConnectionError

	return errors.As(err, &connectionErr) && isSafe
}

func getRetryDelay(attempt int, retryAfter string) (time.Duration, bool) {
	seconds, err := strconv.ParseFloat(retryAfter, 64)
	if errors.Is(err, strconv.ErrRange) || seconds > 60 {
		return 0, false
	}
	if err == nil && seconds >= 0 {
		return time.Duration(seconds * float64(time.Second)), true
	}

	retryAt, err := http.ParseTime(retryAfter)
	if err == nil {
		delay := time.Until(retryAt)
		if delay > time.Minute {
			return 0, false
		}

		return max(delay, 0), true
	}

	delay := 500 * time.Millisecond << min(attempt, 4)
	jitter := time.Duration(mathrand.Int64N(int64(delay) / 4))

	return delay - jitter, true
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
