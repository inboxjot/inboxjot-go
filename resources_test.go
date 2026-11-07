package inboxjot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

type route struct {
	method string
	path   string
	query  string
	body   string
	call   func(context.Context, *Client) error
}

func ignore[T any](_ *T, err error) error {
	return err
}

func checkRoutes(t *testing.T, routes []route) {
	t.Helper()

	client, requests := startServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		reply(w, 200, `{}`)
	})

	for i, want := range routes {
		err := want.call(context.Background(), client)
		if err != nil {
			t.Fatalf("%s %s: %v", want.method, want.path, err)
		}

		got := requests()[i]
		if got.method != want.method || got.path != want.path || got.query != want.query || got.body != want.body {
			t.Fatalf("got %s %s?%s %s, want %s %s?%s %s", got.method, got.path, got.query, got.body, want.method, want.path, want.query, want.body)
		}
	}
}

func decode[T any](t *testing.T, body string) T {
	t.Helper()

	var value T
	err := json.Unmarshal([]byte(body), &value)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

var testTime = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

func TestCampaignRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/campaigns", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.List(ctx, ListCampaignsParams{}))
		}},
		{method: "GET", path: "/v1/campaigns", query: "after=cur_1&direction=asc&per_page=5&query=launch&sort=name&status=draft%2Csent", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.List(ctx, ListCampaignsParams{Query: "launch", Status: []CampaignStatus{CampaignDraft, CampaignSent}, Sort: "name", Direction: "asc", After: "cur_1", PerPage: 5}))
		}},
		{method: "POST", path: "/v1/campaigns", body: `{"name":"Launch","reply_to":null,"preheader":"","include_segment_ids":[]}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Create(ctx, CampaignParams{Name: Value("Launch"), ReplyTo: Null[string](), Preheader: Value(""), IncludeSegmentIDs: []string{}}))
		}},
		{method: "GET", path: "/v1/campaigns/cmp_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Get(ctx, "cmp_1"))
		}},
		{method: "PATCH", path: "/v1/campaigns/cmp_1", body: `{"template_id":null}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Update(ctx, "cmp_1", CampaignParams{TemplateID: Null[string]()}))
		}},
		{method: "DELETE", path: "/v1/campaigns/cmp_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Delete(ctx, "cmp_1"))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/send", body: `{}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Send(ctx, "cmp_1", SendCampaignParams{}))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/send", body: `{"scheduled_at":"2026-10-20T09:00:00Z"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Send(ctx, "cmp_1", SendCampaignParams{ScheduledAt: testTime}))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/cancel", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.Cancel(ctx, "cmp_1"))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/test", body: `{"to":["jane@acme.dev"]}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.SendTest(ctx, "cmp_1", SendTestCampaignParams{To: []string{"jane@acme.dev"}}))
		}},
		{method: "GET", path: "/v1/campaigns/cmp_1/versions", query: "filter=saved&per_page=5", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.ListVersions(ctx, "cmp_1", ListVersionsParams{Filter: "saved", PerPage: 5}))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/versions", body: `{"name":"First"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.CreateVersion(ctx, "cmp_1", CreateVersionParams{Name: "First"}))
		}},
		{method: "GET", path: "/v1/campaigns/cmp_1/versions/ver_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.GetVersion(ctx, "cmp_1", "ver_1"))
		}},
		{method: "PATCH", path: "/v1/campaigns/cmp_1/versions/ver_1", body: `{"name":"Second"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.UpdateVersion(ctx, "cmp_1", "ver_1", UpdateVersionParams{Name: Value("Second")}))
		}},
		{method: "PATCH", path: "/v1/campaigns/cmp_1/versions/ver_1", body: `{"name":null}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.UpdateVersion(ctx, "cmp_1", "ver_1", UpdateVersionParams{}))
		}},
		{method: "POST", path: "/v1/campaigns/cmp_1/versions/ver_1/restore", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Campaigns.RestoreVersion(ctx, "cmp_1", "ver_1"))
		}},
	})
}

func TestTemplateRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/templates", query: "before=cur_1&query=welcome&sort=updated", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.List(ctx, ListTemplatesParams{Query: "welcome", Sort: "updated", Before: "cur_1"}))
		}},
		{method: "POST", path: "/v1/templates", body: `{"name":"Welcome","reply_to":null,"starter_id":"welcome"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.Create(ctx, CreateTemplateParams{Name: "Welcome", ReplyTo: Null[string](), StarterID: "welcome"}))
		}},
		{method: "GET", path: "/v1/templates/tpl_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.Get(ctx, "tpl_1"))
		}},
		{method: "PATCH", path: "/v1/templates/tpl_1", body: `{"preheader":""}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.Update(ctx, "tpl_1", UpdateTemplateParams{Preheader: Value("")}))
		}},
		{method: "DELETE", path: "/v1/templates/tpl_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.Delete(ctx, "tpl_1"))
		}},
		{method: "POST", path: "/v1/templates/tpl_1/publish", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.Publish(ctx, "tpl_1"))
		}},
		{method: "GET", path: "/v1/templates/tpl_1/versions", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.ListVersions(ctx, "tpl_1", ListVersionsParams{}))
		}},
		{method: "POST", path: "/v1/templates/tpl_1/versions", body: `{"name":"First"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.CreateVersion(ctx, "tpl_1", CreateVersionParams{Name: "First"}))
		}},
		{method: "GET", path: "/v1/templates/tpl_1/versions/ver_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.GetVersion(ctx, "tpl_1", "ver_1"))
		}},
		{method: "PATCH", path: "/v1/templates/tpl_1/versions/ver_1", body: `{"name":null}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.UpdateVersion(ctx, "tpl_1", "ver_1", UpdateVersionParams{Name: Null[string]()}))
		}},
		{method: "POST", path: "/v1/templates/tpl_1/versions/ver_1/restore", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Templates.RestoreVersion(ctx, "tpl_1", "ver_1"))
		}},
	})
}

func TestDomainRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/domains", query: "after=cur_1&per_page=5", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Domains.List(ctx, PageParams{After: "cur_1", PerPage: 5}))
		}},
		{method: "POST", path: "/v1/domains", body: `{"name":"acme.dev","region":"us-east-1"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Domains.Create(ctx, CreateDomainParams{Name: "acme.dev", Region: "us-east-1"}))
		}},
		{method: "GET", path: "/v1/domains/dom_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Domains.Get(ctx, "dom_1"))
		}},
		{method: "DELETE", path: "/v1/domains/dom_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Domains.Delete(ctx, "dom_1"))
		}},
		{method: "POST", path: "/v1/domains/dom_1/verify", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Domains.Verify(ctx, "dom_1"))
		}},
	})
}

func TestImageRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/images", query: "direction=desc&query=logo&type=png", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Images.List(ctx, ListImagesParams{Query: "logo", Type: ImageTypePNG, Direction: "desc"}))
		}},
		{method: "GET", path: "/v1/images/img_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Images.Get(ctx, "img_1"))
		}},
		{method: "DELETE", path: "/v1/images/img_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Images.Delete(ctx, "img_1"))
		}},
	})
}

func TestSegmentRoutes(t *testing.T) {
	conditions := &SegmentConditions{Match: SegmentMatchAll, Rules: []SegmentRule{
		{Field: "plan", Operator: OperatorIs, Value: "pro"},
		{Field: "email", Operator: OperatorIsSet},
		{Field: "opened", Operator: OperatorAtLeast, Value: RuleWindow{Count: 2, Days: 30}},
	}}

	checkRoutes(t, []route{
		{method: "GET", path: "/v1/segments", query: "query=pro&sort=contacts&type=dynamic", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.List(ctx, ListSegmentsParams{Type: SegmentDynamic, Query: "pro", Sort: "contacts"}))
		}},
		{method: "POST", path: "/v1/segments", body: `{"name":"Pro","type":"dynamic","conditions":{"match":"all","rules":[{"field":"plan","operator":"is","value":"pro"},{"field":"email","operator":"is_set"},{"field":"opened","operator":"at_least","value":{"count":2,"days":30}}]}}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.Create(ctx, CreateSegmentParams{Name: "Pro", Type: SegmentDynamic, Conditions: conditions}))
		}},
		{method: "GET", path: "/v1/segments/seg_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.Get(ctx, "seg_1"))
		}},
		{method: "PATCH", path: "/v1/segments/seg_1", body: `{"description":null}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.Update(ctx, "seg_1", UpdateSegmentParams{Description: Null[string]()}))
		}},
		{method: "DELETE", path: "/v1/segments/seg_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.Delete(ctx, "seg_1"))
		}},
		{method: "GET", path: "/v1/segments/seg_1/members", query: "per_page=5&query=jane", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.ListMembers(ctx, "seg_1", ListSegmentMembersParams{Query: "jane", PerPage: 5}))
		}},
		{method: "POST", path: "/v1/segments/seg_1/members", body: `{"contact_ids":["con_1"]}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.AddMembers(ctx, "seg_1", AddSegmentMembersParams{ContactIDs: []string{"con_1"}}))
		}},
		{method: "DELETE", path: "/v1/segments/seg_1/members", body: `{"contact_ids":["con_1"]}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Segments.RemoveMembers(ctx, "seg_1", RemoveSegmentMembersParams{ContactIDs: []string{"con_1"}}))
		}},
	})
}

func TestTopicRoutes(t *testing.T) {
	enabled := false

	checkRoutes(t, []route{
		{method: "GET", path: "/v1/topics", query: "before=cur_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Topics.List(ctx, PageParams{Before: "cur_1"}))
		}},
		{method: "POST", path: "/v1/topics", body: `{"name":"News","default_subscribed":false,"visibility":"private"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Topics.Create(ctx, CreateTopicParams{Name: "News", DefaultSubscribed: &enabled, Visibility: TopicPrivate}))
		}},
		{method: "GET", path: "/v1/topics/top_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Topics.Get(ctx, "top_1"))
		}},
		{method: "PATCH", path: "/v1/topics/top_1", body: `{"sender_name":null,"redirect_url":"https://acme.dev"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Topics.Update(ctx, "top_1", UpdateTopicParams{SenderName: Null[string](), RedirectURL: Value("https://acme.dev")}))
		}},
		{method: "DELETE", path: "/v1/topics/top_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Topics.Delete(ctx, "top_1"))
		}},
	})
}

func TestWebhookRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/webhooks", query: "per_page=5", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Webhooks.List(ctx, PageParams{PerPage: 5}))
		}},
		{method: "POST", path: "/v1/webhooks", body: `{"name":"Bounces","url":"https://acme.dev/hook","events":["email.bounced"]}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Webhooks.Create(ctx, CreateWebhookParams{Name: "Bounces", URL: "https://acme.dev/hook", Events: []WebhookEventType{WebhookEmailBounced}}))
		}},
		{method: "GET", path: "/v1/webhooks/ep_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Webhooks.Get(ctx, "ep_1"))
		}},
		{method: "PATCH", path: "/v1/webhooks/ep_1", body: `{"events":[],"status":"disabled"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Webhooks.Update(ctx, "ep_1", UpdateWebhookParams{Events: []WebhookEventType{}, Status: EndpointDisabled}))
		}},
		{method: "DELETE", path: "/v1/webhooks/ep_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Webhooks.Delete(ctx, "ep_1"))
		}},
	})
}

func TestContactRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/contacts", query: "direction=asc&query=jane&segment_id=seg_1&sort=contact&status=pending", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.List(ctx, ListContactsParams{Query: "jane", Status: ContactPending, SegmentID: "seg_1", Sort: "contact", Direction: "asc"}))
		}},
		{method: "POST", path: "/v1/contacts", body: `{"email":"jane@acme.dev"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.Create(ctx, CreateContactParams{Email: "jane@acme.dev"}))
		}},
		{method: "GET", path: "/v1/contacts/con_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.Get(ctx, "con_1"))
		}},
		{method: "PATCH", path: "/v1/contacts/con_1", body: `{"notes":null}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.Update(ctx, "con_1", UpdateContactParams{Notes: Null[string]()}))
		}},
		{method: "DELETE", path: "/v1/contacts/con_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.Delete(ctx, "con_1"))
		}},
		{method: "POST", path: "/v1/contacts/erase", body: `{"email":"jane@acme.dev"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Contacts.Erase(ctx, EraseContactParams{Email: "jane@acme.dev"}))
		}},
	})
}

func TestEmailRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "GET", path: "/v1/emails", query: "sent_before=2026-10-20T09%3A00%3A00Z&sort=to&status=sent%2Copened", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.List(ctx, ListEmailsParams{Status: []EmailStatus{EmailSent, EmailOpened}, SentBefore: testTime, Sort: "to"}))
		}},
		{method: "POST", path: "/v1/emails", body: `{"to":["jane@acme.dev"],"text":"Hi"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.Send(ctx, SendEmailParams{To: []string{"jane@acme.dev"}, Text: "Hi"}))
		}},
		{method: "POST", path: "/v1/emails/batch", body: `[{"to":["jane@acme.dev"],"text":"Hi"}]`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.Batch(ctx, []BatchEmailParams{{To: []string{"jane@acme.dev"}, Text: "Hi"}}))
		}},
		{method: "GET", path: "/v1/emails/em_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.Get(ctx, "em_1"))
		}},
		{method: "PATCH", path: "/v1/emails/em_1", body: `{"scheduled_at":"2026-10-20T09:00:00Z"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.Update(ctx, "em_1", UpdateEmailParams{ScheduledAt: testTime}))
		}},
		{method: "POST", path: "/v1/emails/em_1/cancel", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.Cancel(ctx, "em_1"))
		}},
		{method: "GET", path: "/v1/emails/em_1/attachments", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.ListAttachments(ctx, "em_1"))
		}},
		{method: "GET", path: "/v1/emails/em_1/attachments/att_1", call: func(ctx context.Context, c *Client) error {
			return ignore(c.Emails.GetAttachment(ctx, "em_1", "att_1"))
		}},
	})
}

func TestEventRoutes(t *testing.T) {
	checkRoutes(t, []route{
		{method: "POST", path: "/v1/events", body: `{"name":"signed_up","email":"jane@acme.dev"}`, call: func(ctx context.Context, c *Client) error {
			return ignore(c.Events.Send(ctx, SendEventParams{Name: "signed_up", Email: "jane@acme.dev"}))
		}},
	})
}

func TestDeletesAndRetriesWithABody(t *testing.T) {
	client, requests := startServer(t, func(attempt int, w http.ResponseWriter, _ *http.Request) {
		if attempt == 1 {
			w.Header().Set("Retry-After", "0")
			reply(w, 503, `{"name":"service_unavailable","status":503,"message":"Down","errors":[],"request_id":null}`)
			return
		}

		reply(w, 200, `{"removed":1}`)
	})

	removed, err := client.Segments.RemoveMembers(context.Background(), "seg_1", RemoveSegmentMembersParams{ContactIDs: []string{"con_1"}})
	if err != nil || removed.Removed != 1 {
		t.Fatalf("remove = %+v, %v", removed, err)
	}

	got := requests()
	if len(got) != 2 || got[1].body != `{"contact_ids":["con_1"]}` || got[1].header.Get("Content-Type") != "application/json" {
		t.Fatalf("requests = %+v", got)
	}
}

func TestDecodesCampaigns(t *testing.T) {
	campaign := decode[Campaign](t, `{"id":"cmp_1","name":"Launch","status":"archived","from":"hi@acme.dev","reply_to":null,
		"subject":"Hi","preheader":"","html":null,"text":null,"template_id":null,"template":null,
		"include_segment_ids":["seg_1"],"include_segments":[{"id":"seg_1","name":"All"}],"exclude_segment_ids":[],"exclude_segments":[],
		"topic_id":"top_1","topic":null,"audience_count":null,"queued_count":0,"skipped_count":0,"paused_reason":"new_reason",
		"failure_reason":null,"scheduled_at":null,"started_at":"2026-10-10T09:00:00.000Z","completed_at":null,"warmup_next_limit":null,
		"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}`)
	if campaign.Status != "archived" || *campaign.PausedReason != "new_reason" || campaign.HTML != nil || campaign.Topic != nil ||
		campaign.ScheduledAt != nil || campaign.StartedAt == nil || campaign.IncludeSegments[0].Name != "All" {
		t.Fatalf("campaign = %+v", campaign)
	}

	page := decode[CampaignPage](t, `{"data":[{"id":"cmp_1","name":"Launch","status":"sent","subject":"Hi","include_segments":[],
		"exclude_segments":[],"topic":{"id":"top_1","name":"News"},"audience_count":10,"queued_count":10,"skipped_count":0,"paused_reason":null,
		"scheduled_at":null,"started_at":null,"completed_at":null,"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z",
		"stats":{"delivered":9,"opened":5,"clicked":2}},{"id":"cmp_2","stats":null}],"next_cursor":"cur_2","prev_cursor":null}`)
	if page.Data[0].Stats.Clicked != 2 || page.Data[1].Stats != nil || *page.NextCursor != "cur_2" || page.PrevCursor != nil {
		t.Fatalf("page = %+v", page)
	}

	queued := decode[QueuedIDs](t, `{"ids":["em_1"]}`)
	if queued.IDs[0] != "em_1" {
		t.Fatalf("queued = %+v", queued)
	}
}

