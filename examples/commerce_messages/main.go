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

	catalogID := os.Getenv("WA_CATALOG_ID")
	if catalogID == "" {
		catalogID = "catalog_123"
	}
	productID := os.Getenv("WA_PRODUCT_RETAILER_ID")
	if productID == "" {
		productID = "prod_xyz"
	}
	flowID := os.Getenv("WA_FLOW_ID")
	if flowID == "" {
		flowID = "flow_123"
	}

	client := wacloudapi.New(token, phoneID, wacloudapi.WithRetry(3))
	ctx := context.Background()

	// 1. Send Single Product Message
	fmt.Println("Sending Single Product Message...")
	spRes, err := client.Messages.SendSingleProduct(ctx, recipient, catalogID, productID, "Lihat penawaran produk spesial kami hari ini!")
	if err != nil {
		log.Printf("Error sending single product message: %v\n", err)
	} else {
		fmt.Printf("Single Product message sent! Message ID: %s\n", spRes.Messages[0].ID)
	}

	// 2. Send Multi-Product Message (MPM)
	fmt.Println("Sending Multi-Product Message...")
	mpmReq := &wacloudapi.MultiProductRequest{
		CatalogID:   catalogID,
		HeaderTitle: "Katalog Promo Mingguan",
		BodyText:    "Temukan produk terbaik dengan diskon menarik di katalog kami:",
		FooterText:  "Promo berlaku hingga akhir pekan",
		Sections: []wacloudapi.ProductSection{
			{
				Title: "Kategori Elektronik",
				ProductItems: []wacloudapi.ProductItem{
					{ProductRetailerID: "prod_phone_01"},
					{ProductRetailerID: "prod_laptop_02"},
				},
			},
			{
				Title: "Kategori Aksesoris",
				ProductItems: []wacloudapi.ProductItem{
					{ProductRetailerID: "prod_headset_03"},
				},
			},
		},
	}
	mpmRes, err := client.Messages.SendMultiProduct(ctx, recipient, mpmReq)
	if err != nil {
		log.Printf("Error sending multi-product message: %v\n", err)
	} else {
		fmt.Printf("Multi-Product message sent! Message ID: %s\n", mpmRes.Messages[0].ID)
	}

	// 3. Send WhatsApp Flow Message
	fmt.Println("Sending WhatsApp Flow Message...")
	flowReq := &wacloudapi.FlowMessageRequest{
		FlowID:             flowID,
		FlowToken:          "token_appointment_001",
		FlowCTA:            "Jadwalkan Janji Temu",
		FlowAction:         "navigate",
		FlowMode:           "draft",
		FlowMessageVersion: "3",
		BodyText:           "Silakan klik tombol di bawah untuk membuat janji temu baru.",
		FooterText:         "Layanan 24/7",
		ActionPayload: &wacloudapi.FlowActionPayload{
			Screen: "APPOINTMENT",
			Data: map[string]interface{}{
				"department": "consultation",
			},
		},
	}
	flowRes, err := client.Messages.SendFlow(ctx, recipient, flowReq)
	if err != nil {
		log.Printf("Error sending flow message: %v\n", err)
	} else {
		fmt.Printf("Flow message sent! Message ID: %s\n", flowRes.Messages[0].ID)
	}
}
