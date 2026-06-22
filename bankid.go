package bankid

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// API Constants
const (
	ProductionBaseURL    string = "https://appapi2.bankid.com"
	TestBaseURL          string = "https://appapi2.test.bankid.com"
	APIVersion           string = "/rp/v6.0"
	AuthEndpoint         string = "/auth"
	SignEndpoint         string = "/sign"
	CollectEndpoint      string = "/collect"
	CancelEndpoint       string = "/cancel"
	PhoneAuthEndpoint    string = "/phone/auth"
	PhoneSignEndpoint    string = "/phone/sign"
)

// Environmenter helps setup requests to the BankID API
type Environmenter interface {
	NewClient() *http.Client
	NewRequest(ctx context.Context, endpoint string, body interface{}) (*http.Request, error)
}

type environment struct {
	baseURL      string
	clientConfig *tls.Config
	client       *http.Client
}

// NewEnvironment sets up the certificates and URLs needed to identify ourselves with the BankID service.
// Certificate files are loaded from the provided file paths.
// For loading certificates from memory (e.g. Vault, K8s secrets), use NewEnvironmentFromBytes.
func NewEnvironment(baseURL string, caPath string, rpCertPath string, rpKeyPath string) (Environmenter, error) {
	ca, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("could not load CA Certificate: %s", err.Error())
	}

	rpCert, err := os.ReadFile(rpCertPath)
	if err != nil {
		return nil, fmt.Errorf("could not load RP Certificate: %s", err.Error())
	}

	rpKey, err := os.ReadFile(rpKeyPath)
	if err != nil {
		return nil, fmt.Errorf("could not load RP Key: %s", err.Error())
	}

	return NewEnvironmentFromBytes(baseURL, ca, rpCert, rpKey)
}

// NewEnvironmentFromBytes sets up the certificates and URLs needed to identify ourselves
// with the BankID service, using raw PEM-encoded certificate bytes.
// This is useful when loading certificates from environment variables, Vault, K8s secrets, etc.
func NewEnvironmentFromBytes(baseURL string, ca []byte, rpCert []byte, rpKey []byte) (Environmenter, error) {
	keyPair, err := tls.X509KeyPair(rpCert, rpKey)
	if err != nil {
		return nil, fmt.Errorf("could not load RP Keypair: %s", err.Error())
	}

	caPool := x509.NewCertPool()

	if !caPool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("could not append CA Certificate to pool. Invalid certificate?")
	}

	clientCfg := tls.Config{
		MinVersion:   tls.VersionTLS12, // BankID requires TLS 1.2+
		Certificates: []tls.Certificate{keyPair},
		ClientCAs:    caPool,
		RootCAs:      caPool,
	}

	env := &environment{
		baseURL:      baseURL,
		clientConfig: &clientCfg,
	}
	env.client = env.newClient()
	return env, nil
}

// NewRequest creates an HTTP request for the given endpoint and body
func (e *environment) NewRequest(ctx context.Context, endpoint string, body interface{}) (*http.Request, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	bodyReader := strings.NewReader(string(requestBody))
	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+APIVersion+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

// NewClient returns the shared http.Client with our TLS Config.
// The client is created once and reused to benefit from connection pooling.
func (e *environment) NewClient() *http.Client {
	return e.client
}

// newClient creates the underlying http.Client with TLS and timeout configuration.
func (e *environment) newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig:     e.clientConfig,
			TLSHandshakeTimeout: 5 * time.Second,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
}
