// Package alerting provides Twilio voice and SMS alert delivery for NOC.
// Called when a KAI triage result reaches severity=critical and the
// communication persona (comm.py) needs a phone-based escalation path.
package alerting

import (
	"fmt"

	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioAlerter sends SMS and voice call notifications via the Twilio REST API.
type TwilioAlerter struct {
	client *twilio.RestClient
	from   string // E.164 Twilio number, e.g. "+15005550006"
}

// NewTwilioAlerter constructs a TwilioAlerter using the supplied SID + auth token.
func NewTwilioAlerter(accountSID, authToken, fromNumber string) *TwilioAlerter {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &TwilioAlerter{client: client, from: fromNumber}
}

// SendSMS sends a plain-text SMS to the given E.164 number.
func (a *TwilioAlerter) SendSMS(to, body string) error {
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(a.from)
	params.SetBody(body)
	_, err := a.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio SendSMS: %w", err)
	}
	return nil
}

// CallTwiML places an outbound voice call and reads the supplied TwiML URL.
func (a *TwilioAlerter) CallTwiML(to, twimlURL string) error {
	params := &openapi.CreateCallParams{}
	params.SetTo(to)
	params.SetFrom(a.from)
	params.SetUrl(twimlURL)
	_, err := a.client.Api.CreateCall(params)
	if err != nil {
		return fmt.Errorf("twilio CallTwiML: %w", err)
	}
	return nil
}
