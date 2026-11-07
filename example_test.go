package inboxjot_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/inboxjot/inboxjot-go"
	"github.com/inboxjot/inboxjot-go/webhook"
)

func Example() {
	client := inboxjot.NewClient("ibx_...")

	params := inboxjot.SendEmailParams{
		From:    "Acme <hi@acme.dev>",
		To:      []string{"jane@example.com"},
		Subject: "Welcome",
		HTML:    "<p>Thanks for signing up.</p>",
		Tags:    []inboxjot.Tag{{Name: "category", Value: "welcome"}},
	}
	idempotencyKey := inboxjot.WithIdempotencyKey("welcome-user-123")

	email, err := client.Emails.Send(context.Background(), params, idempotencyKey)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(email.ID)
}

func ExampleEmailService_Send_scheduled() {
	client := inboxjot.NewClient("")

	params := inboxjot.SendEmailParams{
		From:        "Acme <hi@acme.dev>",
		To:          []string{"jane@example.com"},
		Subject:     "Your trial ends tomorrow",
		Text:        "Upgrade any time.",
		ScheduledAt: time.Now().Add(24 * time.Hour),
	}

	email, err := client.Emails.Send(context.Background(), params)
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Emails.Cancel(context.Background(), email.ID)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleError() {
	client := inboxjot.NewClient("")

	_, err := client.Emails.Get(context.Background(), "eml_missing")

	var apiErr *inboxjot.Error
	if errors.As(err, &apiErr) && apiErr.Name == inboxjot.ErrNotFound {
		fmt.Println("no such email")
	}
}

func ExampleContactService_Update() {
	client := inboxjot.NewClient("")

	params := inboxjot.UpdateContactParams{
		FirstName:  inboxjot.Value("Jane"),
		LastName:   inboxjot.Null[string](),
		Properties: map[string]any{"plan": "pro"},
		Topics: []inboxjot.TopicChoice{
			{ID: "top_news", Status: inboxjot.TopicUnsubscribed},
		},
	}

	contact, err := client.Contacts.Update(context.Background(), "jane@example.com", params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(contact.ID)
}

func ExampleEventService_Send() {
	client := inboxjot.NewClient("")

	params := inboxjot.SendEventParams{
		Name:  "signed_up",
		Email: "jane@example.com",
		Data:  map[string]any{"plan": "pro"},
	}

	_, err := client.Events.Send(context.Background(), params)
	if err != nil {
		log.Fatal(err)
	}
}

func Example_webhook() {
	http.HandleFunc("/webhooks/inboxjot", func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unreadable body", http.StatusBadRequest)
			return
		}

		event, err := webhook.Verify(payload, r.Header, "whsec_...")
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}

		switch event.Type {
		case webhook.EmailBounced:
			log.Printf("%s bounced: %s", event.Email.To, event.Email.Bounce.Message)
		case webhook.ContactCreated:
			log.Printf("new contact %s", event.Contact.Email)
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
