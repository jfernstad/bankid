package bankid

import "fmt"

// Collect response statuses
const (
	OrderPending  = "pending"
	OrderFailed   = "failed"
	OrderComplete = "complete"
)

// HintCodes - for Pending and Failed statuses
const (
	PendOutstandingTransaction = "outstandingTransaction"
	PendNoClient               = "noClient"
	PendStarted                = "started"
	PendUserSign               = "userSign"
	PendUserMrtd               = "userMrtd" // New in v6.0: user is scanning ID document

	FailExpiredTransaction = "expiredTransaction"
	FailCertificateErr     = "certificateErr"
	FailUserCancel         = "userCancel"
	FailCancelled          = "cancelled"
	FailStartFailed        = "startFailed"
)

// CallInitiator indicates who initiated the phone call (for /phone/* endpoints)
const (
	CallInitiatorUser = "user"
	CallInitiatorRP   = "RP"
)

// Requirement specifies constraints on the authentication/signing order.
type Requirement struct {
	PinCode             *bool    `json:"pinCode,omitempty"`
	MRTD                *bool    `json:"mrtd,omitempty"`
	CardReader          string   `json:"cardReader,omitempty"`
	CertificatePolicies []string `json:"certificatePolicies,omitempty"`
	PersonalNumber      string   `json:"personalNumber,omitempty"`
}

// AppContext provides device information when authenticating from a native app.
type AppContext struct {
	AppIdentifier   string `json:"appIdentifier,omitempty"`
	DeviceOS        string `json:"deviceOS,omitempty"`
	DeviceIdentifier string `json:"deviceIdentifier,omitempty"`
	DeviceModelName string `json:"deviceModelName,omitempty"`
}

// WebContext provides browser information when authenticating from a web browser.
type WebContext struct {
	DeviceIdentifier string `json:"deviceIdentifier,omitempty"`
	ReferringDomain  string `json:"referringDomain,omitempty"`
	UserAgent        string `json:"userAgent,omitempty"`
}

// AuthRequest initiates an authentication order.
// In v6.0, personalNumber is no longer accepted here.
// The user must be identified via QR code (another device) or autostart (same device).
type AuthRequest struct {
	EndUserIP             string       `json:"endUserIp"`
	Requirement           *Requirement `json:"requirement,omitempty"`
	UserVisibleData       string       `json:"userVisibleData,omitempty"`
	UserVisibleDataFormat string       `json:"userVisibleDataFormat,omitempty"` // "simpleMarkdownV1"
	UserNonVisibleData    string       `json:"userNonVisibleData,omitempty"`
	ReturnRisk            *bool        `json:"returnRisk,omitempty"`
	ReturnURL             string       `json:"returnUrl,omitempty"`  // URL for same-device redirect after completion
	App                   *AppContext  `json:"app,omitempty"`        // Native app device context
	Web                   *WebContext  `json:"web,omitempty"`        // Browser device context
}

// SignRequest initiates a signing order.
// userVisibleData is required for signing (the text to sign).
type SignRequest struct {
	EndUserIP             string       `json:"endUserIp"`
	UserVisibleData       string       `json:"userVisibleData"`
	Requirement           *Requirement `json:"requirement,omitempty"`
	UserVisibleDataFormat string       `json:"userVisibleDataFormat,omitempty"` // "simpleMarkdownV1"
	UserNonVisibleData    string       `json:"userNonVisibleData,omitempty"`
	ReturnRisk            *bool        `json:"returnRisk,omitempty"`
	ReturnURL             string       `json:"returnUrl,omitempty"`  // URL for same-device redirect after completion
	App                   *AppContext  `json:"app,omitempty"`        // Native app device context
	Web                   *WebContext  `json:"web,omitempty"`        // Browser device context
}