func TestDecodesVersions(t *testing.T) {
	detail := decode[DocumentVersionDetail](t, `{"id":"ver_1","reason":"auto_save","name":null,
		"editors":[{"kind":"robot","id":"usr_1","name":"Riti"}],"from":"hi@acme.dev","reply_to":null,"subject":"Hi","preheader":"",
		"html":"<p>Hi</p>","text":"Hi","content":{"type":"doc"},"created_at":"2026-10-10T09:00:00.000Z"}`)
	if detail.Reason != "auto_save" || detail.Name != nil || detail.Editors[0].Kind != "robot" || detail.Content["type"] != "doc" || detail.CreatedAt.IsZero() {
		t.Fatalf("detail = %+v", detail)
	}

	page := decode[DocumentVersionPage](t, `{"data":[{"id":"ver_1","reason":"manual","name":"First","editors":[],"created_at":"2026-10-10T09:00:00.000Z"}],"total":1,"next_cursor":null,"prev_cursor":null}`)
	if page.Total != 1 || page.Data[0].Reason != VersionManual || *page.Data[0].Name != "First" {
		t.Fatalf("page = %+v", page)
	}
}

func TestDecodesTemplates(t *testing.T) {
	template := decode[TemplateDetail](t, `{"id":"tpl_1","name":"Welcome","from":"","reply_to":null,"subject":"","preheader":"","html":"","text":"",
		"published_version_id":null,"has_unpublished_changes":true,"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}`)
	if template.ReplyTo != nil || template.PublishedVersionID != nil || !template.HasUnpublishedChanges {
		t.Fatalf("template = %+v", template)
	}

	page := decode[TemplatePage](t, `{"data":[{"id":"tpl_1","name":"Welcome","from":"","subject":"","preheader":"","stats":{"sent":3,"opened":2,"clicked":1},
		"used_in":{"count":1,"first":{"id":"cmp_1","name":"Launch"}},"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"},
		{"id":"tpl_2","used_in":{"count":0,"first":null}}],"next_cursor":null,"prev_cursor":null}`)
	if page.Data[0].Stats.Sent != 3 || page.Data[0].UsedIn.First.Name != "Launch" || page.Data[1].UsedIn.First != nil {
		t.Fatalf("page = %+v", page)
	}
}

func TestDecodesDomains(t *testing.T) {
	domain := decode[Domain](t, `{"id":"dom_1","name":"acme.dev","mail_from_domain":"send.acme.dev","status":"paused",
		"records":[{"kind":"spf","type":"TXT","name":"acme.dev","value":"v=spf1","status":"pending"},{"kind":"return_path","type":"MX","name":"send","value":"mx","priority":10,"status":"verified"}],
		"region":"us-east-1","tls_policy":"opportunistic","receiving_enabled":false,"receiving_target":"","receiving_status":"not_started",
		"receiving_found_mx":null,"receiving_checked_at":null,"receiving_verified_at":null,"open_tracking":true,"click_tracking":false,
		"tracking_domain":null,"tracking_status":"expired","tracking_records":[],"tracking_verified_at":null,"mailing_address":null,"verified_at":null,
		"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z","sent_last_30_days":1,"bounced_last_30_days":0,"is_default":true,
		"receiving":{"host":"acme.dev","mx":"mx.inboxjot.com","shared_address":null},
		"warmup":{"enabled":false,"started_on":null,"day":null,"limit":null,"next_limit":null,"sent_today":0,"full_volume_on":null}}`)
	if domain.Status != "paused" || domain.TrackingStatus != "expired" || domain.Records[0].Kind != "spf" || domain.Records[0].Priority != nil ||
		*domain.Records[1].Priority != 10 || domain.VerifiedAt != nil || domain.Receiving.SharedAddress != nil || domain.Warmup.Day != nil {
		t.Fatalf("domain = %+v", domain)
	}

	page := decode[DomainPage](t, `{"data":[{"id":"dom_1","name":"acme.dev","status":"verified","pending_record_kinds":["dkim","future"],"region":"us-east-1",
		"is_default":true,"open_tracking":false,"click_tracking":false,"sent_last_30_days":0,"verified_at":"2026-10-10T09:00:00.000Z",
		"created_at":"2026-10-10T09:00:00.000Z"}],"next_cursor":null,"prev_cursor":null}`)
	if page.Data[0].PendingRecordKinds[1] != "future" || page.Data[0].VerifiedAt == nil {
		t.Fatalf("page = %+v", page)
	}
}

