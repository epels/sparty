package handler

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epels/sparty/internal/mock"
)

func TestEnqueue(t *testing.T) {
	noopLogger := log.New(ioutil.Discard, "", 0)
	noopJobqueue := mock.Jobqueue{
		PutFunc: func(uri string) error {
			return nil
		},
	}

	t.Run("Bad method", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/enqueue?uri=foo", nil)

		New(noopLogger, noopLogger, noopJobqueue).ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Got %d, expected 405", rec.Code)
		}
	})

	t.Run("Validation error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue", nil)

		New(noopLogger, noopLogger, noopJobqueue).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Got %d, expected 400", rec.Code)
		}
	})

	t.Run("Jobqueue failure", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue?uri=foo", nil)

		var sb strings.Builder
		errLog := log.New(&sb, "", log.LstdFlags)
		jq := mock.Jobqueue{
			PutFunc: func(uri string) error {
				return errors.New("some error")
			},
		}
		New(errLog, noopLogger, jq).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Got %d, expected 500", rec.Code)
		}
		if s := sb.String(); !strings.Contains(s, "some error") {
			t.Errorf("Got %q, expected to contain some error", s)
		}
	})

	t.Run("OK", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/enqueue?uri=foo", nil)

		var called bool
		jq := mock.Jobqueue{
			PutFunc: func(uri string) error {
				called = true

				if uri != "foo" {
					t.Errorf("Got %q, expected foo", uri)
				}

				return nil
			},
		}
		New(noopLogger, noopLogger, jq).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Got %d, expected 204", rec.Code)
		}
		if !called {
			t.Error("Got false, expected true")
		}
	})
}