// PhoneAuthRequest initiates authentication during a phone call.
// This is the only v6.0 endpoint that accepts personalNumber directly.
type PhoneAuthRequest struct {
	PersonalNumber        string       `json:"personalNumber"`
	CallInitiator         string       `json:"callInitiator"` // "user" or "RP"
	Requirement           *Requirement `json:"requirement,omitempty"`
	UserVisibleData       string       `json:"userVisibleData,omitempty"`
	UserVisibleDataFormat string       `json:"userVisibleDataFormat,omitempty"`
	UserNonVisibleData    string       `json:"userNonVisibleData,omitempty"`
	ReturnRisk            *bool        `json:"returnRisk,omitempty"`
}

// PhoneSignRequest initiates signing during a phone call.
type PhoneSignRequest struct {
	PersonalNumber        string       `json:"personalNumber"`
	CallInitiator         string       `json:"callInitiator"` // "user" or "RP"
	UserVisibleData       string       `json:"userVisibleData"`
	Requirement           *Requirement `json:"requirement,omitempty"`
	UserVisibleDataFormat string       `json:"userVisibleDataFormat,omitempty"`
	UserNonVisibleData    string       `json:"userNonVisibleData,omitempty"`
	ReturnRisk            *bool        `json:"returnRisk,omitempty"`
}

// CollectRequest polls for the status of an ongoing order.
type CollectRequest struct {
	OrderRef string `json:"orderRef"`
}

// CancelRequest cancels an ongoing order.
type CancelRequest struct {
	OrderRef string `json:"orderRef"`
}

// Response is returned by Auth and Sign.
type Response struct {
	OrderRef       string `json:"orderRef"`       // UUID for polling via /collect
	AutoStartToken string `json:"autoStartToken"` // Used to launch BankID on the same device
	QRStartToken   string `json:"qrStartToken"`   // New in v6.0: used to build animated QR codes
	QRStartSecret  string `json:"qrStartSecret"`  // New in v6.0: HMAC key for QR code generation
}

// PhoneResponse is returned by PhoneAuth and PhoneSign.
// Phone endpoints only return an orderRef (no autostart/QR tokens).
type PhoneResponse struct {
	OrderRef string `json:"orderRef"` // UUID for polling via /collect
}

// ErrorResponse is returned when the API call fails.
type ErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Details   string `json:"details"`
}

// Error implements the error interface.
func (e ErrorResponse) Error() string {
	return fmt.Sprintf("failed with code: %s. '%s'", e.ErrorCode, e.Details)
}

// CollectResponse is returned by the /collect endpoint.
type CollectResponse struct {
	OrderRef       string      `json:"orderRef"`
	Status         string      `json:"status"`
	HintCode       string      `json:"hintCode,omitempty"`       // Pending and Failed orders only
	CompletionData *Completion `json:"completionData,omitempty"` // Complete orders only
}

// Completion contains the user data, device info, and signatures for a completed order.
type Completion struct {
	User            User    `json:"user"`
	Device          Device  `json:"device"`
	BankIDIssueDate string  `json:"bankIdIssueDate,omitempty"` // New in v6.0: ISO 8601 date
	StepUp          *StepUp `json:"stepUp,omitempty"`          // New in v6.0
	Signature       string  `json:"signature,omitempty"`       // base64 encoded XML signature
	OCSPResponse    string  `json:"ocspResponse,omitempty"`
	Risk            string  `json:"risk,omitempty"` // New in v6.0: "low", "moderate", or "high"
}

// User contains the authenticated user's identity information.
type User struct {
	PersonalNumber string `json:"personalNumber"` // e.g "197001010000"
	Name           string `json:"name"`
	GivenName      string `json:"givenName"`
	Surname        string `json:"surname"`
}

// Device contains information about the device used for the order.
type Device struct {
	IPAddress string `json:"ipAddress"`        // e.g "192.168.0.1"
	UHI       string `json:"uhi,omitempty"`    // New in v6.0: unique hardware identifier
}

// StepUp contains information about additional security steps performed.
// New in v6.0.
type StepUp struct {
	MRTD bool `json:"mrtd"` // true if MRTD check was performed and passed
}
