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

	// Create QR Code
	qr, err := client.QRCodes.Create(context.Background(), &wacloudapi.CreateQRCodeRequest{
		PrefilledMessage: "Halo, saya ingin bertanya tentang produk.",
		ImageFormat:      wacloudapi.QRCodeImagePNG,
	})
	if err != nil {
		log.Fatalf("Failed to create QR code: %v", err)
	}
	fmt.Printf("QR Code created! Code: %s, DeepLink: %s\n", qr.Code, qr.DeepLinkURL)

	// List QR Codes
	list, err := client.QRCodes.List(context.Background())
	if err != nil {
		log.Fatalf("Failed to list QR codes: %v", err)
	}
	fmt.Printf("Total QR Codes: %d\n", len(list.Data))
}