func TestDecodesImages(t *testing.T) {
	page := decode[ImagePage](t, `{"data":[{"id":"img_1","file_name":"a.png","url":null,"content_type":"image/webp","size_bytes":10,"width":1,"height":1,
		"creator_name":null,"created_at":"2026-10-10T09:00:00.000Z"}],"total":1,"next_cursor":null,"prev_cursor":null}`)
	if page.Total != 1 || page.Data[0].URL != nil || page.Data[0].ContentType != "image/webp" || page.Data[0].CreatorName != nil {
		t.Fatalf("page = %+v", page)
	}
}

var testPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3}

func startUploadServer(t *testing.T, uploadStatus int) (*Client, func() []recorded) {
	t.Helper()

	return startServer(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/presign":
			reply(w, 200, `{"upload_url":"http://`+r.Host+`/r2/uploads/a.png?X-Amz-Signature=sig","key":"uploads/a.png"}`)
		case "/r2/uploads/a.png":
			w.WriteHeader(uploadStatus)
		default:
			reply(w, 201, `{"id":"img_1","file_name":"a.png","url":"https://cdn/a.png","content_type":"image/png","size_bytes":11,"width":1,"height":1,"creator_name":null,"created_at":"2026-10-10T09:00:00.000Z"}`)
		}
	})
}

func TestUploadsImage(t *testing.T) {
	client, requests := startUploadServer(t, 200)

	image, err := client.Images.Upload(context.Background(), UploadImageParams{File: testPNG, FileName: "a.png"})
	if err != nil {
		t.Fatal(err)
	}
	if image.ID != "img_1" {
		t.Fatalf("image = %+v", image)
	}

	got := requests()
	if len(got) != 3 {
		t.Fatalf("got %d requests, want 3", len(got))
	}
	presign, upload, create := got[0], got[1], got[2]

	if presign.method != "POST" || presign.path != "/v1/images/presign" || presign.body != `{"content_type":"image/png","size":11}` {
		t.Fatalf("presign = %s %s %s", presign.method, presign.path, presign.body)
	}
	if upload.method != "PUT" || upload.path != "/r2/uploads/a.png" || upload.query != "X-Amz-Signature=sig" || upload.body != string(testPNG) {
		t.Fatalf("upload = %s %s?%s %q", upload.method, upload.path, upload.query, upload.body)
	}
	if upload.header.Get("Content-Type") != "image/png" || upload.header.Get("Authorization") != "" {
		t.Fatalf("upload headers = %v", upload.header)
	}
	if create.method != "POST" || create.path != "/v1/images" || create.body != `{"key":"uploads/a.png","file_name":"a.png"}` {
		t.Fatalf("create = %s %s %s", create.method, create.path, create.body)
	}
	for _, request := range got {
		if request.header.Get("Idempotency-Key") != "" {
			t.Fatalf("%s %s sent an idempotency key", request.method, request.path)
		}
	}
}

func TestUploadDetectsContentType(t *testing.T) {
	cases := []struct {
		file []byte
		want string
	}{
		{[]byte{0xff, 0xd8, 0xff, 0xe0}, `{"content_type":"image/jpeg","size":4}`},
		{[]byte("GIF89a"), `{"content_type":"image/gif","size":6}`},
	}

	for _, tc := range cases {
		client, requests := startUploadServer(t, 200)

		_, err := client.Images.Upload(context.Background(), UploadImageParams{File: tc.file})
		if err != nil {
			t.Fatal(err)
		}

		got := requests()
		if got[0].body != tc.want || got[2].body != `{"key":"uploads/a.png"}` {
			t.Fatalf("presign = %s, create = %s", got[0].body, got[2].body)
		}
	}
}

