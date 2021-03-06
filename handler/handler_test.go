package handler

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/epels/sparty/internal/mock"
)

const authToken = "secret"

func TestEnqueue(t *testing.T) {
	noopLogger := log.New(ioutil.Discard, "", 0)
	noopJobqueue := mock.Jobqueue{
		PutFunc: func(url string) error {
			return nil
		},
	}

	t.Run("Bad method", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/enqueue?url=foo", nil)
		setAuth(t, req)

		New(noopLogger, noopLogger, noopJobqueue, authToken).ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Got %d, expected 405", rec.Code)
		}
	})

	t.Run("Validation error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue?url=notmatching", nil)
		setAuth(t, req)

		New(noopLogger, noopLogger, noopJobqueue, authToken).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Got %d, expected 400", rec.Code)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Run("No token", func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/enqueue?url=foo", nil)

			New(noopLogger, noopLogger, noopJobqueue, authToken).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Got %d, expected 403", rec.Code)
			}
		})
		t.Run("Incorrect token", func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/enqueue?url=foo", nil)
			req.Header.Set("Authorization", "Token bad")

			New(noopLogger, noopLogger, noopJobqueue, authToken).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Got %d, expected 403", rec.Code)
			}
		})
	})

	t.Run("Jobqueue failure", func(t *testing.T) {
		vals := url.Values{}
		vals.Set("url", "https://open.spotify.com/track/1301WleyT98MSxVHPZCA6M?si=FY7aEiPCT0u3-CuNApJTRg")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue?"+vals.Encode(), nil)
		setAuth(t, req)

		var sb strings.Builder
		errLog := log.New(&sb, "", log.LstdFlags)
		jq := mock.Jobqueue{
			PutFunc: func(url string) error {
				return errors.New("some error")
			},
		}
		New(errLog, noopLogger, jq, authToken).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Got %d, expected 500", rec.Code)
		}
		if s := sb.String(); !strings.Contains(s, "some error") {
			t.Errorf("Got %q, expected to contain some error", s)
		}
	})

	t.Run("OK", func(t *testing.T) {
		vals := url.Values{}
		vals.Set("url", "https://open.spotify.com/track/1301WleyT98MSxVHPZCA6M?si=FY7aEiPCT0u3-CuNApJTRg")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue?"+vals.Encode(), nil)
		setAuth(t, req)

		var called bool
		jq := mock.Jobqueue{
			PutFunc: func(url string) error {
				called = true

				if url != "spotify:track:1301WleyT98MSxVHPZCA6M" {
					t.Errorf("Got %q, expected spotify:track:1301WleyT98MSxVHPZCA6M", url)
				}

				return nil
			},
		}
		New(noopLogger, noopLogger, jq, authToken).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Got %d, expected 204", rec.Code)
		}
		if !called {
			t.Error("Got false, expected true")
		}
	})
}

func setAuth(t *testing.T, r *http.Request) {
	t.Helper()

	r.Header.Set("Authorization", "Token "+authToken)
}

func TestParseSpotifyURL(t *testing.T) {
	t.Run("Matches", func(t *testing.T) {
		uri, err := parseSpotifyURL("https://open.spotify.com/track/1301WleyT98MSxVHPZCA6M?si=FY7aEiPCT0u3-CuNApJTRg")
		if err != nil {
			t.Fatalf("Got %T (%s), expected nil", err, err)
		}
		if uri != "spotify:track:1301WleyT98MSxVHPZCA6M" {
			t.Errorf("Got %q, expected spotify:track:1301WleyT98MSxVHPZCA6M", uri)
		}
	})

	t.Run("No match", func(t *testing.T) {
		_, err := parseSpotifyURL("https://open.spotify.com/track")
		if err == nil {
			t.Fatalf("Got nil, expected error")
		}
	})
}
