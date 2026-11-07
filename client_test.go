package inboxjot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type recorded struct {
	method string
	path   string
	query  string
	body   string
	header http.Header
}

func startServer(t *testing.T, handle func(attempt int, w http.ResponseWriter, r *http.Request)) (*Client, func() []recorded) {
	t.Helper()

	var mu sync.Mutex
	var requests []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		request := recorded{
			method: r.Method,
			path:   r.URL.EscapedPath(),
			query:  r.URL.RawQuery,
			body:   string(body),
			header: r.Header.Clone(),
		}

		mu.Lock()
		requests = append(requests, request)
		attempt := len(requests)
		mu.Unlock()

		handle(attempt, w, r)
	}))
	t.Cleanup(server.Close)

	list := func() []recorded {
		mu.Lock()
		defer mu.Unlock()

		return append([]recorded(nil), requests...)
	}

	return NewClient("ibx_test", WithBaseURL(server.URL)), list
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, body)
}

func TestSendsAuthAndIdempotencyHeaders(t *testing.T) {
	sentEmail := `{"id":"em_1","from":"a@acme.dev","to":[],"cc":[],"bcc":[],"subject":"Hi","tags":[],` +
		`"status":"sent","source":"api","created_at":"2026-10-10T09:00:00.000Z","scheduled_at":null,"sent_at":"2026-10-10T09:00:01.000Z"}`

	client, requests := startServer(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reply(w, 200, sentEmail)
			return
		}

		reply(w, 202, `{"id":"em_1"}`)
	})

	ctx := context.Background()
	params := SendEmailParams{
		From:    "Acme <hi@acme.dev>",
		To:      []string{"jane@acme.dev"},
		Subject: "Hi",
		Text:    "Hello",
	}

	_, err := client.Emails.Send(ctx, params, WithIdempotencyKey("order-1"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Emails.Send(ctx, params)
	if err != nil {
		t.Fatal(err)
	}

	email, err := client.Emails.Get(ctx, "em_1")
	if err != nil {
		t.Fatal(err)
	}
	if email.SentAt == nil || email.ScheduledAt != nil || email.CreatedAt.Year() != 2026 || email.Source != SourceAPI {
		t.Fatalf("decoded email = %+v", email)
	}

	got := requests()
	for _, r := range got {
		authorization := r.header.Get("Authorization")
		if authorization != "Bearer ibx_test" {
			t.Fatalf("Authorization = %q", authorization)
		}
	}

	explicitKey := got[0].header.Get("Idempotency-Key")
	if explicitKey != "order-1" {
		t.Fatalf("explicit key = %q", explicitKey)
	}

	omittedKey := got[1].header.Get("Idempotency-Key")
	if omittedKey != "" {
		t.Fatalf("sent a key without being asked: %q", omittedKey)
	}

	userAgent := got[0].header.Get("User-Agent")
	if userAgent != "inboxjot-go/"+Version {
		t.Fatalf("User-Agent = %q", userAgent)
	}

	getKey := got[2].header.Get("Idempotency-Key")
	if getKey != "" {
		t.Fatalf("GET sent a key %q", getKey)
	}

	wantBody := `{"from":"Acme <hi@acme.dev>","to":["jane@acme.dev"],"subject":"Hi","text":"Hello"}`
	if got[0].body != wantBody {
		t.Fatalf("body = %s", got[0].body)
	}
}

func TestReadsAPIKeyFromEnvironment(t *testing.T) {
	t.Setenv("INBOXJOT_API_KEY", "ibx_env")
	if *NewClient("").apiKey != "ibx_env" {
		t.Fatal("did not read INBOXJOT_API_KEY")
	}

	t.Setenv("INBOXJOT_API_KEY", "")
	_, err := NewClient("").Emails.Get(context.Background(), "em_1")
	if err == nil {
		t.Fatal("expected a missing key error")
	}
}

func TestRetries(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		body     string
		opts     []RequestOption
		attempts int
	}{
		{
			name:     "server error without a key",
			status:   500,
			body:     `{"name":"unknown_error","status":500,"message":"Boom","errors":[],"request_id":null}`,
			attempts: 1,
		},
		{
			name:     "server error with a key",
			status:   500,
			body:     `{"name":"unknown_error","status":500,"message":"Boom","errors":[],"request_id":null}`,
			opts:     []RequestOption{WithIdempotencyKey("event-1")},
			attempts: 2,
		},
		{
			name:     "rate limit without a key",
			status:   429,
			body:     `{"name":"rate_limit_exceeded","status":429,"message":"Slow down","errors":[],"request_id":null}`,
			attempts: 2,
		},
		{
			name:     "concurrent idempotent",
			status:   409,
			body:     `{"name":"concurrent_idempotent_requests","status":409,"message":"Running","errors":[],"request_id":null}`,
			opts:     []RequestOption{WithIdempotencyKey("event-1")},
			attempts: 2,
		},
		{
			name:     "conflict",
			status:   409,
			body:     `{"name":"conflict","status":409,"message":"Exists","errors":[],"request_id":null}`,
			attempts: 1,
		},
		{
			name:     "validation",
			status:   400,
			body:     `{"name":"validation_error","status":400,"message":"Bad","errors":[],"request_id":null}`,
			attempts: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, requests := startServer(t, func(attempt int, w http.ResponseWriter, _ *http.Request) {
				if attempt == 1 {
					w.Header().Set("Retry-After", "0")
					reply(w, tc.status, tc.body)
					return
				}

				reply(w, 202, `{"contact_id":"ct_1"}`)
			})

			params := SendEventParams{Name: "signed_up", Email: "jane@acme.dev"}
			_, err := client.Events.Send(context.Background(), params, tc.opts...)

			got := requests()
			if len(got) != tc.attempts {
				t.Fatalf("attempts = %d, want %d", len(got), tc.attempts)
			}

			shouldSucceed := tc.attempts == 2
			if (err == nil) != shouldSucceed {
				t.Fatalf("err = %v", err)
			}

			if !shouldSucceed {
				return
			}

			firstKey := got[0].header.Get("Idempotency-Key")
			retryKey := got[1].header.Get("Idempotency-Key")
			if firstKey != retryKey {
				t.Fatal("retry changed the idempotency key")
			}
		})
	}
}

