package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type EmailStatus string

const (
	EmailScheduled   EmailStatus = "scheduled"
	EmailQueued      EmailStatus = "queued"
	EmailSending     EmailStatus = "sending"
	EmailSent        EmailStatus = "sent"
	EmailSoftBounced EmailStatus = "soft_bounced"
	EmailDelivered   EmailStatus = "delivered"
	EmailOpened      EmailStatus = "opened"
	EmailClicked     EmailStatus = "clicked"
	EmailBounced     EmailStatus = "bounced"
	EmailRejected    EmailStatus = "rejected"
	EmailFailed      EmailStatus = "failed"
	EmailCanceled    EmailStatus = "canceled"
	EmailComplained  EmailStatus = "complained"
)

type EmailSource string

const (
	SourceAPI        EmailSource = "api"
	SourceSMTP       EmailSource = "smtp"
	SourceDashboard  EmailSource = "dashboard"
	SourceMCP        EmailSource = "mcp"
	SourceCampaign   EmailSource = "campaign"
	SourceAutomation EmailSource = "automation"
)

type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Template struct {
	ID        string         `json:"id"`
	Variables map[string]any `json:"variables,omitempty"`
}

type AttachmentParams struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content,omitempty"`
	Path        string `json:"path,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

type SendEmailParams struct {
	From        string             `json:"from,omitempty"`
	To          []string           `json:"to"`
	Cc          []string           `json:"cc,omitempty"`
	Bcc         []string           `json:"bcc,omitempty"`
	ReplyTo     []string           `json:"reply_to,omitempty"`
	Subject     string             `json:"subject,omitempty"`
	Template    *Template          `json:"template,omitempty"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	Tags        []Tag              `json:"tags,omitempty"`
	ScheduledAt time.Time          `json:"scheduled_at,omitzero"`
	Attachments []AttachmentParams `json:"attachments,omitempty"`
}

type BatchEmailParams struct {
	From     string            `json:"from,omitempty"`
	To       []string          `json:"to"`
	Cc       []string          `json:"cc,omitempty"`
	Bcc      []string          `json:"bcc,omitempty"`
	ReplyTo  []string          `json:"reply_to,omitempty"`
	Subject  string            `json:"subject,omitempty"`
	Template *Template         `json:"template,omitempty"`
	HTML     string            `json:"html,omitempty"`
	Text     string            `json:"text,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Tags     []Tag             `json:"tags,omitempty"`
}

type SendEmailResponse struct {
	ID string `json:"id"`
}

type BatchResponse struct {
	Data []SendEmailResponse `json:"data"`
}

type Email struct {
	ID          string      `json:"id"`
	From        string      `json:"from"`
	To          []string    `json:"to"`
	Cc          []string    `json:"cc"`
	Bcc         []string    `json:"bcc"`
	Subject     string      `json:"subject"`
	Tags        []Tag       `json:"tags"`
	Status      EmailStatus `json:"status"`
	Source      EmailSource `json:"source"`
	CreatedAt   time.Time   `json:"created_at"`
	ScheduledAt *time.Time  `json:"scheduled_at"`
	SentAt      *time.Time  `json:"sent_at"`
}

type ListEmailsParams struct {
	Query      string
	Status     []EmailStatus
	StatusOp   string // "is" or "not"
	Source     EmailSource
	SentAfter  time.Time
	SentBefore time.Time
	Sort       string // "sent" or "to"
	Direction  string // "asc" or "desc"
	After      string
	Before     string
	PerPage    int
}

type EmailPage struct {
	Data       []Email `json:"data"`
	NextCursor *string `json:"next_cursor"`
	PrevCursor *string `json:"prev_cursor"`
}

type UpdateEmailParams struct {
	ScheduledAt time.Time `json:"scheduled_at"`
}

type RescheduledEmail struct {
	ID          string     `json:"id"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

type CanceledEmail struct {
	ID     string      `json:"id"`
	Status EmailStatus `json:"status"`
}

type Attachment struct {
	ID                 string     `json:"id"`
	Filename           string     `json:"filename"`
	ContentType        string     `json:"content_type"`
	Size               int        `json:"size"`
	ContentDisposition string     `json:"content_disposition"`
	ContentID          *string    `json:"content_id"`
	DeletedAt          *time.Time `json:"deleted_at"`
	DownloadURL        *string    `json:"download_url"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

type AttachmentList struct {
	Data []Attachment `json:"data"`
}

type EmailService struct {
	client *Client
}

func (s *EmailService) Send(ctx context.Context, params SendEmailParams, opts ...RequestOption) (*SendEmailResponse, error) {
	return call[SendEmailResponse](ctx, s.client, http.MethodPost, "/v1/emails", params, opts)
}

func (s *EmailService) Batch(ctx context.Context, emails []BatchEmailParams, opts ...RequestOption) (*BatchResponse, error) {
	return call[BatchResponse](ctx, s.client, http.MethodPost, "/v1/emails/batch", emails, opts)
}

func (s *EmailService) List(ctx context.Context, params ListEmailsParams, opts ...RequestOption) (*EmailPage, error) {
	query := url.Values{}
	setQuery(query, "query", params.Query)
	setStatuses(query, params.Status)
	setQuery(query, "status_op", params.StatusOp)
	setQuery(query, "source", string(params.Source))
	if !params.SentAfter.IsZero() {
		query.Set("sent_after", params.SentAfter.Format(time.RFC3339))
	}
	if !params.SentBefore.IsZero() {
		query.Set("sent_before", params.SentBefore.Format(time.RFC3339))
	}
	setQuery(query, "sort", params.Sort)
	setQuery(query, "direction", params.Direction)
	setPage(query, params.After, params.Before, params.PerPage)

	return call[EmailPage](ctx, s.client, http.MethodGet, withQuery("/v1/emails", query), nil, opts)
}

func (s *EmailService) Get(ctx context.Context, emailID string, opts ...RequestOption) (*Email, error) {
	return call[Email](ctx, s.client, http.MethodGet, buildPath("emails", emailID), nil, opts)
}

func (s *EmailService) Update(ctx context.Context, emailID string, params UpdateEmailParams, opts ...RequestOption) (*RescheduledEmail, error) {
	return call[RescheduledEmail](ctx, s.client, http.MethodPatch, buildPath("emails", emailID), params, opts)
}

func (s *EmailService) Cancel(ctx context.Context, emailID string, opts ...RequestOption) (*CanceledEmail, error) {
	return call[CanceledEmail](ctx, s.client, http.MethodPost, buildPath("emails", emailID, "cancel"), nil, opts)
}

func (s *EmailService) ListAttachments(ctx context.Context, emailID string, opts ...RequestOption) (*AttachmentList, error) {
	return call[AttachmentList](ctx, s.client, http.MethodGet, buildPath("emails", emailID, "attachments"), nil, opts)
}

func (s *EmailService) GetAttachment(ctx context.Context, emailID, attachmentID string, opts ...RequestOption) (*Attachment, error) {
	path := buildPath("emails", emailID, "attachments", attachmentID)

	return call[Attachment](ctx, s.client, http.MethodGet, path, nil, opts)
}
