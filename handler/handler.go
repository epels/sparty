package handler

import (
	"fmt"
	"log"
	"net/http"
)

// @todo: Add some basic auth.
type handler struct {
	http.Handler

	errLog, infoLog *log.Logger
	jq              jobqueue
	token           string
}

type jobqueue interface {
	// Put puts a job into the jobqueue that will, upon consumption by the
	// worker, enqueue the referenced song in Spotify.
	Put(url string) error
}

var _ http.Handler = (*handler)(nil) // Compile-time assurance.

func New(errLog, infoLog *log.Logger, jq jobqueue, token string) *handler {
	h := handler{
		errLog:  errLog,
		infoLog: infoLog,
		jq:      jq,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/enqueue", h.method(http.MethodPost, h.auth(token, h.log(h.enqueue))))
	h.Handler = mux

	return &h
}

func (h *handler) auth(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t := r.Header.Get("Authorization"); t != "Token "+token {
			h.infoLog.Printf("Failed auth attempt from %q @ %q", r.UserAgent(), r.RemoteAddr)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (h *handler) log(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.infoLog.Printf("%s %s (%q @ %q)", r.Method, r.URL, r.UserAgent(), r.RemoteAddr)
		next(w, r)
	}
}

func (h *handler) method(m string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != m {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// enqueue accepts a song by its Spotify url and sticks a job into the jobqueue
// to actually send it over to the Spotify Web API. Handler responds with a 204
// if the song is accepted for delivery, but this does not guarantee it will
// actually play.
func (h *handler) enqueue(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "Missing required parameter: url")
		return
	}

	if err := h.jq.Put(url); err != nil {
		h.errLog.Printf("%T: Put: %s", h.jq, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
