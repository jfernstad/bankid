# BankID API v6.0

Go library for integrating with the [BankID](https://www.bankid.com/) authentication service (Sweden).

> **⚠️ Breaking changes from v5:** This library has been upgraded to BankID API v6.0. The `personalNumber` parameter has been removed from `/auth` and `/sign` endpoints. Authentication now requires either animated QR codes (another device) or autostart tokens (same device). See the [BankID Developer Portal](https://www.bankid.com/en/utvecklare) for full details.

## Features

- **Authentication** (`/auth`) — Initiate user authentication
- **Signing** (`/sign`) — Initiate document signing
- **Phone authentication** (`/phone/auth`) — Authenticate during phone calls
- **Phone signing** (`/phone/sign`) — Sign during phone calls
- **Collect** (`/collect`) — Poll for order status
- **Cancel** (`/cancel`) — Cancel ongoing orders
- **Animated QR codes** — Generate QR code data per the Secure Start specification
- **Recommended messages** — RFA messages in Swedish and English (per RP Guidelines v6.0)

## Authentication Flow

BankID v6.0 requires "Secure Start" for all authentication flows:

### Same Device (Autostart)
1. Call `Auth()` with the end user's IP
2. Use `autoStartToken` from the response to open `bankid:///?autostarttoken=<token>&redirect=null`
3. Poll `Collect()` every 2 seconds until complete or failed

### Another Device (QR Code)
1. Call `Auth()` with the end user's IP
2. Use `qrStartToken` and `qrStartSecret` from the response
3. Generate animated QR data every second using `GenerateQRData()`
4. Display the QR code to the user
5. Poll `Collect()` every 2 seconds until complete or failed

### Phone Call
1. Call `PhoneAuth()` with the user's personal number and call initiator
2. Poll `Collect()` every 2 seconds until complete or failed

## Example

```golang
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/jfernstad/bankid"
)

func main() {
    p, _ := bankid.NewMessages("en")

    caTestPath := "../CA/test.crt"
    rpCrtPath := "../rp/bankid_rp_test.crt" // NOTE: Replace with your RP certificate
    rpKeyPath := "../rp/bankid_rp_test.key"  // NOTE: Replace with your RP key

    // bankid.TestBaseURL or bankid.ProductionBaseURL
    env, err := bankid.NewEnvironment(bankid.TestBaseURL, caTestPath, rpCrtPath, rpKeyPath)
    if err != nil {
        log.Fatalf("Could not create environment: %s", err.Error())
    }

    ctx := context.Background()

    // v6.0: No personal number in auth request!
    rsp, err := bankid.Auth(ctx, env, &bankid.AuthRequest{
        EndUserIP: "127.0.0.1",
    })
    if err != nil {
        log.Fatalf("Could not start auth: %s", err.Error())
    }

    fmt.Printf("Order started: %s\n", rsp.OrderRef)
    fmt.Printf("AutoStartToken: %s\n", rsp.AutoStartToken)

    // Generate animated QR codes for another-device flow
    orderStart := time.Now()

    for {
        elapsed := time.Since(orderStart)
        qrData := bankid.GenerateQRData(rsp.QRStartToken, rsp.QRStartSecret, elapsed)
        fmt.Printf("QR: %s\n", qrData)

        collectRsp, err := bankid.Collect(ctx, env, rsp.OrderRef)
        if err != nil {
            log.Fatalf("Could not collect: %s", err.Error())
        }

        switch collectRsp.Status {
        case bankid.OrderPending:
            switch collectRsp.HintCode {
            case bankid.PendOutstandingTransaction, bankid.PendNoClient:
                fmt.Println(p.Msg(bankid.RFA1))
            case bankid.PendStarted:
                fmt.Println(p.Msg(bankid.RFA15_B))
            case bankid.PendUserSign:
                fmt.Println(p.Msg(bankid.RFA9))
            case bankid.PendUserMrtd:
                fmt.Println(p.Msg(bankid.RFA23))
            default:
                fmt.Println(p.Msg(bankid.RFA21))
            }
        case bankid.OrderFailed:
            fmt.Println(p.Msg(bankid.RFA22))
            os.Exit(1)
        case bankid.OrderComplete:
            fmt.Printf("✅ %s authenticated!\n", collectRsp.CompletionData.User.Name)
            os.Exit(0)
        }

        time.Sleep(2 * time.Second)
    }
}
```

For signing data, use `bankid.Sign()` with a `SignRequest` instead. The flow is the same.

## QR Code Generation

The `GenerateQRData()` function produces the data string to encode as a QR image:

```golang
// Call every second with increasing elapsed time
qrData := bankid.GenerateQRData(rsp.QRStartToken, rsp.QRStartSecret, elapsed)
// Use any QR library to render qrData as a QR code image
```

The algorithm: `bankid.<qrStartToken>.<seconds>.<HMAC-SHA256(qrStartSecret, seconds)>`

## Error Handling

BankID errors can be inspected using `IsErrorResponse()`:

```golang
_, err := bankid.Auth(ctx, env, req)
if errRsp, ok := bankid.IsErrorResponse(err); ok {
    switch errRsp.ErrorCode {
    case "alreadyInProgress":
        // Show RFA4 message
    case "invalidParameters":
        // Check your request
    }
}
```
