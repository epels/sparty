package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
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
	Put(uri string) error
}

var (
	_ http.Handler = (*handler)(nil) // Compile-time assurance.

	spotifyURLRe = regexp.MustCompile("^https:\\/\\/open.spotify\\..*\\/track\\/(.*)\\?si=.*$")
)

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

	uri, err := parseSpotifyURL(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "Invalid value for parameter: url (%s)\n", url)
		return
	}

	if err := h.jq.Put(uri); err != nil {
		h.errLog.Printf("%T: Put: %s", h.jq, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseSpotifyURL parses a full Spotify URL in the Spotify app's sharing
// format, e.g. https://open.spotify.com/track/1301WleyT98MSxVHPZCA6M?si=FY7aEiPCT0u3-CuNApJTRg,
// to its Spotify "URI": spotify:track:1301WleyT98MSxVHPZCA6M.
func parseSpotifyURL(url string) (string, error) {
	uriSubs := spotifyURLRe.FindStringSubmatch(url)
	if len(uriSubs) != 2 {
		return "", errors.New("url is not a valid Spotify track URL")
	}
	return "spotify:track:" + uriSubs[1], nil
}