func TestRetryAfterOverSixtySecondsReturnsTheError(t *testing.T) {
	httpDate := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)

	for _, retryAfter := range []string{"61", "86400", "99999999999999999999", httpDate} {
		client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", retryAfter)
			reply(w, 429, `{"name":"limit_exceeded","status":429,"message":"Daily limit","errors":[],"request_id":null}`)
		})

		_, err := client.Emails.Get(context.Background(), "em_1")

		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Name != ErrLimitExceeded {
			t.Fatalf("Retry-After %s: err = %v", retryAfter, err)
		}
		if attempts := len(requests()); attempts != 1 {
			t.Fatalf("Retry-After %s: attempts = %d, want 1", retryAfter, attempts)
		}
	}
}

func TestGetRetryDelay(t *testing.T) {
	delay, ok := getRetryDelay(0, "60")
	if !ok || delay != time.Minute {
		t.Fatalf("Retry-After 60 = %v, %v", delay, ok)
	}

	cases := []struct {
		attempt int
		low     time.Duration
		high    time.Duration
	}{
		{attempt: 0, low: 375 * time.Millisecond, high: 500 * time.Millisecond},
		{attempt: 2, low: 1500 * time.Millisecond, high: 2 * time.Second},
		{attempt: 4, low: 6 * time.Second, high: 8 * time.Second},
		{attempt: 100, low: 6 * time.Second, high: 8 * time.Second},
	}
	for _, tc := range cases {
		for _, retryAfter := range []string{"", "soon", "-1"} {
			delay, ok := getRetryDelay(tc.attempt, retryAfter)
			if !ok || delay < tc.low || delay > tc.high {
				t.Fatalf("attempt %d, Retry-After %q = %v, %v", tc.attempt, retryAfter, delay, ok)
			}
		}
	}
}

func TestStopsAfterMaxRetries(t *testing.T) {
	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		reply(w, 503, `{"name":"unknown_error","status":503,"message":"Down","errors":[],"request_id":"req_1"}`)
	})

	_, err := client.Emails.Get(context.Background(), "em_1")

	attempts := len(requests())
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.RequestID != "req_1" {
		t.Fatalf("err = %v", err)
	}
}

