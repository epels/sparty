package spotify

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnsureToken(t *testing.T) {
	t.Run("No token yet", func(t *testing.T) {
		var called bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			expAuthHeader := "Basic Zm9vOmJhcg==" // "Basic base64(foo:bar)"
			if ah := r.Header.Get("Authorization"); ah != expAuthHeader {
				t.Errorf("Got %q, expected %q", ah, expAuthHeader)
			}
			if r.Method != http.MethodPost {
				t.Errorf("Got %q, expected POST", r.Method)
			}
			if r.URL.Path != "/api/token" {
				t.Errorf("Got %q, expected /api/token", r.URL.Path)
			}

			_, _ = fmt.Fprintf(w, `{"access_token":"secret","token_type":"Bearer","expires_in":3600,"scope":""}`)
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }

		if err := c.refreshToken(); err != nil {
			t.Fatalf("Got %T (%s), expected nil", err, err)
		}

		if !called {
			t.Error("Got false, expected true")
		}
		if c.token == nil {
			t.Fatal("Got nil, expected *token")
		}
		if c.token.bearer != "secret" {
			t.Errorf("Got %q, expected secret", c.token.bearer)
		}
		expExpiredAt := now.Add(3600 * time.Second)
		if !c.token.expiresAt.Equal(expExpiredAt) {
			t.Errorf("Got %s, expected %s", c.token.expiresAt, expExpiredAt)
		}
	})

	t.Run("Valid token", func(t *testing.T) {
		var called bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }
		c.token = &token{
			expiresAt: now.Add(10 * time.Second),
			bearer:    "secret",
		}

		if err := c.refreshToken(); err != nil {
			t.Fatalf("Got %T (%s), expected nil", err, err)
		}

		if called {
			t.Error("Got true, expected false")
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		var called bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			expAuthHeader := "Basic Zm9vOmJhcg==" // "Basic base64(foo:bar)"
			if ah := r.Header.Get("Authorization"); ah != expAuthHeader {
				t.Errorf("Got %q, expected %q", ah, expAuthHeader)
			}
			if r.Method != http.MethodPost {
				t.Errorf("Got %q, expected POST", r.Method)
			}
			if r.URL.Path != "/api/token" {
				t.Errorf("Got %q, expected /api/token", r.URL.Path)
			}

			_, _ = fmt.Fprintf(w, `{"access_token":"secret","token_type":"Bearer","expires_in":3600,"scope":""}`)
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }
		c.token = &token{
			bearer:    "expired",
			expiresAt: now.Add(4 * time.Second),
		}

		if err := c.refreshToken(); err != nil {
			t.Fatalf("Got %T (%s), expected nil", err, err)
		}

		if !called {
			t.Error("Got false, expected true")
		}
		if c.token == nil {
			t.Fatal("Got nil, expected *token")
		}
		if c.token.bearer != "secret" {
			t.Errorf("Got %q, expected secret", c.token.bearer)
		}
		expExpiredAt := now.Add(3600 * time.Second)
		if !c.token.expiresAt.Equal(expExpiredAt) {
			t.Errorf("Got %s, expected %s", c.token.expiresAt, expExpiredAt)
		}
	})
}
