package bankid

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	handler func(w http.ResponseWriter, r *http.Request) // Reset this one for every test
	server  *httptest.Server
}

func (t *testEnv) NewClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, t.server.Listener.Addr().String())
			},
			TLSHandshakeTimeout: 5 * time.Second,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
}
func (te *testEnv) NewRequest(ctx context.Context, endpoint string, body interface{}) (*http.Request, error) {
	// Close any previous server to prevent leaks
	if te.server != nil {
		te.server.Close()
	}
	te.server = httptest.NewServer(http.HandlerFunc(te.handler))

	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	bodyReader := strings.NewReader(string(requestBody))
	req, err := http.NewRequestWithContext(ctx, "POST", "http://"+te.server.URL+"/"+APIVersion+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	return req, nil

}

type tt struct {
	name    string
	handler func(w http.ResponseWriter, r *http.Request)
	assert  func(t *testing.T, resp interface{}, err error)
}

//
// Auth, Sign, Collect, PhoneAuth, PhoneSign tests for v6.0
//

func TestSignAuthCollect_v6(t *testing.T) {
	testFunctions := []*tt{
		{
			name: "Expected: Successful, w/ correct response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				authRsp := Response{
					OrderRef:       "131daac9-16c6-4618-beb0-365768f37288",
					AutoStartToken: "dbbee61c-357b-4fd8-b103-392eed10be7a",
					QRStartToken:   "67df3917-fa0d-44e5-b327-edcc928297f8",
					QRStartSecret:  "d28db9a7-4cde-429e-a983-359be676944c",
				}
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(&authRsp)
			},
			assert: func(t *testing.T, resp interface{}, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "Expected: Successful, w/ incorrect response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write(nil) // Empty and invalid
			},
			assert: func(t *testing.T, resp interface{}, err error) {
				assert.NotNil(t, err)
				assert.Empty(t, resp)
			},
		},
		{
			name: "Expected: Fail, w/ correct response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				errRsp := ErrorResponse{} // Empty but valid
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(&errRsp)
			},
			assert: func(t *testing.T, resp interface{}, err error) {
				assert.NotNil(t, err)
				assert.NotEmpty(t, err.Error())
				assert.Empty(t, resp)
			},
		},
		{
			name: "Expected: Fail, w/ incorrect response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(400)
				w.Write(nil) // Empty and invalid
			},
			assert: func(t *testing.T, resp interface{}, err error) {
				assert.NotNil(t, err)
				assert.NotEmpty(t, err.Error())
				assert.Empty(t, resp)
			},
		},
		{
			name: "Expected: Fail, unexpected status codes return an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(101)
				w.Write(nil)
			},
			assert: func(t *testing.T, resp interface{}, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "unexpected HTTP status")
				assert.Empty(t, resp)
			},
		},
	}

	env := &testEnv{}
	ctx := context.Background()

	// Auth - v6.0 (no personal number)
	for _, at := range testFunctions {
		env.handler = at.handler
		t.Logf("Auth: %s", at.name)
		resp, err := Auth(ctx, env, &AuthRequest{
			EndUserIP: "127.0.0.1",
		})
		at.assert(t, resp, err)
	}

	// Sign - v6.0 (no personal number)
	for _, at := range testFunctions {
		env.handler = at.handler
		t.Logf("Sign: %s", at.name)
		resp, err := Sign(ctx, env, &SignRequest{
			EndUserIP:       "127.0.0.1",
			UserVisibleData: "Test signing data",
		})
		at.assert(t, resp, err)
	}

	// Collecting status
	for _, at := range testFunctions {
		env.handler = at.handler
		t.Logf("Collect: %s", at.name)
		resp, err := Collect(ctx, env, "dbbee61c-357b-4fd8-b103-392eed10be7a")
		at.assert(t, resp, err)
	}

	env.server.Close()
}

