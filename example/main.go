package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jfernstad/bankid"
	"github.com/mdp/qrterminal/v3"
)

func main() {

	// Let's use the official messages
	// for events and errors, in English
	p, _ := bankid.NewMessages("en")

	caTestPath := "../CA/test.crt"
	rpCrtPath := "../rp/bankid_rp_test.crt" // NOTE: Replace with your RP (Relying Partner) certificate
	rpKeyPath := "../rp/bankid_rp_test.key"  // NOTE: Replace with your RP key

	// bankid.TestBaseURL or bankid.ProductionBaseURL
	env, err := bankid.NewEnvironment(bankid.TestBaseURL, caTestPath, rpCrtPath, rpKeyPath)
	if err != nil {
		log.Printf(" !! Could not create TestEnvironment: %s", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	// BankID v6.0: No personal number in the auth request!
	// The user identifies themselves via the BankID app,
	// either by scanning a QR code (another device) or via autostart (same device).
	ipAddr := "127.0.0.1" // IP of the end user

	// Print message as instructed by the RP Guidelines v6.0
	fmt.Println(" >> " + p.Msg(bankid.RFA20))

	rsp, err := bankid.Auth(ctx, env, &bankid.AuthRequest{
		EndUserIP: ipAddr,
	})
	if err != nil {
		log.Printf(" !! Could not connect to server: %s\n", err.Error())
		os.Exit(1)
	}

	// Auth started!
	// The response now contains qrStartToken and qrStartSecret
	// for generating an animated QR code, as well as autoStartToken
	// for launching the BankID app on the same device.
	fmt.Printf(" >> Order started (orderRef: %s)\n", rsp.OrderRef)
	fmt.Printf(" >> AutoStartToken: %s\n", rsp.AutoStartToken)
	fmt.Printf(" >> QRStartToken:   %s\n", rsp.QRStartToken)

	// For same-device flow, open: bankid:///?autostarttoken=<autoStartToken>&redirect=null
	// For another-device flow, display the animated QR code:
	orderStartTime := time.Now()

	collectResponse := &bankid.CollectResponse{}
	done := false
	for !done {
		// Clear terminal screen so the QR code animates in place
		fmt.Print("\033[H\033[2J")

		// Generate animated QR code data (changes every second)
		elapsed := time.Since(orderStartTime)
		qrData := bankid.GenerateQRData(rsp.QRStartToken, rsp.QRStartSecret, elapsed)
		
		fmt.Println(" >> Scan this QR code with your BankID app:")
		qrterminal.GenerateHalfBlock(qrData, qrterminal.L, os.Stdout)

		collectResponse, err = bankid.Collect(ctx, env, rsp.OrderRef)
		if err != nil {
			log.Printf(" !! Could not collect: %s\n", err.Error())
			os.Exit(1)
		}

		switch collectResponse.Status {
		case bankid.OrderPending:
			switch collectResponse.HintCode {
			case bankid.PendOutstandingTransaction:
				fmt.Println(" >> " + p.Msg(bankid.RFA1))
			case bankid.PendNoClient:
				fmt.Println(" >> " + p.Msg(bankid.RFA1))
			case bankid.PendStarted:
				fmt.Println(" >> " + p.Msg(bankid.RFA15_B))
			case bankid.PendUserSign:
				fmt.Println(" >> " + p.Msg(bankid.RFA9))
			case bankid.PendUserMrtd:
				fmt.Println(" >> " + p.Msg(bankid.RFA23))
			default:
				// Handle unknown hint codes gracefully (RP Guidelines v6.0)
				fmt.Println(" >> " + p.Msg(bankid.RFA21))
			}
		case bankid.OrderFailed:
			done = true
			switch collectResponse.HintCode {
			case bankid.FailCancelled:
				fmt.Println(" >> " + p.Msg(bankid.RFA3))
			case bankid.FailUserCancel:
				fmt.Println(" >> " + p.Msg(bankid.RFA6))
			case bankid.FailExpiredTransaction:
				fmt.Println(" >> " + p.Msg(bankid.RFA8))
			default:
				fmt.Println(" >> " + p.Msg(bankid.RFA22))
			}
		case bankid.OrderComplete:
			done = true
			log.Println(" >> 😎 Auth Complete ")
			log.Printf(" >> %s signed in!\n", collectResponse.CompletionData.User.Name)
			if collectResponse.CompletionData.BankIDIssueDate != "" {
				log.Printf(" >> BankID issued: %s\n", collectResponse.CompletionData.BankIDIssueDate)
			}
			if collectResponse.CompletionData.StepUp != nil {
				log.Printf(" >> MRTD verified: %v\n", collectResponse.CompletionData.StepUp.MRTD)
			}
			if collectResponse.CompletionData.Risk != "" {
				log.Printf(" >> Risk level: %s\n", collectResponse.CompletionData.Risk)
			}
		}
		// Don't spam the service plz
		time.Sleep(2 * time.Second)
	}

	// Just to demonstrate cancelling, we'll probably never end up here.
	if collectResponse.Status == bankid.OrderPending {
		err = bankid.Cancel(ctx, env, rsp.OrderRef)
		if err != nil {
			log.Printf(" !! Could not cancel request: %s\n", err.Error())
		}
		log.Printf(" >> Auth cancelled\n")
	}
}
