package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"

	"github.com/itsmeabde/wacloudapi/flows"
)

func main() {
	pemStr := os.Getenv("WA_FLOWS_PRIVATE_KEY_PEM")
	passphrase := os.Getenv("WA_FLOWS_KEY_PASSPHRASE")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	var pemBytes []byte
	if pemStr != "" {
		pemBytes = []byte(pemStr)
	} else {
		// Generate an ephemeral RSA 2048-bit private key in PEM format for local demonstration
		log.Println("WA_FLOWS_PRIVATE_KEY_PEM not set; generating ephemeral RSA key for demo...")
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatalf("Failed to generate RSA key: %v", err)
		}
		derBytes := x509.MarshalPKCS1PrivateKey(privKey)
		pemBytes = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: derBytes,
		})
	}

	var h *flows.Handler
	var err error
	if passphrase != "" {
		h, err = flows.NewHandlerFromPEM(pemBytes, passphrase)
	} else {
		h, err = flows.NewHandlerFromPEM(pemBytes)
	}
	if err != nil {
		log.Fatalf("Failed to initialize flows handler: %v", err)
	}

	// 1. Handle screen navigation/data exchange for "APPOINTMENT"
	h.HandleScreen("APPOINTMENT", func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		log.Printf("[Flow APPOINTMENT] Action: %s, FlowToken: %s, Data: %+v\n", req.Action, req.FlowToken, req.Data)

		return &flows.Response{
			Screen: "CONFIRMATION",
			Data: map[string]interface{}{
				"appointment_id": "APT-2026-9988",
				"date":           "2026-09-01",
				"time":           "10:00 AM",
				"doctor":         "Dr. Jane Doe",
				"department":     req.Data["department"],
				"status":         "confirmed",
			},
		}, nil
	})

	// 2. Handle screen for "SURVEY"
	h.HandleScreen("SURVEY", func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		log.Printf("[Flow SURVEY] Action: %s, Data: %+v\n", req.Action, req.Data)
		return &flows.Response{
			Screen: "THANK_YOU",
			Data: map[string]interface{}{
				"message": "Terima kasih atas masukan Anda!",
			},
		}, nil
	})

	// 3. Handle Flow Health Check (ping)
	h.OnPing(func(ctx context.Context, req *flows.Request) (*flows.Response, error) {
		log.Println("[Flow Health Check] Ping received")
		return &flows.Response{
			Data: map[string]interface{}{
				"status": "active",
			},
		}, nil
	})

	// 4. Handle internal errors during decryption / processing
	h.OnError(func(ctx context.Context, err error) {
		log.Printf("[Flow Error]: %v\n", err)
	})

	http.Handle("/flows", h)
	log.Printf("WhatsApp Flows Data Endpoint server listening on :%s/flows...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
