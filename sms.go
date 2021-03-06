package gotwilio

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SmsResponse is returned after a text/sms message is posted to Twilio
type SmsResponse struct {
	Sid          string  `json:"sid"`
	DateCreated  string  `json:"date_created"`
	DateUpdate   string  `json:"date_updated"`
	DateSent     string  `json:"date_sent"`
	AccountSid   string  `json:"account_sid"`
	To           string  `json:"to"`
	From         string  `json:"from"`
	MediaUrl     string  `json:"media_url"`
	NumMedia     string  `json:"num_media"`
	NumSegments  string  `json:"num_segments"`
	Body         string  `json:"body"`
	Status       string  `json:"status"`
	Direction    string  `json:"direction"`
	ApiVersion   string  `json:"api_version"`
	Price        *string `json:"price,omitempty"`
	Url          string  `json:"uri"`
	ErrorCode    int     `json:"error_code"`
	ErrorMessage string  `json:"error_message"`
}

// DateCreatedAsTime returns SmsResponse.DateCreated as a time.Time object
// instead of a string.
func (sms *SmsResponse) DateCreatedAsTime() (time.Time, error) {
	return time.Parse(time.RFC1123Z, sms.DateCreated)
}

// DateUpdateAsTime returns SmsResponse.DateUpdate as a time.Time object
// instead of a string.
func (sms *SmsResponse) DateUpdateAsTime() (time.Time, error) {
	return time.Parse(time.RFC1123Z, sms.DateUpdate)
}

// DateSentAsTime returns SmsResponse.DateSent as a time.Time object
// instead of a string.
func (sms *SmsResponse) DateSentAsTime() time.Time {
	out, _ := time.Parse(time.RFC1123Z, sms.DateSent)
	return out
}

func (sms *SmsResponse) IsMMS() bool {
	return sms.NumMedia != "0"
}

func (sms *SmsResponse) GetSegments() int {
	if sms.NumSegments == "" || sms.NumSegments == "1" {
		return 1
	}

	val, err := strconv.Atoi(sms.NumSegments)
	if err != nil {
		log.Println("Error getting num segments", sms.Sid, sms.NumSegments)
		return 1
	}

	return val
}

func (sms *SmsResponse) IsInbound() bool {
	return strings.Contains(sms.Direction, "inbound")
}

func whatsapp(phone string) string {
	return "whatsapp:" + phone
}

// SendWhatsApp uses Twilio to send a WhatsApp message.
// See https://www.twilio.com/docs/sms/whatsapp/tutorial/send-and-receive-media-messages-whatsapp-python
func (twilio *Twilio) SendWhatsApp(from, to, body, statusCallback, applicationSid string) (smsResponse *SmsResponse, exception *Exception, err error) {
	return twilio.SendSMS(whatsapp(from), whatsapp(to), body, statusCallback, applicationSid)
}

// SendSMS uses Twilio to send a text message.
// See http://www.twilio.com/docs/api/rest/sending-sms for more information.
func (twilio *Twilio) SendSMS(from, to, body, statusCallback, applicationSid string) (smsResponse *SmsResponse, exception *Exception, err error) {
	formValues := initFormValues(to, body, []string{}, statusCallback, applicationSid)
	formValues.Set("From", from)

	smsResponse, exception, err = twilio.sendMessage(formValues)
	return
}

// GetSMS uses Twilio to get information about a text message.
// See https://www.twilio.com/docs/api/rest/sms for more information.
func (twilio *Twilio) GetSMS(sid string) (smsResponse *SmsResponse, exception *Exception, err error) {
	twilioUrl := twilio.BaseUrl + "/Accounts/" + twilio.AccountSid + "/SMS/Messages/" + sid + ".json"

	res, err := twilio.get(twilioUrl)
	if err != nil {
		return smsResponse, exception, err
	}
	defer res.Body.Close()

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return smsResponse, exception, err
	}

	if res.StatusCode != http.StatusOK {
		exception = new(Exception)
		err = json.Unmarshal(responseBody, exception)

		// We aren't checking the error because we don't actually care.
		// It's going to be passed to the client either way.
		return smsResponse, exception, err
	}

	smsResponse = new(SmsResponse)
	err = json.Unmarshal(responseBody, smsResponse)
	return smsResponse, exception, err
}

// SendSMSWithCopilot uses Twilio Copilot to send a text message.
// See https://www.twilio.com/docs/api/rest/sending-messages-copilot
func (twilio *Twilio) SendSMSWithCopilot(messagingServiceSid, to, body, statusCallback, applicationSid string) (smsResponse *SmsResponse, exception *Exception, err error) {
	formValues := initFormValues(to, body, []string{}, statusCallback, applicationSid)
	formValues.Set("MessagingServiceSid", messagingServiceSid)

	smsResponse, exception, err = twilio.sendMessage(formValues)
	return
}

// SendMMS uses Twilio to send a multimedia message.
func (twilio *Twilio) SendMMS(from, to, body string, mediaUrl []string, statusCallback, applicationSid string) (smsResponse *SmsResponse, exception *Exception, err error) {
	formValues := initFormValues(to, body, mediaUrl, statusCallback, applicationSid)
	formValues.Set("From", from)

	smsResponse, exception, err = twilio.sendMessage(formValues)
	return
}

