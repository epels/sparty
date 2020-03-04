package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/epels/sparty/handler"
	"github.com/epels/sparty/jobqueue"
	"github.com/epels/sparty/spotify"
)

var (
	errLog  = log.New(os.Stderr, "[ERROR]: ", log.LstdFlags|log.Lshortfile)
	infoLog = log.New(os.Stdout, "[INFO]: ", log.LstdFlags|log.Lshortfile)
)

var (
	spartyAuthToken     = mustGetenv("SPARTY_AUTH_TOKEN")
	spotifyClientID     = mustGetenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = mustGetenv("SPOTIFY_CLIENT_SECRET")
	spotifyRefreshToken = mustGetenv("SPOTIFY_REFRESH_TOKEN")
)

func main() {
	// PORT is set by Google App Engine.
	p := os.Getenv("PORT")
	if p == "" {
		p = "8080"
	}
	addr := ":" + p
	jq := jobqueue.NewMemory()
	sc := spotify.NewClient(spotifyClientID, spotifyClientSecret, spotifyRefreshToken)

	// Channels that can cancel the execution of the daemon.
	errCh := make(chan error, 2)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the job consumer/worker.
	jqCtx, jqCancel := context.WithCancel(context.Background())
	defer jqCancel()
	go func() {
		infoLog.Print("Starting job worker")
		err := jq.Consume(jqCtx, func(url string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := sc.AddToQueue(ctx, url); err != nil {
				errLog.Printf("spotify: Client.AddToQueue: %s", err)
				return
			}
			infoLog.Printf("Enqueued %s", url)
		})
		errCh <- fmt.Errorf("jobqueue: memory.Consume: %s", err)
	}()

	// Create the API server and start listening.
	h := handler.New(errLog, infoLog, jq, spartyAuthToken)
	s := http.Server{
		Addr:    addr,
		Handler: h,

		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go func() {
		infoLog.Printf("Starting server on %q", addr)
		err := s.ListenAndServe()
		errCh <- fmt.Errorf("net/http: Server.ListenAndServe: %s", err)
	}()

	// Handle shutdown due to API error, job consumer failure, or signal.
	select {
	case err := <-errCh:
		errLog.Printf("Exiting with error: %s", err)
	case sig := <-sigCh:
		infoLog.Printf("Exiting with signal: %s", sig)
	}

	sCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer sCancel()
	if err := s.Shutdown(sCtx); err != nil {
		errLog.Printf("net/http: Server.Shutdown: %v", err)
	}
	if err := jq.Close(); err != nil {
		errLog.Printf("jobqueue: memory.Close: %s", err)
	}
}

func mustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		errLog.Fatalf("Missing required environment variable: %s", key)
	}
	return val
}
