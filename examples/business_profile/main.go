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

	if token == "" || phoneID == "" {
		log.Fatal("WA_ACCESS_TOKEN and WA_PHONE_NUMBER_ID must be set")
	}

	client := wacloudapi.New(token, phoneID)

	// Get Profile
	profile, err := client.BusinessProfile.Get(context.Background())
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}
	fmt.Printf("Profile About: %s, Vertical: %s\n", profile.About, profile.Vertical)

	// Update Profile
	err = client.BusinessProfile.Update(context.Background(), &wacloudapi.UpdateBusinessProfileRequest{
		About:       "Official Customer Support",
		Description: "Contact us for inquiries and help.",
		Vertical:    wacloudapi.VerticalRetail,
		Websites:    []string{"https://example.com"},
	})
	if err != nil {
		log.Fatalf("Failed to update profile: %v", err)
	}
	fmt.Println("Profile successfully updated!")
}
