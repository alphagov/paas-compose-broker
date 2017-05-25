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
func New(token string, apiURL *url.URL) *Client {
	return &Client{
		Token:  token,
		APIURL: apiURL,
	}
}

func (c *Client) requestDo(reqtype, path string, body []byte) (resp *http.Response, err error) {
	url := c.APIURL.String() + path
	req, err := http.NewRequest(reqtype, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
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

func validateResponse(resp *http.Response, err error) (Response, error) {
	cr := Response{}
	if err != nil {
		return cr, err
	}
	err = decodeResponse(resp, &cr)
	if err != nil {
		return cr, err
	}
	return cr, nil
}

// CreateDeployment will execute an API call to deployment with the use of an
// existing client.
func (c *Client) CreateDeployment(accountID, name, dbtype, dataCenter, version string, units int, ssl, wiredtiger bool) (Recipe, error) {

	depReq := struct {
		Deployment `json:"deployment"`
	}{
		Deployment{
			AccountID:  accountID,
			Name:       name,
			Type:       dbtype,
			Datacenter: dataCenter,
			Version:    version,
			Units:      units,
			SSL:        ssl,
			WiredTiger: wiredtiger,
		},
	}
	depResp := Recipe{}
	var body []byte
	body, err := json.Marshal(&depReq)
	if err != nil {
		return depResp, err
	}
	resp, err := c.requestDo("POST", "/deployments", body)
	if err != nil {
		return depResp, err
	}
	err = decodeResponse(resp, &depResp)
	if err != nil {
		return depResp, err
	}
	return depResp, nil
}
