package bankid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateQRData generates the animated QR code data string for BankID v6.0.
//
// The QR code must be updated every second to create the animated effect required
// by the BankID Secure Start specification.
//
// Algorithm:
//
//	qrAuthCode = HMAC-SHA256(qrStartSecret, qrTime)
//	qrData     = "bankid." + qrStartToken + "." + qrTime + "." + qrAuthCode
//
// Parameters:
//   - qrStartToken: from the Auth/Sign response
//   - qrStartSecret: from the Auth/Sign response
//   - elapsed: time since the order was initiated
//
// The caller is responsible for rendering the QR data string into an actual
// QR code image (e.g. using a QR library). This keeps the bankid library
// dependency-free.
func GenerateQRData(qrStartToken, qrStartSecret string, elapsed time.Duration) string {
	qrTime := fmt.Sprintf("%d", int(elapsed.Seconds()))

	mac := hmac.New(sha256.New, []byte(qrStartSecret))
	mac.Write([]byte(qrTime))
	qrAuthCode := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("bankid.%s.%s.%s", qrStartToken, qrTime, qrAuthCode)
}