func TestHonorsRetryAfter(t *testing.T) {
	client, _ := startServer(t, func(attempt int, w http.ResponseWriter, _ *http.Request) {
		if attempt == 1 {
			w.Header().Set("Retry-After", "1")
			reply(w, 409, `{"name":"concurrent_idempotent_requests","status":409,"message":"Running","errors":[],"request_id":null}`)
			return
		}

		reply(w, 200, `{"id":"em_1","status":"canceled"}`)
	})

	started := time.Now()
	canceled, err := client.Emails.Cancel(context.Background(), "em_1")
	elapsed := time.Since(started)

	if err != nil || canceled.Status != EmailCanceled {
		t.Fatalf("cancel = %+v, %v", canceled, err)
	}
	if elapsed < time.Second {
		t.Fatalf("retried after %v, want at least 1s", elapsed)
	}
}

func dropConnection(w http.ResponseWriter) {
	conn, _, _ := w.(http.Hijacker).Hijack()
	conn.Close()
}

func TestRetriesNetworkErrors(t *testing.T) {
	client, requests := startServer(t, func(attempt int, w http.ResponseWriter, _ *http.Request) {
		if attempt == 1 {
			dropConnection(w)
			return
		}

		reply(w, 200, `{"status":"ok"}`)
	})

	deleted, err := client.Contacts.Delete(context.Background(), "ct_1")

	attempts := len(requests())
	if err != nil || deleted.Status != "ok" || attempts != 2 {
		t.Fatalf("delete = %+v, %v after %d attempts", deleted, err, attempts)
	}
}

func TestNoRetryOfNetworkErrorOnPostWithoutKey(t *testing.T) {
	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		dropConnection(w)
	})

	_, err := client.Emails.Cancel(context.Background(), "em_1")

	var connectionErr *ConnectionError
	if !errors.As(err, &connectionErr) {
		t.Fatalf("err = %T %v", err, err)
	}
	if attempts := len(requests()); attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestConnectionDroppedMidBodyIsAConnectionError(t *testing.T) {
	client, requests := startServer(t, func(attempt int, w http.ResponseWriter, _ *http.Request) {
		if attempt == 1 {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(200)
			io.WriteString(w, `{"id":`)
			w.(http.Flusher).Flush()
			dropConnection(w)
			return
		}

		reply(w, 200, `{"id":"em_1"}`)
	})

	email, err := client.Emails.Get(context.Background(), "em_1")
	if err != nil || email.ID != "em_1" || len(requests()) != 2 {
		t.Fatalf("retry = %+v, %v after %d attempts", email, err, len(requests()))
	}

	client2, _ := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		dropConnection(w)
	})
	client2.maxRetries = 0

	_, err = client2.Emails.Get(context.Background(), "em_1")

	var connectionErr *ConnectionError
	if !errors.As(err, &connectionErr) {
		t.Fatalf("err = %T %v", err, err)
	}
}

