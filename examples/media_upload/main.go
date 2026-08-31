package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/itsmeabde/wacloudapi"
)

func main() {
	token := os.Getenv("WA_ACCESS_TOKEN")
	phoneID := os.Getenv("WA_PHONE_NUMBER_ID")
	recipient := os.Getenv("WA_RECIPIENT_PHONE")

	if token == "" || phoneID == "" || recipient == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_RECIPIENT_PHONE must be set")
	}

	client := wacloudapi.New(token, phoneID)

	// Upload in-memory buffer
	buf := bytes.NewReader([]byte("dummy content"))
	uploaded, err := client.Media.Upload(context.Background(), "notes.txt", buf, "text/plain")
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
	fmt.Printf("Media uploaded! ID: %s\n", uploaded.ID)

	// Send Document using Media ID
	res, err := client.Messages.SendDocument(context.Background(), recipient,
		wacloudapi.MediaByID(uploaded.ID),
		wacloudapi.WithCaption("Berikut catatan terlampir"),
		wacloudapi.WithFilename("notes.txt"),
	)
	if err != nil {
		log.Fatalf("Send document failed: %v", err)
	}
	fmt.Printf("Document message sent! ID: %s\n", res.Messages[0].ID)
}
