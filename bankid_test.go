package bankid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidEnvironment(t *testing.T) {
	// Bad CA file
	_, err := NewEnvironment("BASE_URL", "INVALID_CA", "", "")
	assert.NotNil(t, err)

	// Bad Cert/Key files
	_, err = NewEnvironment("BASE_URL", "./CA/test.crt", "INVALID_RP_CERT", "INVALID_RP_KEY")
	assert.NotNil(t, err)

	// Wrong CA file
	_, err = NewEnvironment("BASE_URL", "./rp/bankid_rp_test.key", "./rp/bankid_rp_test.crt", "./rp/bankid_rp_test.key")
	assert.NotNil(t, err)
}

func TestValidEnvironment(t *testing.T) {
	// Cert Files OK
	_, err := NewEnvironment("BASE_URL", "./CA/test.crt", "./rp/bankid_rp_test.crt", "./rp/bankid_rp_test.key")
	assert.Nil(t, err)
}

func TestRequestsBad(t *testing.T) {
	env, err := NewEnvironment("BASE_URL:🤣", "./CA/test.crt", "./rp/bankid_rp_test.crt", "./rp/bankid_rp_test.key")
	assert.Nil(t, err)
	assert.NotNil(t, env)

	ctx := context.Background()

	// Bad body
	var invalidBodyType chan int
	_, err = env.NewRequest(ctx, "endpoint", invalidBodyType)
	assert.NotNil(t, err)

	// Bad schema
	_, err = env.NewRequest(ctx, "endpoint", "")
	assert.NotNil(t, err)
}

func TestRequestsOK(t *testing.T) {
	env, err := NewEnvironment(ProductionBaseURL, "./CA/test.crt", "./rp/bankid_rp_test.crt", "./rp/bankid_rp_test.key")
	assert.Nil(t, err)
	assert.NotNil(t, env)

	ctx := context.Background()

	// All A OK
	req, err := env.NewRequest(ctx, "endpoint", "")
	assert.Nil(t, err)
	assert.NotNil(t, req)

	// Verify v6.0 API path
	assert.Contains(t, req.URL.String(), "/rp/v6.0")

	// Verify content-type header
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

	// Initiate client
	client := env.NewClient()
	assert.NotNil(t, client)
}
