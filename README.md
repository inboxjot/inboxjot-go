![Inboxjot](https://inboxjot.com/og.png)

<p align="center">The official Go SDK for the Inboxjot email API. <br />Email for builders and their agents.</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/inboxjot/inboxjot-go"><img src="https://inboxjot.com/badges/go.svg?v=2" alt="Go Reference" height="24" /></a>
  <a href="./LICENSE"><img src="https://inboxjot.com/badges/license.svg?v=2" alt="License" height="24" /></a>
  <a href="https://inboxjot.com/docs"><img src="https://inboxjot.com/badges/docs.svg?v=2" alt="Docs" height="24" /></a>
</p>

Send email, manage contacts, campaigns, templates and domains, and verify webhooks from your Go app. The SDK uses only the standard library and needs Go 1.24 or later.

### Quick Start

```sh
go get github.com/inboxjot/inboxjot-go
```

```go
client := inboxjot.NewClient("ibx_...")

email, err := client.Emails.Send(ctx, inboxjot.SendEmailParams{
  From:    "Acme <hi@acme.dev>",
  To:      []string{"jane@example.com"},
  Subject: "Welcome",
  HTML:    "<p>Hi Jane</p>",
})
```

`NewClient("")` reads `INBOXJOT_API_KEY` from the environment. The options are `WithBaseURL`, `WithHTTPClient`, `WithTimeout` and `WithMaxRetries`.

### Resources

```go
client.Emails.Batch(ctx, []inboxjot.BatchEmailParams{{From: from, To: []string{"a@example.com"}, Subject: subject, HTML: html}})
client.Emails.List(ctx, inboxjot.ListEmailsParams{Source: inboxjot.SourceAPI, PerPage: 50})

client.Contacts.Create(ctx, inboxjot.CreateContactParams{Email: "jane@example.com", FirstName: inboxjot.Value("Jane")})
client.Contacts.Update(ctx, "jane@example.com", inboxjot.UpdateContactParams{LastName: inboxjot.Null[string]()})
client.Contacts.Erase(ctx, inboxjot.EraseContactParams{Email: "jane@example.com"})
client.Events.Send(ctx, inboxjot.SendEventParams{Name: "signed_up", Email: "jane@example.com"})

client.Campaigns.Send(ctx, "cmp_...", inboxjot.SendCampaignParams{ScheduledAt: time.Now().Add(24 * time.Hour)})
client.Templates.Publish(ctx, "tpl_...")
client.Segments.AddMembers(ctx, "seg_...", inboxjot.AddSegmentMembersParams{ContactIDs: []string{"con_..."}})
```

Every method takes a `context.Context`, then path ids, then one params struct, then request options. `Contacts.Get`, `Update` and `Delete` take a contact id or the contact's email. In update params, `inboxjot.Value(v)` sets a field, `inboxjot.Null[T]()` clears it, and a field you leave out does not change.

An image that is already online goes straight into the content as `<img src="https://…">`. To host a PNG, JPEG or GIF of up to 5 MB on Inboxjot, upload it:

```go
file, _ := os.ReadFile("logo.png")
image, err := client.Images.Upload(ctx, inboxjot.UploadImageParams{File: file, FileName: "logo.png"})
```

The client covers the whole public API: `Emails`, `Contacts`, `Events`, `Campaigns`, `Templates`, `Domains`, `Images`, `Segments`, `Topics` and `Webhooks`. The [API reference](https://inboxjot.com/docs) lists every endpoint and field.

### Pagination

Lists page with a cursor. Pass `NextCursor` back as `After`. It is `nil` on the last page.

```go
params := inboxjot.ListContactsParams{PerPage: 100}
for {
  page, err := client.Contacts.List(ctx, params)
  if err != nil || page.NextCursor == nil {
    break
  }
  params.After = *page.NextCursor
}
```

### Idempotency and Retries

Pass an idempotency key as a request option:

```go
client.Emails.Send(ctx, email, inboxjot.WithIdempotencyKey("welcome-"+userID))
```

The client retries with exponential backoff and honors `Retry-After`. A `429` always retries. A `5xx`, a timeout or a network error retries only for a `GET`, a `DELETE` or a request with an idempotency key, so an email is never sent twice. A `Retry-After` over 60 seconds, such as the daily sending limit, goes straight to the caller. `Images.Upload` refuses an idempotency key, because each attempt gets a new upload URL.

### Errors

An API error is an `*inboxjot.Error` with a typed `Name`, plus `Status`, `Message`, `Errors` and `RequestID`. A network error or timeout is an `*inboxjot.ConnectionError`.

```go
var apiErr *inboxjot.Error
if errors.As(err, &apiErr) && apiErr.Name == inboxjot.ErrNotFound {
  // ...
}
```

### Verify Webhooks

Inboxjot signs webhooks with the [Standard Webhooks](https://www.standardwebhooks.com) spec. Verify the raw body before you parse it:

```go
func handle(w http.ResponseWriter, r *http.Request) {
  payload, _ := io.ReadAll(r.Body)
  event, err := webhook.Verify(payload, r.Header, os.Getenv("INBOXJOT_WEBHOOK_SECRET"))
  if err != nil {
    http.Error(w, "invalid signature", http.StatusBadRequest)
    return
  }
  if event.Type == webhook.EmailBounced {
    log.Println(event.Email.EmailID, event.Email.Bounce.Type)
  }
}
```

A bad signature, a missing header or a timestamp more than 5 minutes off returns an error.

### Contributing

Run `gofmt -l .`, `go vet ./...` and `go test -race ./...` before you open a pull request.

### License

[MIT](./LICENSE) &copy; [Arik Chakma](https://x.com/imarikchakma)
