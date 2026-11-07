package inboxjot

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type DomainStatus string

const (
	DomainNotStarted       DomainStatus = "not_started"
	DomainPending          DomainStatus = "pending"
	DomainVerified         DomainStatus = "verified"
	DomainFailed           DomainStatus = "failed"
	DomainTemporaryFailure DomainStatus = "temporary_failure"
)

type RecordKind string

const (
	RecordDKIM          RecordKind = "dkim"
	RecordReturnPath    RecordKind = "return_path"
	RecordDMARC         RecordKind = "dmarc"
	RecordTrackingCNAME RecordKind = "tracking_cname"
	RecordTrackingCAA   RecordKind = "tracking_caa"
)

type TrackingStatus string

const (
	TrackingNotStarted TrackingStatus = "not_started"
	TrackingPending    TrackingStatus = "pending"
	TrackingIssuing    TrackingStatus = "issuing"
	TrackingVerified   TrackingStatus = "verified"
	TrackingFailed     TrackingStatus = "failed"
)

type CreateDomainParams struct {
	Name       string `json:"name"`
	Region     string `json:"region,omitempty"`
	ReturnPath string `json:"return_path,omitempty"`
}

type DomainRecord struct {
	Kind     RecordKind   `json:"kind"`
	Type     string       `json:"type"`
	Name     string       `json:"name"`
	Value    string       `json:"value"`
	Priority *int         `json:"priority"`
	Status   DomainStatus `json:"status"`
}

type DomainReceiving struct {
	Host          string  `json:"host"`
	MX            string  `json:"mx"`
	SharedAddress *string `json:"shared_address"`
}

type DomainWarmup struct {
	Enabled      bool    `json:"enabled"`
	StartedOn    *string `json:"started_on"`
	Day          *int64  `json:"day"`
	Limit        *int64  `json:"limit"`
	NextLimit    *int64  `json:"next_limit"`
	SentToday    int64   `json:"sent_today"`
	FullVolumeOn *string `json:"full_volume_on"`
}

type Domain struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	MailFromDomain      string          `json:"mail_from_domain"`
	Status              DomainStatus    `json:"status"`
	Records             []DomainRecord  `json:"records"`
	Region              string          `json:"region"`
	TLSPolicy           string          `json:"tls_policy"`
	ReceivingEnabled    bool            `json:"receiving_enabled"`
	ReceivingTarget     string          `json:"receiving_target"`
	ReceivingStatus     DomainStatus    `json:"receiving_status"`
	ReceivingFoundMX    *string         `json:"receiving_found_mx"`
	ReceivingCheckedAt  *time.Time      `json:"receiving_checked_at"`
	ReceivingVerifiedAt *time.Time      `json:"receiving_verified_at"`
	OpenTracking        bool            `json:"open_tracking"`
	ClickTracking       bool            `json:"click_tracking"`
	TrackingDomain      *string         `json:"tracking_domain"`
	TrackingStatus      TrackingStatus  `json:"tracking_status"`
	TrackingRecords     []DomainRecord  `json:"tracking_records"`
	TrackingVerifiedAt  *time.Time      `json:"tracking_verified_at"`
	MailingAddress      *string         `json:"mailing_address"`
	VerifiedAt          *time.Time      `json:"verified_at"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	SentLast30Days      int             `json:"sent_last_30_days"`
	BouncedLast30Days   int             `json:"bounced_last_30_days"`
	IsDefault           bool            `json:"is_default"`
	Receiving           DomainReceiving `json:"receiving"`
	Warmup              DomainWarmup    `json:"warmup"`
}

type DomainListItem struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Status             DomainStatus `json:"status"`
	PendingRecordKinds []RecordKind `json:"pending_record_kinds"`
	Region             string       `json:"region"`
	IsDefault          bool         `json:"is_default"`
	OpenTracking       bool         `json:"open_tracking"`
	ClickTracking      bool         `json:"click_tracking"`
	SentLast30Days     int          `json:"sent_last_30_days"`
	VerifiedAt         *time.Time   `json:"verified_at"`
	CreatedAt          time.Time    `json:"created_at"`
}

type DomainPage struct {
	Data       []DomainListItem `json:"data"`
	NextCursor *string          `json:"next_cursor"`
	PrevCursor *string          `json:"prev_cursor"`
}

type DomainService struct {
	client *Client
}

func (s *DomainService) List(ctx context.Context, params PageParams, opts ...RequestOption) (*DomainPage, error) {
	query := url.Values{}
	setPage(query, params.After, params.Before, params.PerPage)

	return call[DomainPage](ctx, s.client, http.MethodGet, withQuery("/v1/domains", query), nil, opts)
}

func (s *DomainService) Create(ctx context.Context, params CreateDomainParams, opts ...RequestOption) (*Domain, error) {
	return call[Domain](ctx, s.client, http.MethodPost, "/v1/domains", params, opts)
}

func (s *DomainService) Get(ctx context.Context, domainID string, opts ...RequestOption) (*Domain, error) {
	return call[Domain](ctx, s.client, http.MethodGet, buildPath("domains", domainID), nil, opts)
}

func (s *DomainService) Delete(ctx context.Context, domainID string, opts ...RequestOption) (*StatusOK, error) {
	return call[StatusOK](ctx, s.client, http.MethodDelete, buildPath("domains", domainID), nil, opts)
}

func (s *DomainService) Verify(ctx context.Context, domainID string, opts ...RequestOption) (*Domain, error) {
	return call[Domain](ctx, s.client, http.MethodPost, buildPath("domains", domainID, "verify"), nil, opts)
}
