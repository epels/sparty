package spotify

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerToken(t *testing.T) {
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
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("Got %q, expected application/x-www-form-urlencoded", ct)
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("Got %T (%s), expected nil", err, err)
			}
			if gt := r.PostForm["grant_type"]; gt[0] != "refresh_token" {
				t.Errorf("Got %q, expected refresh_token", gt[0])
			}
			if rt := r.PostForm["refresh_token"]; rt[0] != "baz" {
				t.Errorf("Got %q, expected baz", rt[0])
			}

			_, _ = fmt.Fprint(w, `{"access_token":"secret","token_type":"Bearer","expires_in":3600,"scope":"user-modify-playback-state"}`)
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar", "baz")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }

		if err := c.bearerToken(context.Background()); err != nil {
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
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Auth endpoint called even though a valid token is present")
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar", "baz")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }
		c.token = &token{
			expiresAt: now.Add(10 * time.Second),
			bearer:    "secret",
		}

		if err := c.bearerToken(context.Background()); err != nil {
			t.Fatalf("Got %T (%s), expected nil", err, err)
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
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("Got %q, expected application/x-www-form-urlencoded", ct)
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("Got %T (%s), expected nil", err, err)
			}
			if gt := r.PostForm["grant_type"]; gt[0] != "refresh_token" {
				t.Errorf("Got %q, expected refresh_token", gt[0])
			}
			if rt := r.PostForm["refresh_token"]; rt[0] != "baz" {
				t.Errorf("Got %q, expected baz", rt[0])
			}

			_, _ = fmt.Fprint(w, `{"access_token":"secret","token_type":"Bearer","expires_in":3600,"scope":"user-modify-playback-state"}`)
		}))
		defer ts.Close()

		now, _ := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		c := NewClient("foo", "bar", "baz")
		c.authBaseURL = ts.URL
		c.nowFunc = func() time.Time { return now }
		c.token = &token{
			bearer:    "expired",
			expiresAt: now.Add(4 * time.Second),
		}

		if err := c.bearerToken(context.Background()); err != nil {
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

func TestAPIRequest(t *testing.T) {
	apiTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ah := r.Header.Get("Authorization"); ah != "Bearer secret" {
			t.Errorf("Got %q, expected Bearer secret", ah)
		}

		if r.Method != http.MethodPatch {
			t.Errorf("Got %q, expected POST", r.Method)
		}
		if r.URL.Path != "/foo" {
			t.Errorf("Got %q, expected /foo", r.URL.Path)
		}
		if v := r.URL.Query().Get("key"); v != "val" {
			t.Errorf("Got %q, expected val", v)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Got %q, expected application/json", ct)
		}
		b, _ := ioutil.ReadAll(r.Body)
		if s := string(b); s != `{"foo":"bar"}` {
			t.Errorf(`Got %q, expected {"foo":"bar"}`, s)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "hello")
	}))
	defer apiTS.Close()
	authTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"access_token":"secret","token_type":"Bearer","expires_in":3600,"scope":"user-modify-playback-state"}`)
	}))
	defer authTS.Close()

	c := NewClient("foo", "bar", "baz")
	c.apiBaseURL = apiTS.URL
	c.authBaseURL = authTS.URL

	res, err := c.apiRequest(context.Background(), http.MethodPatch, "/foo?key=val", struct {
		Foo string `json:"foo"`
	}{
		Foo: "bar",
	})
	if err != nil {
		t.Fatalf("Got %T (%s), expected nil", err, err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("Got %d, expected 201", res.StatusCode)
	}
	b, _ := ioutil.ReadAll(res.Body)
	if s := string(b); s != "hello" {
		t.Errorf("Got %q, expected hello", b)
	}
}

func TestAddToQueue(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		var called bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			if r.URL.Path != "/v1/me/player/add-to-queue" {
				t.Errorf("Got %q, expected /v1/me/player/add-to-queue", r.URL.Path)
			}
			if uri := r.URL.Query().Get("uri"); uri != "foo" {
				t.Errorf("Got %q, expected foo", uri)
			}

			w.WriteHeader(http.StatusNoContent)
		}))

		c := NewClient("foo", "bar", "baz")
		c.apiBaseURL = ts.URL
		c.token = &token{
			bearer:    "secret",
			expiresAt: c.nowFunc().Add(1800 * time.Second),
		}
		if err := c.AddToQueue(context.Background(), "foo"); err != nil {
			t.Errorf("Got %T (%s), expected nil", err, err)
		}
		if !called {
			t.Error("Got false, expected true")
		}
	})

	t.Run("Bad response", func(t *testing.T) {
		var called bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			if r.URL.Path != "/v1/me/player/add-to-queue" {
				t.Errorf("Got %q, expected /v1/me/player/add-to-queue", r.URL.Path)
			}
			if uri := r.URL.Query().Get("uri"); uri != "foo" {
				t.Errorf("Got %q, expected foo", uri)
			}

			w.WriteHeader(http.StatusInternalServerError)
		}))

		c := NewClient("foo", "bar", "baz")
		c.apiBaseURL = ts.URL
		c.token = &token{
			bearer:    "secret",
			expiresAt: c.nowFunc().Add(1800 * time.Second),
		}
		if err := c.AddToQueue(context.Background(), "foo"); err == nil {
			t.Error("Got nil, expected error")
		}
		if !called {
			t.Error("Got false, expected true")
		}
	})
}
