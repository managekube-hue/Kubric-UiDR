package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/managekube-hue/Kubric-UiDR/internal/ksvc"
)

func main() {
	cfg := ksvc.LoadConfig()

	srv, err := ksvc.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ksvc: init failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("ksvc: listening on %s\n", cfg.ListenAddr)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ksvc: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ksvc: stopped")
}
