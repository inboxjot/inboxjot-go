package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/inboxjot/inboxjot-go"
)

const Tolerance = 5 * time.Minute

var (
	ErrMissingHeaders   = errors.New("webhook: missing webhook-id, webhook-timestamp or webhook-signature header")
	ErrInvalidTimestamp = errors.New("webhook: timestamp is invalid or too far from now")
	ErrInvalidSignature = errors.New("webhook: no matching signature")
)

type EventType = inboxjot.WebhookEventType

const (
	EmailSent            = inboxjot.WebhookEmailSent
	EmailDelivered       = inboxjot.WebhookEmailDelivered
	EmailOpened          = inboxjot.WebhookEmailOpened
	EmailClicked         = inboxjot.WebhookEmailClicked
	EmailBounced         = inboxjot.WebhookEmailBounced
	EmailComplained      = inboxjot.WebhookEmailComplained
	EmailDeliveryDelayed = inboxjot.WebhookEmailDeliveryDelayed
	EmailRejected        = inboxjot.WebhookEmailRejected
	EmailFailed          = inboxjot.WebhookEmailFailed
	EmailUnsubscribed    = inboxjot.WebhookEmailUnsubscribed
	ContactCreated       = inboxjot.WebhookContactCreated
	ContactUpdated       = inboxjot.WebhookContactUpdated
	ContactDeleted       = inboxjot.WebhookContactDeleted
)

type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Data      json.RawMessage `json:"data"`

	Email   *EmailData   `json:"-"`
	Contact *ContactData `json:"-"`
}

type EmailData struct {
	EmailID   string         `json:"email_id"`
	From      string         `json:"from"`
	To        []string       `json:"to"`
	Subject   string         `json:"subject"`
	Tags      []inboxjot.Tag `json:"tags"`
	Actor     EmailActor     `json:"actor"`
	Link      string         `json:"link"`
	Bounce    *Bounce        `json:"bounce"`
	Complaint *Complaint     `json:"complaint"`
	Reason    string         `json:"reason"`
}

type EmailActor string

const (
	ActorHuman    EmailActor = "human"
	ActorAppleMPP EmailActor = "apple_mpp"
	ActorScanner  EmailActor = "scanner"
	ActorUnknown  EmailActor = "unknown"
)

type BounceType string

const (
	BouncePermanent    BounceType = "permanent"
	BounceTransient    BounceType = "transient"
	BounceUndetermined BounceType = "undetermined"
)

type ComplaintType string

const ComplaintAbuse ComplaintType = "abuse"

type Bounce struct {
	Type    BounceType `json:"type"`
	Message string     `json:"message"`
}

type Complaint struct {
	Type ComplaintType `json:"type"`
}

type ContactData struct {
	ContactID  string                 `json:"contact_id"`
	Email      string                 `json:"email"`
	FirstName  *string                `json:"first_name"`
	LastName   *string                `json:"last_name"`
	Status     inboxjot.ContactStatus `json:"status"`
	Language   string                 `json:"language"`
	Properties map[string]any         `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type rawEvent Event

	err := json.Unmarshal(data, (*rawEvent)(e))
	if err != nil {
		return err
	}

	eventType := string(e.Type)
	switch {
	case strings.HasPrefix(eventType, "email."):
		e.Email = &EmailData{}
		return json.Unmarshal(e.Data, e.Email)
	case strings.HasPrefix(eventType, "contact."):
		e.Contact = &ContactData{}
		return json.Unmarshal(e.Data, e.Contact)
	}

	return nil
}

func Verify(payload []byte, headers http.Header, secret string) (*Event, error) {
	return verify(payload, headers, secret, time.Now())
}

func verify(payload []byte, headers http.Header, secret string, now time.Time) (*Event, error) {
	id := headers.Get("webhook-id")
	timestamp := headers.Get("webhook-timestamp")
	signatures := headers.Get("webhook-signature")
	if id == "" || timestamp == "" || signatures == "" {
		return nil, ErrMissingHeaders
	}

	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, ErrInvalidTimestamp
	}

	tolerance := int64(Tolerance / time.Second)
	tooOld := seconds < now.Unix()-tolerance
	tooNew := seconds > now.Unix()+tolerance
	if tooOld || tooNew {
		return nil, ErrInvalidTimestamp
	}

	expected, err := sign(secret, id, seconds, payload)
	if err != nil {
		return nil, err
	}
	if !hasSignature(signatures, expected) {
		return nil, ErrInvalidSignature
	}

	event := &Event{}
	err = json.Unmarshal(payload, event)
	if err != nil {
		return nil, fmt.Errorf("webhook: decode event: %w", err)
	}

	return event, nil
}

func Sign(secret, id string, timestamp time.Time, payload []byte) (string, error) {
	signature, err := sign(secret, id, timestamp.Unix(), payload)
	if err != nil {
		return "", err
	}

	return "v1," + signature, nil
}

func sign(secret, id string, timestamp int64, payload []byte) (string, error) {
	encodedKey := strings.TrimPrefix(secret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return "", fmt.Errorf("webhook: invalid secret: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "%s.%d.", id, timestamp)
	mac.Write(payload)

	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func hasSignature(header, expected string) bool {
	for _, entry := range strings.Fields(header) {
		version, signature, ok := strings.Cut(entry, ",")
		if !ok || version != "v1" {
			continue
		}
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return true
		}
	}

	return false
}
