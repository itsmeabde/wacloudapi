package wacloudapi

type CodeMethod string

const (
	CodeMethodSMS   CodeMethod = "SMS"
	CodeMethodVoice CodeMethod = "VOICE"
)

type PhoneNumberDetails struct {
	ID                      string `json:"id"`
	DisplayPhoneNumber      string `json:"display_phone_number"`
	DisplayPhoneNumberClean string `json:"display_phone_number_clean,omitempty"`
	VerifiedName            string `json:"verified_name"`
	QualityRating           string `json:"quality_rating"`
	CodeVerificationStatus  string `json:"code_verification_status,omitempty"`
	EligibilityForAPIStatus string `json:"eligibility_for_api_business_verification,omitempty"`
	NameStatus              string `json:"name_status,omitempty"`
	NewNameStatus           string `json:"new_name_status,omitempty"`
	Status                  string `json:"status,omitempty"`
}

type ListPhoneNumbersResponse struct {
	Data   []PhoneNumberDetails `json:"data"`
	Paging *Paging              `json:"paging,omitempty"`
}
