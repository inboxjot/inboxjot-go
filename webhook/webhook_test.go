package webhook

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	specSecret    = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	specID        = "msg_p5jXN8AQM9LWM0D4loKWxJek"
	specTimestamp = 1614265330
	specPayload   = `{"test": 2432232314}`
	specSignature = "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="
)

func makeHeaders(id string, timestamp int64, signature string) http.Header {
	headers := http.Header{}
	headers.Set("webhook-id", id)
	headers.Set("webhook-timestamp", strconv.FormatInt(timestamp, 10))
	headers.Set("webhook-signature", signature)

	return headers
}

func TestSignsLikeTheStandardWebhooksSpec(t *testing.T) {
	now := time.Unix(specTimestamp, 0)

	signature, err := Sign(specSecret, specID, now, []byte(specPayload))
	if err != nil || signature != specSignature {
		t.Fatalf("Sign = %q, %v", signature, err)
	}

	headers := makeHeaders(specID, specTimestamp, "v1,bm90IGl0 "+specSignature)
	_, err = verify([]byte(specPayload), headers, specSecret, now)
	if err != nil {
		t.Fatalf("verify = %v", err)
	}
}

func TestSignsWithASecretMadeBeforeTheSwitch(t *testing.T) {
	secret := "whsec_3kTmPqZ8vXw2RbN7cYhJ4dLsF9gA6eUu"

	signature, err := Sign(secret, "evt_1", time.Unix(1, 0), []byte("{}"))
	if err != nil || signature[:3] != "v1," {
		t.Fatalf("Sign = %q, %v", signature, err)
	}
}

func TestVerifiesAndDecodesEvents(t *testing.T) {
	secret := "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	now := time.Now()

	cases := []struct {
		payload string
		check   func(*Event) bool
	}{
		{
			`{"id":"evt_1","type":"email.bounced","created_at":"2026-10-10T09:00:00.000Z",` +
				`"data":{"email_id":"em_1","to":["jane@acme.dev"],"from":"hi@acme.dev","subject":"Hi",` +
				`"tags":[{"name":"kind","value":"receipt"}],"bounce":{"type":"permanent","message":"No mailbox"}}}`,
			func(e *Event) bool {
				return e.Type == EmailBounced &&
					e.Email.Bounce.Type == BouncePermanent &&
					e.Email.Complaint == nil &&
					e.Email.Tags[0].Value == "receipt" &&
					e.Contact == nil
			},
		},
		{
			`{"id":"evt_3","type":"email.clicked","created_at":"2026-10-10T09:00:00.000Z",` +
				`"data":{"email_id":"em_1","to":["jane@acme.dev"],"from":"hi@acme.dev","subject":"Hi",` +
				`"actor":"apple_mpp","link":"https://acme.dev"}}`,
			func(e *Event) bool {
				return e.Type == EmailClicked &&
					e.Email.Actor == ActorAppleMPP &&
					e.Email.Link == "https://acme.dev" &&
					e.Email.Bounce == nil
			},
		},
		{
			`{"id":"evt_4","type":"email.complained","created_at":"2026-10-10T09:00:00.000Z",` +
				`"data":{"email_id":"em_1","to":["jane@acme.dev"],"from":"hi@acme.dev","subject":"Hi",` +
				`"complaint":{"type":"abuse"}}}`,
			func(e *Event) bool {
				return e.Type == EmailComplained &&
					e.Email.Complaint.Type == ComplaintAbuse
			},
		},
		{
			`{"id":"evt_2","type":"contact.updated","created_at":"2026-10-10T09:00:00.000Z",` +
				`"data":{"contact_id":"ct_1","email":"jane@acme.dev","first_name":null,"last_name":"Doe",` +
				`"status":"subscribed","language":"en","properties":{"plan":"pro"},` +
				`"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}}`,
			func(e *Event) bool {
				return e.Type == ContactUpdated &&
					e.Contact.FirstName == nil &&
					e.Contact.Properties["plan"] == "pro" &&
					e.Email == nil
			},
		},
	}

	for _, tc := range cases {
		payload := []byte(tc.payload)
		signature, _ := Sign(secret, "evt_1", now, payload)
		headers := makeHeaders("evt_1", now.Unix(), signature)

		event, err := Verify(payload, headers, secret)
		if err != nil || !tc.check(event) {
			t.Fatalf("Verify = %+v, %v", event, err)
		}
	}
}

func TestRejectsBadRequests(t *testing.T) {
	now := time.Unix(specTimestamp, 0)
	otherVersion := "v1a,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="

	cases := []struct {
		name    string
		headers http.Header
		payload string
		want    error
	}{
		{
			name:    "missing headers",
			headers: http.Header{},
			payload: specPayload,
			want:    ErrMissingHeaders,
		},
		{
			name:    "changed payload",
			headers: makeHeaders(specID, specTimestamp, specSignature),
			payload: `{"test": 1}`,
			want:    ErrInvalidSignature,
		},
		{
			name:    "changed id",
			headers: makeHeaders("msg_other", specTimestamp, specSignature),
			payload: specPayload,
			want:    ErrInvalidSignature,
		},
		{
			name:    "other version",
			headers: makeHeaders(specID, specTimestamp, otherVersion),
			payload: specPayload,
			want:    ErrInvalidSignature,
		},
		{
			name:    "old timestamp",
			headers: makeHeaders(specID, specTimestamp-301, specSignature),
			payload: specPayload,
			want:    ErrInvalidTimestamp,
		},
		{
			name:    "future timestamp",
			headers: makeHeaders(specID, specTimestamp+301, specSignature),
			payload: specPayload,
			want:    ErrInvalidTimestamp,
		},
		{
			name:    "largest timestamp",
			headers: makeHeaders(specID, math.MaxInt64, specSignature),
			payload: specPayload,
			want:    ErrInvalidTimestamp,
		},
		{
			name:    "smallest timestamp",
			headers: makeHeaders(specID, math.MinInt64, specSignature),
			payload: specPayload,
			want:    ErrInvalidTimestamp,
		},
	}

	for _, tc := range cases {
		_, err := verify([]byte(tc.payload), tc.headers, specSecret, now)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestDecodesUnknownValues(t *testing.T) {
	payload := []byte(`{"id":"evt_5","type":"email.new_thing","created_at":"2026-10-10T09:00:00.000Z",` +
		`"data":{"email_id":"em_1","to":["jane@acme.dev"],"from":"hi@acme.dev","subject":"Hi","actor":"robot",` +
		`"bounce":{"type":"strange","message":"?"}}}`)
	now := time.Now()
	signature, _ := Sign(specSecret, "evt_5", now, payload)

	event, err := Verify(payload, makeHeaders("evt_5", now.Unix(), signature), specSecret)
	if err != nil {
		t.Fatal(err)
	}
	if event.Email.Actor != "robot" || event.Email.Bounce.Type != "strange" {
		t.Fatalf("email = %+v", event.Email)
	}

	payload = []byte(`{"id":"evt_6","type":"domain.verified","created_at":"2026-10-10T09:00:00.000Z","data":{"id":"dom_1"}}`)
	signature, _ = Sign(specSecret, "evt_6", now, payload)

	event, err = Verify(payload, makeHeaders("evt_6", now.Unix(), signature), specSecret)
	if err != nil {
		t.Fatal(err)
	}
	if event.Email != nil || event.Contact != nil || string(event.Data) != `{"id":"dom_1"}` {
		t.Fatalf("event = %+v", event)
	}
}
