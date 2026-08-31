package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/itsmeabde/wacloudapi"
)

func main() {
	token := os.Getenv("WA_ACCESS_TOKEN")
	phoneID := os.Getenv("WA_PHONE_NUMBER_ID")
	wabaID := os.Getenv("WA_WABA_ID")

	if token == "" || phoneID == "" || wabaID == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_WABA_ID must be set")
	}

	client := wacloudapi.New(token, phoneID, wacloudapi.WithWABAID(wabaID))

	// List Templates
	list, err := client.Templates.List(context.Background(), &wacloudapi.ListTemplatesRequest{
		Limit: 5,
	})
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("Found %d templates:\n", len(list.Data))
	for _, tpl := range list.Data {
		fmt.Printf("- %s [%s] (%s)\n", tpl.Name, tpl.Category, tpl.Status)
	}
}