func TestUploadRefusesIdempotencyKey(t *testing.T) {
	client, requests := startUploadServer(t, 200)

	_, err := client.Images.Upload(context.Background(), UploadImageParams{File: testPNG}, WithIdempotencyKey("key_1"))
	if err == nil || err.Error() != "inboxjot: image uploads do not support an idempotency key" {
		t.Fatalf("err = %v", err)
	}
	if len(requests()) != 0 {
		t.Fatalf("sent %d requests", len(requests()))
	}
}

func TestUploadRefusesNonImage(t *testing.T) {
	client, requests := startUploadServer(t, 200)

	_, err := client.Images.Upload(context.Background(), UploadImageParams{File: []byte("hello")})
	if err == nil || err.Error() != "inboxjot: the image must be a PNG, JPEG or GIF file" {
		t.Fatalf("err = %v", err)
	}
	if len(requests()) != 0 {
		t.Fatalf("sent %d requests", len(requests()))
	}
}

func TestUploadFailure(t *testing.T) {
	client, requests := startUploadServer(t, 403)

	_, err := client.Images.Upload(context.Background(), UploadImageParams{File: testPNG})

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 403 || apiErr.Message != "the image upload failed with status 403" {
		t.Fatalf("err = %v", err)
	}
	if len(requests()) != 2 {
		t.Fatalf("sent %d requests, want 2", len(requests()))
	}
}

func TestDecodesSegments(t *testing.T) {
	segment := decode[Segment](t, `{"id":"seg_1","name":"Pro","description":null,"type":"smart",
		"conditions":{"match":"none","rules":[{"field":"plan","operator":"fuzzy"},{"field":"opened","operator":"at_least","value":{"count":2,"days":30}}]},
		"contact_count":null,"counted_at":null,"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}`)
	if segment.Type != "smart" || segment.Conditions.Match != "none" || segment.Conditions.Rules[0].Operator != "fuzzy" ||
		segment.Conditions.Rules[0].Value != nil || segment.ContactCount != nil || segment.Description != nil {
		t.Fatalf("segment = %+v", segment)
	}

	page := decode[SegmentPage](t, `{"data":[{"id":"seg_1","name":"Pro","description":"Paid","type":"static","contact_count":4,
		"counted_at":"2026-10-10T09:00:00.000Z","created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}],
		"total":1,"next_cursor":null,"prev_cursor":null}`)
	if page.Total != 1 || *page.Data[0].ContactCount != 4 || page.Data[0].CountedAt == nil {
		t.Fatalf("page = %+v", page)
	}

	members := decode[SegmentMemberPage](t, `{"data":[{"id":"con_1","email":"jane@acme.dev","first_name":null,"last_name":null,"status":"bounced",
		"joined_at":"2026-10-10T09:00:00.000Z"}],"next_cursor":null,"prev_cursor":null}`)
	if members.Data[0].Status != "bounced" || members.Data[0].FirstName != nil {
		t.Fatalf("members = %+v", members)
	}

	if decode[AddedMembers](t, `{"added":2}`).Added != 2 || decode[RemovedMembers](t, `{"removed":3}`).Removed != 3 {
		t.Fatal("member counts did not decode")
	}
}

func TestDecodesTopics(t *testing.T) {
	page := decode[TopicPage](t, `{"data":[{"id":"top_1","name":"News","description":null,"default_subscribed":true,"visibility":"hidden",
		"double_opt_in":false,"form_enabled":false,"sender_name":null,"sender_email":null,"redirect_url":null,"is_default":true,
		"counts":{"subscribed":3,"pending":1,"unsubscribed":0},"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}],
		"next_cursor":null,"prev_cursor":null}`)
	topic := page.Data[0]
	if topic.Visibility != "hidden" || topic.Description != nil || topic.Counts.Pending != 1 || !topic.IsDefault {
		t.Fatalf("topic = %+v", topic)
	}
}

