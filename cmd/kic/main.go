package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/managekube-hue/Kubric-UiDR/internal/kic"
)

func main() {
	cfg := kic.LoadConfig()

	srv, err := kic.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kic: init failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("kic: listening on %s\n", cfg.ListenAddr)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kic: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("kic: stopped")
}
