package bankid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxResponseSize limits how much data we read from BankID API responses (1 MB).
// This prevents memory exhaustion from malformed or malicious responses.
const maxResponseSize = 1 << 20

// Auth initiates an authentication order.
//
// In BankID v6.0, the user's personal number is no longer accepted here.
// The response contains an autoStartToken (for same-device flows) and
// qrStartToken/qrStartSecret (for animated QR code flows on another device).
func Auth(ctx context.Context, env Environmenter, req *AuthRequest) (*Response, error) {
	encodedReq := *req

	if encodedReq.UserVisibleData != "" {
		encodedReq.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserVisibleData))
	}

	if encodedReq.UserNonVisibleData != "" {
		encodedReq.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserNonVisibleData))
	}

	output := &Response{}
	rsp, err := call(ctx, AuthEndpoint, env, &encodedReq, responseParser)
	if err == nil && rsp != nil {
		output = rsp.(*Response)
	}
	return output, err
}

// Sign initiates a signing order.
//
// The userVisibleData in the request will be base64-encoded automatically.
// If userNonVisibleData is provided, it will also be base64-encoded.
// The original request struct is not modified.
func Sign(ctx context.Context, env Environmenter, req *SignRequest) (*Response, error) {
	// Copy the request to avoid mutating the caller's struct (prevents double-encoding on reuse)
	encodedReq := *req

	// Base64 encode with padding
	if encodedReq.UserVisibleData != "" {
		encodedReq.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserVisibleData))
	}

	if encodedReq.UserNonVisibleData != "" {
		encodedReq.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserNonVisibleData))
	}

	output := &Response{}
	rsp, err := call(ctx, SignEndpoint, env, &encodedReq, responseParser)
	if err == nil && rsp != nil {
		output = rsp.(*Response)
	}
	return output, err
}

// PhoneAuth initiates authentication during a phone call.
// This is the only v6.0 flow that accepts a personal number directly,
// as the customer service agent already knows who they are speaking to.
func PhoneAuth(ctx context.Context, env Environmenter, req *PhoneAuthRequest) (*PhoneResponse, error) {
	encodedReq := *req

	if encodedReq.UserVisibleData != "" {
		encodedReq.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserVisibleData))
	}

	if encodedReq.UserNonVisibleData != "" {
		encodedReq.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserNonVisibleData))
	}

	output := &PhoneResponse{}
	rsp, err := call(ctx, PhoneAuthEndpoint, env, &encodedReq, phoneResponseParser)
	if err == nil && rsp != nil {
		output = rsp.(*PhoneResponse)
	}
	return output, err
}

// PhoneSign initiates signing during a phone call.
// The userVisibleData will be base64-encoded automatically.
// The original request struct is not modified.
func PhoneSign(ctx context.Context, env Environmenter, req *PhoneSignRequest) (*PhoneResponse, error) {
	// Copy the request to avoid mutating the caller's struct (prevents double-encoding on reuse)
	encodedReq := *req

	if encodedReq.UserVisibleData != "" {
		encodedReq.UserVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserVisibleData))
	}

	if encodedReq.UserNonVisibleData != "" {
		encodedReq.UserNonVisibleData = base64.StdEncoding.EncodeToString([]byte(encodedReq.UserNonVisibleData))
	}

	output := &PhoneResponse{}
	rsp, err := call(ctx, PhoneSignEndpoint, env, &encodedReq, phoneResponseParser)
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
	limited := io.LimitReader(rsp.Body, maxResponseSize)

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		authRsp := Response{}
		err := json.NewDecoder(limited).Decode(&authRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &authRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(limited).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}

	return nil, fmt.Errorf("unexpected HTTP status: %d", rsp.StatusCode)
}

func phoneResponseParser(rsp *http.Response) (interface{}, error) {
	defer rsp.Body.Close()
	limited := io.LimitReader(rsp.Body, maxResponseSize)

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		phoneRsp := PhoneResponse{}
		err := json.NewDecoder(limited).Decode(&phoneRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &phoneRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(limited).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}

	return nil, fmt.Errorf("unexpected HTTP status: %d", rsp.StatusCode)
}

func collectParser(rsp *http.Response) (interface{}, error) {
	defer rsp.Body.Close()
	limited := io.LimitReader(rsp.Body, maxResponseSize)

	// OK
	if rsp.StatusCode >= 200 && rsp.StatusCode < 400 {
		collectRsp := CollectResponse{}
		err := json.NewDecoder(limited).Decode(&collectRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse successful response: %w", err)
		}
		return &collectRsp, nil
	}

	// Fail
	if rsp.StatusCode >= 400 {
		errRsp := ErrorResponse{}
		err := json.NewDecoder(limited).Decode(&errRsp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, errRsp
	}

	return nil, fmt.Errorf("unexpected HTTP status: %d", rsp.StatusCode)
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
