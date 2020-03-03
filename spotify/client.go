package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type client struct {
	httpc                   *http.Client
	authBaseURL, authHeader string
	token                   *token

	// nowFunc returns the current local time. Can be used to instrument tests.
	nowFunc func() time.Time
}

type token struct {
	bearer    string
	expiresAt time.Time
}

// If the token expires within this duration, request a new one anyway: this
// guards against the next request still failing due to an expired token.
const expiryThreshold = 5 * time.Second

func NewClient(cID, cSecret string) *client {
	ah := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cID, cSecret)))
	return &client{
		httpc: &http.Client{
			Timeout: 5 * time.Second,
		},
		authBaseURL: "https://accounts.spotify.com",
		authHeader:  ah,
		nowFunc:     time.Now,
	}
}

func (c *client) refreshToken() error {
	if c.token != nil && c.token.bearer != "" {
		// Already have a token, so no-op if it doesn't expire in the near
		// future.
		if !c.token.expiresAt.Before(c.nowFunc().Add(expiryThreshold)) {
			return nil
		}
	}

	u := c.authBaseURL + "/api/token?grant_type=client_credentials"
	req, err := http.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return fmt.Errorf("net/http: NewRequest: %s", err)
	}
	req.Header.Set("Authorization", c.authHeader)

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
