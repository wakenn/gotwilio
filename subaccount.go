package gotwilio

import (
	"encoding/json"
	"net/http"

	"github.com/google/go-querystring/query"
)

// IncomingPhoneNumber represents a phone number resource owned by the calling account in Twilio
type IncomingSubAccount struct {
	SID          string `json:"sid"`
	FriendlyName string `url:"FriendlyName,omitempty" json:"FriendlyName"`
	AuthToken    string `url:"auth_token,omitempty" json:"auth_token"`
}

// CreateIncomingPhoneNumber creates an IncomingPhoneNumber resource via the Twilio REST API.
// https://www.twilio.com/docs/phone-numbers/api/incomingphonenumber-resource#create-an-incomingphonenumber-resource
func (twilio *Twilio) CreateSubAccount(options IncomingSubAccount) (*IncomingSubAccount, *Exception, error) {
	// convert options to HTTP form
	form, err := query.Values(options)
	if err != nil {
		return nil, nil, err
	}

	res, err := twilio.post(form, twilio.BaseUrl+"/Accounts.json")
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(res.Body)

	// handle NULL response
	if res.StatusCode != http.StatusCreated {
		exception := new(Exception)
		err = decoder.Decode(exception)
		return nil, exception, err
	}

	incomingSubAccount := new(IncomingSubAccount)
	err = decoder.Decode(incomingSubAccount)
	return incomingSubAccount, nil, err
}
