package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type PagerDutyClient struct {
	httpClient *http.Client
	hostname   string
	apiToken   string
}

const (
	// PAGERDUTY_API_URL is the base URL for the PagerDuty API.
	PAGERDUTY_API_URL = "https://api.pagerduty.com"

	// CLIENT_URL is the URL for the client that is triggering the event.
	// This is used in the client_url field of the event payload.
	CLIENT_URL = "https://cloudprober.org/"
)

func NewPagerDutyClient(hostname, apiToken string) *PagerDutyClient {
	return &PagerDutyClient{
		httpClient: &http.Client{},
		hostname:   hostname,
		apiToken:   apiToken,
	}
}

func (p *PagerDutyClient) SendEventV2(event *PagerDutyEventV2Request) (*PagerDutyEventV2Response, error) {
	req, err := event.newHTTPRequest(p.hostname)
	if err != nil {
		return nil, err
	}

	resp, err := p.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check status code, return error if not 200
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PagerDuty API returned status code %d, body: %s", resp.StatusCode, resp.Body)
	}

	body := &PagerDutyEventV2Response{}
	err = json.NewDecoder(resp.Body).Decode(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Do sends an HTTP request and returns an HTTP response, ensuring that the API
// token is set in the request header.
func (p *PagerDutyClient) do(req *http.Request) (*http.Response, error) {
	// The PagerDuty API requires authentication, using the API token.
	// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTUx-authentication
	req.Header.Set("Authorization", "Token token="+p.apiToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
