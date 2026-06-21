package bankid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateQRData_Format(t *testing.T) {
	token := "67df3917-fa0d-44e5-b327-edcc928297f8"
	secret := "d28db9a7-4cde-429e-a983-359be676944c"

	data := GenerateQRData(token, secret, 0)

	// Should have format: bankid.<token>.<time>.<authcode>
	parts := strings.Split(data, ".")
	assert.Equal(t, 4, len(parts), "QR data should have 4 dot-separated parts")
	assert.Equal(t, "bankid", parts[0])
	assert.Equal(t, token, parts[1])
	assert.Equal(t, "0", parts[2])
	// parts[3] is the HMAC-SHA256 hex digest
	assert.Len(t, parts[3], 64, "HMAC-SHA256 hex digest should be 64 characters")
}

func TestGenerateQRData_TimeProgression(t *testing.T) {
	token := "67df3917-fa0d-44e5-b327-edcc928297f8"
	secret := "d28db9a7-4cde-429e-a983-359be676944c"

	data0 := GenerateQRData(token, secret, 0*time.Second)
	data1 := GenerateQRData(token, secret, 1*time.Second)
	data2 := GenerateQRData(token, secret, 2*time.Second)

	// Each second should produce a different QR code
	assert.NotEqual(t, data0, data1)
	assert.NotEqual(t, data1, data2)
	assert.NotEqual(t, data0, data2)

	// Time component should change
	assert.Contains(t, data0, ".0.")
	assert.Contains(t, data1, ".1.")
	assert.Contains(t, data2, ".2.")
}

func TestGenerateQRData_CorrectHMAC(t *testing.T) {
	token := "67df3917-fa0d-44e5-b327-edcc928297f8"
	secret := "d28db9a7-4cde-429e-a983-359be676944c"

	// Manually compute expected HMAC for time=5
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("5"))
	expectedAuthCode := hex.EncodeToString(mac.Sum(nil))

	data := GenerateQRData(token, secret, 5*time.Second)
	parts := strings.Split(data, ".")

	assert.Equal(t, expectedAuthCode, parts[3])
}

func TestGenerateQRData_Deterministic(t *testing.T) {
	token := "67df3917-fa0d-44e5-b327-edcc928297f8"
	secret := "d28db9a7-4cde-429e-a983-359be676944c"

	// Same inputs should always produce the same output
	data1 := GenerateQRData(token, secret, 10*time.Second)
	data2 := GenerateQRData(token, secret, 10*time.Second)
	assert.Equal(t, data1, data2)
}
