// Package spotify exports an interface to the Spotify Web API.
package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type client struct {
	httpc                    *http.Client
	apiBaseURL, authBaseURL  string
	authHeader, refreshToken string
	token                    *token

	// nowFunc returns the current local time. Can be used to instrument tests.
	nowFunc func() time.Time
}

type token struct {
	bearer    string
	expiresAt time.Time
}

// If the token expires within this duration, apiRequest a new one anyway: this
// guards against the next apiRequest still failing due to an expired token.
const expiryThreshold = 5 * time.Second

func NewClient(cID, cSecret, refreshToken string) *client {
	ah := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cID, cSecret)))
	return &client{
		httpc: &http.Client{
			Timeout: 5 * time.Second,
		},
		apiBaseURL:   "https://api.spotify.com",
		authBaseURL:  "https://accounts.spotify.com",
		authHeader:   ah,
		refreshToken: refreshToken,
		nowFunc:      time.Now,
	}
}

func (c *client) bearerToken() error {
	if c.token != nil && c.token.bearer != "" {
		// Already have a token, so no-op if it doesn't expire in the near
		// future.
		if !c.token.expiresAt.Before(c.nowFunc().Add(expiryThreshold)) {
			return nil
		}
	}

	vals := url.Values{}
	vals.Set("grant_type", "refresh_token")
	vals.Set("refresh_token", c.refreshToken)
	req, err := http.NewRequest(http.MethodPost, c.authBaseURL+"/api/token", strings.NewReader(vals.Encode()))
	if err != nil {
		return fmt.Errorf("net/http: NewRequest: %s", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("net/http: Request.Do: %s", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	var data struct {
		ExpiresInSecs int    `json:"expires_in"`
		Token         string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return fmt.Errorf("encoding/json: Decoder.Decode: %s", err)
	}

	c.token = &token{
		bearer:    data.Token,
		expiresAt: c.nowFunc().Add(time.Duration(data.ExpiresInSecs) * time.Second),
	}
	return nil
}

// apiRequest sends a request and gets the response. Data is optional, but if
// set, it will be JSON encoded and written to the request body.
func (c *client) apiRequest(method, path string, data interface{}) (*http.Response, error) {
	if err := c.bearerToken(); err != nil {
		return nil, fmt.Errorf("bearerToken: %s", err)
	}

	var ct string
	var rr io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encoding/json: Marshal: %s", err)
		}
		ct = "application/json"
		rr = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.apiBaseURL+path, rr)
	if err != nil {
		return nil, fmt.Errorf("net/http: NewRequest: %s", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token.bearer)
	req.Header.Set("Content-Type", ct)

	res, err := c.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("net/http: Client.Do: %s", err)
	}
	return res, nil
}

// AddToQueue adds an item, defined by uri, to the end of the user's current
// playback queue.
func (c *client) AddToQueue(uri string) error {
	res, err := c.apiRequest(http.MethodPost, "/v1/me/player/add-to-queue?uri="+uri, nil)
	if err != nil {
		return fmt.Errorf("apiRequest: %s", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNoContent {
		rs, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unexpected response status %d with body %s", res.StatusCode, rs)
	}
	return nil
}