func TestDecodesWebhookEndpoints(t *testing.T) {
	base := `"id":"ep_1","name":"Bounces","url":"https://acme.dev","events":["email.bounced","domain.verified"],"status":"paused","disabled_reason":null,
		"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"`

	created := decode[CreatedWebhookEndpoint](t, `{`+base+`,"secret":"whsec_abc"}`)
	if created.Secret != "whsec_abc" || created.ID != "ep_1" || created.Status != "paused" || created.Events[1] != "domain.verified" {
		t.Fatalf("created = %+v", created)
	}

	detail := decode[WebhookEndpointDetail](t, `{`+base+`,"deliveries_24h":1,"success_rate":null,"avg_duration_ms":null,"retrying":0,
		"recent_deliveries":[{"delivery_id":"del_1","event_id":"evt_1","event_type":"email.bounced","endpoint_id":"ep_1","email_id":"em_1","status":200,
		"attempt":1,"duration_ms":40,"created_at":"2026-10-10T09:00:00.000Z","last_attempt_at":"2026-10-10T09:00:00Z","next_attempt_at":null}]}`)
	if detail.SuccessRate != nil || detail.AvgDurationMs != nil || detail.RecentDeliveries[0].Status != 200 || detail.RecentDeliveries[0].NextAttemptAt != nil || detail.URL != "https://acme.dev" {
		t.Fatalf("detail = %+v", detail)
	}

	page := decode[WebhookEndpointPage](t, `{"data":[{`+base+`,"success_rate":99.5,"last_delivery_at":null}],"next_cursor":null,"prev_cursor":null}`)
	if *page.Data[0].SuccessRate != 99.5 || page.Data[0].LastDeliveryAt != nil || page.Data[0].Name != "Bounces" {
		t.Fatalf("page = %+v", page)
	}
}

func TestDecodesContacts(t *testing.T) {
	contact := decode[Contact](t, `{"id":"con_1","email":"jane@acme.dev","first_name":null,"last_name":null,"status":"blocked","language":"en",
		"properties":{"plan":null,"seats":3},"segments":[],"topics":[{"id":"top_1","name":"News","status":"snoozed","subscribed":false},{"id":"top_2","name":"Tips","status":null,"subscribed":true}],
		"created_at":"2026-10-10T09:00:00.000Z","updated_at":"2026-10-10T09:00:00.000Z"}`)
	if contact.Status != "blocked" || *contact.Topics[0].Status != "snoozed" || contact.Topics[1].Status != nil || contact.FirstName != nil {
		t.Fatalf("contact = %+v", contact)
	}

	page := decode[ContactPage](t, `{"data":[{"id":"con_1","email":"jane@acme.dev","first_name":null,"last_name":null,"properties":{},"status":"subscribed",
		"is_suppressed":true,"segments":[],"last_activity":null,"created_at":"2026-10-10T09:00:00.000Z"}],"total":1,"next_cursor":null,"prev_cursor":null}`)
	if page.Total != 1 || !page.Data[0].IsSuppressed || page.Data[0].LastActivity != nil {
		t.Fatalf("page = %+v", page)
	}

	if decode[StatusOK](t, `{"status":"ok"}`).Status != "ok" {
		t.Fatal("status did not decode")
	}
	if decode[EraseContactResponse](t, `{"status":"done"}`).Status != "done" {
		t.Fatal("erase status did not decode")
	}
}

func TestDecodesEmails(t *testing.T) {
	email := decode[Email](t, `{"id":"em_1","from":"hi@acme.dev","to":["jane@acme.dev"],"cc":[],"bcc":[],"subject":"Hi","tags":[{"name":"kind","value":"receipt"}],
		"status":"quarantined","source":"zapier","scheduled_at":null,"sent_at":null,"created_at":"2026-10-10T09:00:00.000Z"}`)
	if email.Status != "quarantined" || email.Source != "zapier" || email.SentAt != nil || email.Tags[0].Value != "receipt" {
		t.Fatalf("email = %+v", email)
	}

	attachment := decode[Attachment](t, `{"id":"att_1","filename":"a.pdf","content_type":"application/pdf","size":10,"content_id":null,"content_disposition":"attachment",
		"deleted_at":null,"download_url":null,"expires_at":null}`)
	if attachment.ContentID != nil || attachment.DownloadURL != nil || attachment.Size != 10 {
		t.Fatalf("attachment = %+v", attachment)
	}

	rescheduled := decode[RescheduledEmail](t, `{"id":"em_1","scheduled_at":null}`)
	if rescheduled.ScheduledAt != nil {
		t.Fatalf("rescheduled = %+v", rescheduled)
	}

	batch := decode[BatchResponse](t, `{"data":[{"id":"em_1"}]}`)
	if batch.Data[0].ID != "em_1" {
		t.Fatalf("batch = %+v", batch)
	}
}

func TestDecodesEvents(t *testing.T) {
	if decode[SendEventResponse](t, `{"contact_id":"con_1"}`).ContactID != "con_1" {
		t.Fatal("contact id did not decode")
	}
}
