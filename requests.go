package bankid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Auth initiates an authentication order.
//
// In BankID v6.0, the user's personal number is no longer accepted here.
// The response contains an autoStartToken (for same-device flows) and
// qrStartToken/qrStartSecret (for animated QR code flows on another device).
func Auth(ctx context.Context, env Environmenter, req *AuthRequest) (*Response, error) {
	output := &Response{}
	rsp, err := call(ctx, AuthEndpoint, env, req, responseParser)
	if err == nil && rsp != nil {
		output = rsp.(*Response)
	}
	return output, err
}

// Sign initiates a signing order.
//
// The userVisibleData in the request will be base64-encoded automatically.
// If userNonVisibleData is provided, it will also be base64-encoded.
func Sign(ctx context.Context, env Environmenter, req *SignRequest) (*Response, error) {
	// Base64 encode with padding
	if req.UserVisibleData != "" {
		req.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(req.UserVisibleData))
	}

	if req.UserNonVisibleData != "" {
		req.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(req.UserNonVisibleData))
	}

	output := &Response{}
	rsp, err := call(ctx, SignEndpoint, env, req, responseParser)
	if err == nil && rsp != nil {
		output = rsp.(*Response)
	}
	return output, err
}

// PhoneAuth initiates authentication during a phone call.
// This is the only v6.0 flow that accepts a personal number directly,
// as the customer service agent already knows who they are speaking to.
func PhoneAuth(ctx context.Context, env Environmenter, req *PhoneAuthRequest) (*PhoneResponse, error) {
	output := &PhoneResponse{}
	rsp, err := call(ctx, PhoneAuthEndpoint, env, req, phoneResponseParser)
	if err == nil && rsp != nil {
		output = rsp.(*PhoneResponse)
	}
	return output, err
}

// PhoneSign initiates signing during a phone call.
// The userVisibleData will be base64-encoded automatically.
func PhoneSign(ctx context.Context, env Environmenter, req *PhoneSignRequest) (*PhoneResponse, error) {
	if req.UserVisibleData != "" {
		req.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(req.UserVisibleData))
	}

	if req.UserNonVisibleData != "" {
		req.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(req.UserNonVisibleData))
	}

	output := &PhoneResponse{}
	rsp, err := call(ctx, PhoneSignEndpoint, env, req, phoneResponseParser)
	if err == nil && rsp != nil {
		output = rsp.(*PhoneResponse)
	}
	return output, err
}

// Collect polls for the status of an ongoing order.
// Call this approximately every 2 seconds until the order reaches
// a final status (complete or failed).
func Collect(ctx context.Context, env Environmenter, orderRef string) (*CollectResponse, error) {
	requestBody := CollectRequest{
		OrderRef: orderRef,
	}

	output := &CollectResponse{}
	rsp, err := call(ctx, CollectEndpoint, env, &requestBody, collectParser)
	if err == nil && rsp != nil {
		output = rsp.(*CollectResponse)
	}
	return output, err
}

// Cancel cancels an ongoing order.
func Cancel(ctx context.Context, env Environmenter, orderRef string) error {
	requestBody := CancelRequest{
		OrderRef: orderRef,
	}
	_, err := call(ctx, CancelEndpoint, env, &requestBody, responseParser)
	return err
}

// responseParserFunc is used to parse the BankID API response
type responseParserFunc func(*http.Response) (interface{}, error)

func call(ctx context.Context, endpoint string, env Environmenter, requestBody interface{}, rspParser responseParserFunc) (interface{}, error) {
	req, err := env.NewRequest(ctx, endpoint, requestBody)
	if err != nil {
		return nil, err
	}

	client := env.NewClient()

	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return rspParser(rsp)
}

func responseParser(rsp *http.Response) (interface{}, error) {
	defer rsp.Body.Close()

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		authRsp := Response{}
		err := json.NewDecoder(rsp.Body).Decode(&authRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &authRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(rsp.Body).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}

	// We don't care about HTTP 1xx messages
	return nil, nil
}

func phoneResponseParser(rsp *http.Response) (interface{}, error) {
	defer rsp.Body.Close()

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		phoneRsp := PhoneResponse{}
		err := json.NewDecoder(rsp.Body).Decode(&phoneRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &phoneRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(rsp.Body).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}

	// We don't care about HTTP 1xx messages
	return nil, nil
}

func collectParser(rsp *http.Response) (interface{}, error) {
	defer rsp.Body.Close()

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		collectRsp := CollectResponse{}
		err := json.NewDecoder(rsp.Body).Decode(&collectRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &collectRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(rsp.Body).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}
	// We don't care about HTTP 1xx messages
	return nil, nil
}

// IsErrorResponse checks if the given error is a BankID ErrorResponse
// and returns it if so. This allows callers to inspect error codes.
func IsErrorResponse(err error) (*ErrorResponse, bool) {
	var errRsp ErrorResponse
	if errors.As(err, &errRsp) {
		return &errRsp, true
	}
	return nil, false
}
