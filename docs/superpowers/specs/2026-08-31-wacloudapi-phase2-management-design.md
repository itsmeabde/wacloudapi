# Product Requirement Document (PRD) & Technical Design: WhatsApp Business Management APIs (Phase 2)

**Module**: `github.com/itsmeabde/wacloudapi`  
**Go Version**: `go 1.22+`  
**Status**: Approved Spec  
**Author**: @itsmeabde & Antigravity  
**Date**: 2026-08-31  

---

## 1. Executive Summary & Goals

Melanjutkan fondasi v1.0 (`wacloudapi` Messaging, Media, dan Webhooks), **Fase 2** menambahkan dukungan penuh untuk **WhatsApp Business Management APIs** resmi dari Meta Graph API (v21.0+):
1. **Business Profile Management** (`client.BusinessProfile`)
2. **Message Template Management** (`client.Templates`)
3. **Phone Numbers & 2-Step Verification** (`client.PhoneNumbers`)
4. **Message QR Code Management** (`client.QRCodes`)

Tetap mengusung prinsip utama:
- **Zero External Dependencies** (100% Go Standard Library).
- **Idiomatic Go & Functional Options**.
- **Resilient & Type-Safe**.

---

## 2. Scope & Feature Requirements

### 2.1 WABA ID & Configuration
- Menambahkan `WithWABAID(wabaID string)` pada `Config`.
- Helper `ErrMissingWABAID` jika pemanggilan endpoint berbasis WABA tidak menyediakan WABA ID di config maupun di request.

### 2.2 Business Profile API (`client.BusinessProfile`)
- **Get Profile**: `Get(ctx context.Context, fields ...string) (*BusinessProfile, error)`
- **Update Profile**: `Update(ctx context.Context, req *UpdateBusinessProfileRequest) error`
- **Fields**: About, Address, Description, Email, Profile Picture URL, Websites (hingga 2 URLs), Vertical/Industri (dengan enum typed string).

### 2.3 Message Template Management API (`client.Templates`)
- **Create**: `Create(ctx context.Context, req *CreateTemplateRequest) (*CreateTemplateResponse, error)`
- **List**: `List(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error)` dengan cursor pagination (`HasNext()`, `Next()`) dan filter (`Category`, `Status`, `Name`, `Limit`).
- **Get**: `Get(ctx context.Context, templateID string) (*TemplateDetails, error)`
- **Update**: `Update(ctx context.Context, templateID string, req *UpdateTemplateRequest) (*UpdateTemplateResponse, error)`
- **Delete**: `DeleteByName(ctx context.Context, name string, wabaID ...string) error` dan `DeleteByID(ctx context.Context, templateID string) error`

### 2.4 Phone Numbers & Two-Step Verification API (`client.PhoneNumbers`)
- **List**: `List(ctx context.Context, wabaID ...string) (*ListPhoneNumbersResponse, error)`
- **Get**: `Get(ctx context.Context, phoneNumberID string) (*PhoneNumberDetails, error)`
- **Request Code**: `RequestCode(ctx context.Context, phoneNumberID string, codeMethod CodeMethod, language string) error` (SMS atau VOICE).
- **Verify Code**: `VerifyCode(ctx context.Context, phoneNumberID string, code string) error`
- **Register**: `Register(ctx context.Context, phoneNumberID, pin string, cert ...string) error`
- **Deregister**: `Deregister(ctx context.Context, phoneNumberID string) error`
- **SetTwoStepPIN**: `SetTwoStepPIN(ctx context.Context, phoneNumberID, pin string) error`

### 2.5 Message QR Code Management API (`client.QRCodes`)
- **Create**: `Create(ctx context.Context, req *CreateQRCodeRequest) (*QRCodeDetails, error)`
- **List**: `List(ctx context.Context) (*ListQRCodesResponse, error)`
- **Get**: `Get(ctx context.Context, qrCodeID string) (*QRCodeDetails, error)`
- **Update**: `Update(ctx context.Context, qrCodeID, prefilledMessage string) (*QRCodeDetails, error)`
- **Delete**: `Delete(ctx context.Context, qrCodeID string) error`
- **Download Image**: `DownloadImage(ctx context.Context, qrCodeID string) ([]byte, error)`

---

## 3. Package Structure

```text
github.com/itsmeabde/wacloudapi/
├── profile.go               # BusinessProfileService implementation
├── profile_models.go        # Structs & Enums untuk Business Profile
├── templates.go             # TemplatesService implementation
├── templates_models.go      # Structs & Enums untuk Template Management
├── phone_numbers.go         # PhoneNumbersService implementation
├── phone_numbers_models.go  # Structs & Enums untuk Phone Numbers & 2FA
├── qr_codes.go              # QRCodesService implementation
├── qr_codes_models.go       # Structs & Enums untuk Message QR Codes
└── examples/
    ├── business_profile/    # Contoh get & update profile
    ├── templates_management/# Contoh create, list, & delete templates
    └── qr_codes/            # Contoh create, list, & download QR codes
```
