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
	recipient := os.Getenv("WA_RECIPIENT_PHONE")

	if token == "" || phoneID == "" || recipient == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_RECIPIENT_PHONE must be set")
	}

	client := wacloudapi.New(token, phoneID, wacloudapi.WithRetry(3))

	// Send Text Message
	res, err := client.Messages.SendText(context.Background(), recipient, "Halo dari wacloudapi Go SDK!")
	if err != nil {
		log.Fatalf("Error sending text: %v", err)
	}
	fmt.Printf("Message sent! ID: %s\n", res.Messages[0].ID)

	// Send Interactive Button Message
	interactive := &wacloudapi.InteractiveMessage{
		Type: wacloudapi.InteractiveTypeButton,
		Body: wacloudapi.InteractiveBody{Text: "Apakah Anda ingin melanjutkan konfirmasi?"},
		Action: wacloudapi.InteractiveAction{
			Buttons: []wacloudapi.ButtonAction{
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_yes", Title: "Ya, Setuju"}},
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_no", Title: "Batalkan"}},
			},
		},
	}
	resBtn, err := client.Messages.SendInteractive(context.Background(), recipient, interactive)
	if err != nil {
		log.Fatalf("Error sending interactive: %v", err)
	}
	fmt.Printf("Interactive message sent! ID: %s\n", resBtn.Messages[0].ID)
}