func TestPhoneAuth_v6(t *testing.T) {
	env := &testEnv{}
	ctx := context.Background()

	env.handler = func(w http.ResponseWriter, r *http.Request) {
		rsp := PhoneResponse{
			OrderRef: "131daac9-16c6-4618-beb0-365768f37288",
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(&rsp)
	}

	resp, err := PhoneAuth(ctx, env, &PhoneAuthRequest{
		PersonalNumber: "198001010000",
		CallInitiator:  CallInitiatorUser,
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "131daac9-16c6-4618-beb0-365768f37288", resp.OrderRef)

	env.server.Close()
}

func TestPhoneSign_v6(t *testing.T) {
	env := &testEnv{}
	ctx := context.Background()

	env.handler = func(w http.ResponseWriter, r *http.Request) {
		rsp := PhoneResponse{
			OrderRef: "131daac9-16c6-4618-beb0-365768f37288",
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(&rsp)
	}

	resp, err := PhoneSign(ctx, env, &PhoneSignRequest{
		PersonalNumber:  "198001010000",
		CallInitiator:   CallInitiatorRP,
		UserVisibleData: "Sign this document",
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)

	env.server.Close()
}

func TestCancel_v6(t *testing.T) {
	testFunctions := []*tt{
		{
			name: "Expected: Successful, w/ correct response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				authRsp := Response{} // Empty but valid
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(&authRsp)
			},
			assert: func(t *testing.T, _ interface{}, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "Expected: Successful, w/ incorrect response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write(nil) // Empty and invalid
			},
			assert: func(t *testing.T, _ interface{}, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "Expected: Fail, w/ correct response body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				errRsp := ErrorResponse{} // Empty but valid
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(&errRsp)
			},
			assert: func(t *testing.T, _ interface{}, err error) {
				assert.NotNil(t, err)
			},
		},
	}

	env := &testEnv{}
	ctx := context.Background()

	for _, at := range testFunctions {
		env.handler = at.handler
		t.Logf("Cancel: %s", at.name)
		err := Cancel(ctx, env, "dbbee61c-357b-4fd8-b103-392eed10be7a")
		at.assert(t, nil, err)
	}

	env.server.Close()
}

func TestCollectCompletion_v6(t *testing.T) {
	env := &testEnv{}
	ctx := context.Background()

	env.handler = func(w http.ResponseWriter, r *http.Request) {
		rsp := CollectResponse{
			OrderRef: "131daac9-16c6-4618-beb0-365768f37288",
			Status:   OrderComplete,
			CompletionData: &Completion{
				User: User{
					PersonalNumber: "197001010000",
					Name:           "Test Testsson",
					GivenName:      "Test",
					Surname:        "Testsson",
				},
				Device: Device{
					IPAddress: "192.168.0.1",
					UHI:       "abc123def456",
				},
				BankIDIssueDate: "2023-01-01Z",
				StepUp: &StepUp{
					MRTD: false,
				},
				Signature:    "PHNpZ25hdHVyZT4=",
				OCSPResponse: "MIIHfgoBAKCCB3c=",
				Risk:         "low",
			},
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(&rsp)
	}

	resp, err := Collect(ctx, env, "dbbee61c-357b-4fd8-b103-392eed10be7a")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, OrderComplete, resp.Status)
	assert.NotNil(t, resp.CompletionData)
	assert.Equal(t, "Test Testsson", resp.CompletionData.User.Name)
	assert.Equal(t, "abc123def456", resp.CompletionData.Device.UHI)
	assert.Equal(t, "2023-01-01Z", resp.CompletionData.BankIDIssueDate)
	assert.NotNil(t, resp.CompletionData.StepUp)
	assert.False(t, resp.CompletionData.StepUp.MRTD)
	assert.Equal(t, "low", resp.CompletionData.Risk)

	env.server.Close()
}

func TestIsErrorResponse(t *testing.T) {
	env := &testEnv{}
	ctx := context.Background()

	env.handler = func(w http.ResponseWriter, r *http.Request) {
		errRsp := ErrorResponse{
			ErrorCode: "alreadyInProgress",
			Details:   "An order is already in progress",
		}
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(&errRsp)
	}

	_, err := Auth(ctx, env, &AuthRequest{EndUserIP: "127.0.0.1"})
	assert.NotNil(t, err)

	bankidErr, ok := IsErrorResponse(err)
	assert.True(t, ok)
	assert.Equal(t, "alreadyInProgress", bankidErr.ErrorCode)

	env.server.Close()
}

//
// Test invalid environment
//

type invalidEnv struct {
	testEnv
	request      *http.Request
	requestError error
}

func (t *invalidEnv) NewRequest(ctx context.Context, endpoint string, body interface{}) (*http.Request, error) {
	return t.request, t.requestError
}

func TestCallMethod(t *testing.T) {
	env := &invalidEnv{}
	ctx := context.Background()

	env.request = &http.Request{}
	env.requestError = nil
	req, err := call(ctx, "", env, nil, nil)
	assert.Nil(t, req)
	assert.NotNil(t, err)

	env.requestError = fmt.Errorf("fake invalid response")
	req, err = call(ctx, "", env, nil, nil)
	assert.Nil(t, req)
	assert.NotNil(t, err)
}
