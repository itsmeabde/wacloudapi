package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/itsmeabde/wacloudapi/webhook"
)

func main() {
	verifyToken := os.Getenv("WA_VERIFY_TOKEN")
	appSecret := os.Getenv("WA_APP_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h := webhook.NewHandler(verifyToken, appSecret)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("[Text Message] From: %s, Body: %s\n", msg.From, msg.Text.Body)
		return nil
	})

	h.OnMediaMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("[Media Message] From: %s, Type: %s, MediaID: %s\n", msg.From, msg.Type, msg.MediaID())
		return nil
	})

	h.OnInteractiveMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Interactive.ButtonReply != nil {
			fmt.Printf("[Button Reply] From: %s, Title: %s, ID: %s\n", msg.From, msg.Interactive.ButtonReply.Title, msg.Interactive.ButtonReply.ID)
		}
		return nil
	})

	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		fmt.Printf("[Status Update] ID: %s, Status: %s, Recipient: %s\n", status.ID, status.Status, status.RecipientID)
		return nil
	})

	h.OnError(func(ctx context.Context, err error) {
		log.Printf("[Webhook Error]: %v\n", err)
	})

	http.Handle("/webhook", h)
	log.Printf("Webhook server running on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
