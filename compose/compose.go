package compose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// New creates a struct responsible for the communcation with the compose API.
func New(token string, apiURL *url.URL) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("client: token should not be empty")
	}
	if apiURL == nil {
		return nil, fmt.Errorf("client: apiURl should not be nil")
	}

	client := &Client{
		Token:  token,
		APIURL: apiURL,
	}

	return client, nil
}

func (c *Client) requestDo(reqtype, path string, body []byte) (resp *http.Response, err error) {
	url := c.APIURL.String() + path
	req, err := http.NewRequest(reqtype, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	switch reqtype {
	default:
		req.Header.Set("Content-Type", "application/hal+json")
	case "POST":
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	// TODO make sure, we will be always expecting <= 202 http response code.
	if resp.StatusCode > 202 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s", string(body))
	}

	return resp, nil
}

func decodeResponse(r *http.Response, v interface{}) error {
	if v == nil {
		return fmt.Errorf("nil interface provided to decodeResponse")
	}

	bodyBytes, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(bodyBytes, &v)

	return err
}

// CreateDeployment will execute an API call to deployment with the use of an
// existing client.
func (c *Client) CreateDeployment(deployment *Deployment) (*Deployment, error) {
	body, err := json.Marshal(&deployment)
	if err != nil {
		return nil, err
	}

	res, err := c.requestDo("POST", "/deployments", body)
	if err != nil {
		return nil, err
	}

	err = decodeResponse(res, &deployment)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}
