package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/managekube-hue/Kubric-UiDR/internal/vdr"
)

func main() {
	cfg := vdr.LoadConfig()

	srv, err := vdr.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdr: init failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("vdr: listening on %s\n", cfg.ListenAddr)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "vdr: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("vdr: stopped")
}