func TestTimeoutCoversReadingTheBody(t *testing.T) {
	client, _ := startServer(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{"id":`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	client.maxRetries = 0
	client.timeout = 50 * time.Millisecond

	_, err := client.Emails.Get(context.Background(), "em_1")

	var connectionErr *ConnectionError
	if !errors.As(err, &connectionErr) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %T %v", err, err)
	}
}

func TestCanceledContextIsNotAConnectionError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, requests := startServer(t, func(_ int, _ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	})

	_, err := client.Emails.Get(ctx, "em_1")

	var connectionErr *ConnectionError
	if !errors.Is(err, context.Canceled) || errors.As(err, &connectionErr) {
		t.Fatalf("err = %T %v", err, err)
	}
	if attempts := len(requests()); attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDebugOutputHidesTheAPIKey(t *testing.T) {
	client := NewClient("ibx_secret_value")

	for _, output := range []string{
		fmt.Sprintf("%v", client),
		fmt.Sprintf("%+v", client),
		fmt.Sprintf("%#v", client),
		fmt.Sprintf("%v", *client),
		fmt.Sprintf("%+v", *client),
		fmt.Sprintf("%#v", *client),
		fmt.Sprintf("%+v", client.Emails),
	} {
		if strings.Contains(output, "ibx_secret_value") {
			t.Fatalf("output has the key: %s", output)
		}
	}
}

func TestMapsErrors(t *testing.T) {
	validationError := `{"name":"validation_error","status":400,"message":"to is required",` +
		`"errors":[{"message":"to is required","location":"to"}],"request_id":"req_2"}`

	client, _ := startServer(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/proxy") {
			w.Header().Set("X-Request-Id", "req_proxy")
			w.WriteHeader(502)
			io.WriteString(w, "Bad gateway")
			return
		}

		reply(w, 400, validationError)
	})
	client.maxRetries = 0

	_, err := client.Emails.Send(context.Background(), SendEmailParams{})

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T", err)
	}

	mapped := apiErr.Name == ErrValidation &&
		apiErr.Status == 400 &&
		apiErr.RequestID == "req_2" &&
		apiErr.Errors[0].Location == "to"
	if !mapped {
		t.Fatalf("error = %+v", apiErr)
	}

	if apiErr.Error() != "inboxjot: validation_error (400): to is required" {
		t.Fatalf("message = %q", apiErr.Error())
	}

	_, err = client.Emails.Get(context.Background(), "proxy")

	mapped = errors.As(err, &apiErr) &&
		apiErr.Name == "" &&
		apiErr.Status == 502 &&
		apiErr.Message == "Bad gateway" &&
		apiErr.RequestID == "req_proxy"
	if !mapped {
		t.Fatalf("proxy error = %+v", apiErr)
	}
}

func TestPatchBodies(t *testing.T) {
	updatedContact := `{"id":"ct_1","email":"jane+news@acme.dev","first_name":null,"last_name":"Lovelace",` +
		`"status":"subscribed","language":"en","properties":{},"segments":[{"id":"seg_1","name":"Customers"}],` +
		`"topics":[{"id":"tp_1","name":"News","subscribed":true,"status":null}],` +
		`"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}`

	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		reply(w, 200, updatedContact)
	})

	ctx := context.Background()
	cases := []struct {
		params UpdateContactParams
		want   string
	}{
		{
			params: UpdateContactParams{},
			want:   `{}`,
		},
		{
			params: UpdateContactParams{
				FirstName: Null[string](),
				LastName:  Value("Lovelace"),
			},
			want: `{"first_name":null,"last_name":"Lovelace"}`,
		},
		{
			params: UpdateContactParams{
				Notes:      Value(""),
				Properties: map[string]any{"plan": nil, "seats": 3},
			},
			want: `{"notes":"","properties":{"plan":null,"seats":3}}`,
		},
		{
			params: UpdateContactParams{
				SegmentIDs: []string{},
				Topics:     []TopicChoice{{ID: "tp_1", Status: TopicUnsubscribed}},
			},
			want: `{"segment_ids":[],"topics":[{"id":"tp_1","status":"unsubscribed"}]}`,
		},
	}

	for i, tc := range cases {
		contact, err := client.Contacts.Update(ctx, "jane+news@acme.dev", tc.params)
		if err != nil {
			t.Fatal(err)
		}

		decoded := contact.FirstName == nil &&
			contact.Segments[0].Name == "Customers" &&
			*contact.LastName == "Lovelace" &&
			contact.Topics[0].Status == nil &&
			!contact.CreatedAt.IsZero() &&
			!contact.UpdatedAt.IsZero()
		if !decoded {
			t.Fatalf("decoded contact = %+v", contact)
		}

		got := requests()[i]
		sent := got.body == tc.want &&
			got.method == http.MethodPatch &&
			got.path == "/v1/contacts/jane+news@acme.dev"
		if !sent {
			t.Fatalf("case %d: %s %s %s, want body %s", i, got.method, got.path, got.body, tc.want)
		}
	}

	scheduledAt := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	_, err := client.Emails.Update(ctx, "em_1", UpdateEmailParams{ScheduledAt: scheduledAt})
	if err != nil {
		t.Fatal(err)
	}

	rescheduleBody := requests()[len(cases)].body
	if rescheduleBody != `{"scheduled_at":"2026-10-20T09:00:00Z"}` {
		t.Fatalf("reschedule body = %s", rescheduleBody)
	}
}

func TestSendBodyEncodesAttachmentsAndSchedule(t *testing.T) {
	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		reply(w, 202, `{"id":"em_1"}`)
	})

	params := SendEmailParams{
		To:          []string{"jane@acme.dev"},
		Template:    &Template{ID: "tpl_1"},
		ScheduledAt: time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC),
		Attachments: []AttachmentParams{{Filename: "a.txt", Content: []byte("hi")}},
	}

	_, err := client.Emails.Send(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"to":["jane@acme.dev"],"template":{"id":"tpl_1"},"scheduled_at":"2026-10-20T09:00:00Z",` +
		`"attachments":[{"filename":"a.txt","content":"aGk="}]}`
	got := requests()[0].body
	if got != want {
		t.Fatalf("body = %s", got)
	}
}

func TestSavedContactDecodesTopics(t *testing.T) {
	savedContact := `{"id":"ct_1","email":"jane@acme.dev","first_name":"Jane","last_name":null,` +
		`"status":"subscribed","language":"en","properties":{},"segments":[],` +
		`"topics":[{"id":"tp_1","name":"News","subscribed":true,"status":"subscribed"}],` +
		`"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z","created":true}`

	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		reply(w, 201, savedContact)
	})

	params := CreateContactParams{Email: "jane@acme.dev", FirstName: Value("Jane"), SegmentIDs: []string{"seg_1"}}
	contact, err := client.Contacts.Create(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	decoded := contact.Created &&
		contact.ID == "ct_1" &&
		*contact.FirstName == "Jane" &&
		*contact.Topics[0].Status == ContactSubscribed
	if !decoded {
		t.Fatalf("decoded contact = %+v", contact)
	}

	body := requests()[0].body
	if body != `{"email":"jane@acme.dev","first_name":"Jane","segment_ids":["seg_1"]}` {
		t.Fatalf("body = %s", body)
	}
}

func TestEraseContact(t *testing.T) {
	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		reply(w, 202, `{"status":"queued"}`)
	})

	erased, err := client.Contacts.Erase(context.Background(), EraseContactParams{Email: "jane+news@acme.dev"})
	if err != nil {
		t.Fatal(err)
	}
	if erased.Status != ErasureQueued {
		t.Fatalf("erase = %+v", erased)
	}

	got := requests()[0]
	sent := got.method == http.MethodPost &&
		got.path == "/v1/contacts/erase" &&
		got.query == "" &&
		got.body == `{"email":"jane+news@acme.dev"}`
	if !sent {
		t.Fatalf("%s %s?%s %s", got.method, got.path, got.query, got.body)
	}
}

func TestListQueries(t *testing.T) {
	emailPage := `{"data":[{"id":"em_1","from":"a@acme.dev","to":[],"cc":[],"bcc":[],"subject":"Hi","tags":[],` +
		`"status":"sent","source":"smtp","created_at":"2026-10-10T09:00:00.000Z","scheduled_at":null,"sent_at":null}],` +
		`"next_cursor":"cur_2","prev_cursor":null}`
	contactPage := `{"data":[{"id":"ct_1","email":"jane@acme.dev","first_name":null,"last_name":null,"properties":{},` +
		`"status":"subscribed","is_suppressed":false,"segments":[{"id":"seg_1","name":"Customers"}],` +
		`"last_activity":{"type":"opened","at":"2026-10-10T09:00:00Z"},"created_at":"2026-10-10T09:00:00Z"}],` +
		`"total":1,"next_cursor":null,"prev_cursor":null}`

	client, requests := startServer(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/contacts" {
			reply(w, 200, contactPage)
			return
		}

		reply(w, 200, emailPage)
	})

	ctx := context.Background()

	_, err := client.Emails.List(ctx, ListEmailsParams{})
	if err != nil {
		t.Fatal(err)
	}

	emails, err := client.Emails.List(ctx, ListEmailsParams{
		Status:    []EmailStatus{EmailBounced, EmailFailed},
		StatusOp:  "not",
		Source:    SourceSMTP,
		SentAfter: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		After:     "cur_1",
		PerPage:   50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if *emails.NextCursor != "cur_2" || emails.PrevCursor != nil || emails.Data[0].Source != SourceSMTP {
		t.Fatalf("email page = %+v", emails)
	}

	contacts, err := client.Contacts.List(ctx, ListContactsParams{
		Sort:      "last_activity",
		Status:    ContactSubscribed,
		SegmentID: "seg_1",
	})
	if err != nil {
		t.Fatal(err)
	}

	decoded := contacts.Total == 1 &&
		contacts.NextCursor == nil &&
		contacts.Data[0].Segments[0].ID == "seg_1" &&
		contacts.Data[0].LastActivity.Type == "opened"
	if !decoded {
		t.Fatalf("contact page = %+v", contacts)
	}

	got := requests()
	wantQueries := []string{
		"",
		"after=cur_1&per_page=50&sent_after=2026-10-01T00%3A00%3A00Z&source=smtp&status=bounced%2Cfailed&status_op=not",
		"segment_id=seg_1&sort=last_activity&status=subscribed",
	}
	for i, want := range wantQueries {
		if got[i].query != want {
			t.Fatalf("request %d query = %s, want %s", i, got[i].query, want)
		}
	}
}
