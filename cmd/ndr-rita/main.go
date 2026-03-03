package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/managekube-hue/Kubric-UiDR/internal/ndr/rita"
)

func main() {
	listenAddr := getenv("NDR_RITA_LISTEN", ":4096")
	clickhouseURL := getenv("CLICKHOUSE_URL", "")

	svc, err := rita.New(clickhouseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ndr-rita init failed: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           svc.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "ndr-rita server error: %v\n", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