// Core method to send message
func (twilio *Twilio) sendMessage(formValues url.Values) (smsResponse *SmsResponse, exception *Exception, err error) {
	twilioUrl := twilio.BaseUrl + "/Accounts/" + twilio.AccountSid + "/Messages.json"

	res, err := twilio.post(formValues, twilioUrl)
	if err != nil {
		return smsResponse, exception, err
	}
	defer res.Body.Close()

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return smsResponse, exception, err
	}

	if res.StatusCode != http.StatusCreated {
		exception = new(Exception)
		err = json.Unmarshal(responseBody, exception)

		// We aren't checking the error because we don't actually care.
		// It's going to be passed to the client either way.
		return smsResponse, exception, err
	}

	smsResponse = new(SmsResponse)
	err = json.Unmarshal(responseBody, smsResponse)
	return smsResponse, exception, err
}

func (twilio *Twilio) GetConversation(to, from, createdOnOrBefore, createdAfter string) ([]*SmsResponse, *Exception, error) {
	convo := []*SmsResponse{}
	inbound, exc, err := twilio.GetMessages(to, from, createdOnOrBefore, createdAfter)
	if exc != nil || err != nil {
		return nil, exc, err
	}
	convo = append(convo, inbound...)

	outbound, exc, err := twilio.GetMessages(from, to, createdOnOrBefore, createdAfter)
	if exc != nil || err != nil {
		return nil, exc, err
	}

	convo = append(convo, outbound...)

	sort.Slice(convo, func(i int, j int) bool {
		return convo[i].DateSentAsTime().Unix() < convo[j].DateSentAsTime().Unix()
	})

	return convo, nil, nil

}
func (twilio *Twilio) GetMessages(to, from, createdOnOrBefore, createdAfter string) ([]*SmsResponse, *Exception, error) {
	values := url.Values{}
	if to != "" {
		values.Set("To", to)
	}
	if from != "" {
		values.Set("From", from)
	}
	if createdOnOrBefore != "" {
		values.Set("DateCreatedOnOrBefore", createdOnOrBefore)
	}
	if createdAfter != "" {
		values.Set("DateCreatedAfter", createdAfter)
	}

	values.Set("PageSize", "1000")

	twilioUrl := twilio.BaseUrl + "/Accounts/" + twilio.AccountSid + "/Messages.json"

	// Retrieve all messages FROM the host to the client
	var (
		url *url.URL
		err error
	)
	if url, err = url.Parse(twilioUrl); err != nil {
		return nil, nil, err
	}
	url.RawQuery = values.Encode()

	resp, err := twilio.get(url.String())
	if err != nil {
		return nil, nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		exc := new(Exception)
		err = json.Unmarshal(respBody, exc)
		return nil, exc, err
	}

	lr := twilio.newListResources()
	if err := json.Unmarshal(respBody, lr); err != nil {
		return nil, nil, err
	}
	frs := lr.Messages
	log.Println("FIRST TO MSGS", url.String(), len(lr.Messages))

	for {
		if lr.NextPageUri == "" {
			break
		}

		uri := "https://api.twilio.com" + lr.NextPageUri
		resp, err := twilio.get(uri)
		if err != nil {
			return nil, nil, err
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}

		if resp.StatusCode != http.StatusOK {
			exc := new(Exception)
			err = json.Unmarshal(respBody, exc)
			return nil, exc, err
		}

		lr = twilio.newListResources()
		if err := json.Unmarshal(respBody, lr); err != nil {
			return nil, nil, err
		}

		log.Println("NEXT: URI TO MSGS", uri, len(lr.Messages))
		frs = append(frs, lr.Messages...)
	}

	return frs, nil, nil
}

func (twilio *Twilio) GetMessage(sid string) (*SmsResponse, *Exception, error) {
	twilioURL := twilio.BaseUrl + "/Accounts/" + twilio.AccountSid + "/Messages/" + sid + ".json"

	// Retrieve all messages FROM the host to the client
	var (
		url *url.URL
		err error
	)
	if url, err = url.Parse(twilioURL); err != nil {
		return nil, nil, err
	}

	resp, err := twilio.get(url.String())
	if err != nil {
		return nil, nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		exc := new(Exception)
		err = json.Unmarshal(respBody, exc)
		return nil, exc, err
	}

	var sms SmsResponse
	if err := json.Unmarshal(respBody, &sms); err != nil {
		return nil, nil, err
	}
	return &sms, nil, nil
}

// Form values initialization
func initFormValues(to, body string, mediaUrl []string, statusCallback, applicationSid string) url.Values {
	formValues := url.Values{}

	formValues.Set("To", to)
	formValues.Set("Body", body)

	if len(mediaUrl) > 0 {
		for _, value := range mediaUrl {
			formValues.Add("MediaUrl", value)
		}
	}

	if statusCallback != "" {
		formValues.Set("StatusCallback", statusCallback)
	}

	if applicationSid != "" {
		formValues.Set("ApplicationSid", applicationSid)
	}

	return formValues
}
