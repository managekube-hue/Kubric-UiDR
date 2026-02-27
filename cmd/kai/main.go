package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/managekube-hue/Kubric-UiDR/internal/kai"
)

func main() {
	cfg := kai.LoadConfig()

	s, err := kai.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kai: init failed: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("kai: listening on %s\n", cfg.ListenAddr)
	if err := s.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kai: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("kai: stopped")
}
