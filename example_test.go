package bankid_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jfernstad/bankid"
)

func ExampleAuth() {
	// Initialize messages for hint codes (optional)
	_, _ = bankid.NewMessages("en")

	// Set up the environment using certificates
	// Normally you'd read these from files or a secret manager
	caCertPath := "CA/test.crt"
	rpCertPath := "rp/bankid_rp_test.crt"
	rpKeyPath := "rp/bankid_rp_test.key"

	env, err := bankid.NewEnvironment(
		bankid.TestBaseURL,
		caCertPath,
		rpCertPath,
		rpKeyPath,
		bankid.WithTimeout(15*time.Second), // Custom timeout
	)
	if err != nil {
		log.Fatalf("Could not create environment: %v", err)
	}

	ctx := context.Background()

	// Start an authentication request
	req := &bankid.AuthRequest{
		EndUserIP:       "127.0.0.1",
		UserVisibleData: "Login to MyService", // Automatically base64-encoded
	}

	rsp, err := bankid.Auth(ctx, env, req)
	if err != nil {
		// Inspect specific BankID errors
		if errRsp, ok := bankid.IsErrorResponse(err); ok {
			log.Fatalf("BankID Error: %s - %s", errRsp.ErrorCode, errRsp.Details)
		}
		log.Fatalf("Could not start auth: %v", err)
	}

	fmt.Printf("Started auth. OrderRef: %s\n", rsp.OrderRef)

	// Poll the collect endpoint every 2 seconds
	// collectRsp, err := bankid.Collect(ctx, env, rsp.OrderRef)
}
