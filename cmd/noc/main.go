package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/managekube-hue/Kubric-UiDR/internal/noc"
)

func main() {
	cfg := noc.LoadConfig()

	srv, err := noc.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "noc: init failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("noc: listening on %s\n", cfg.ListenAddr)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "noc: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("noc: stopped")
}
