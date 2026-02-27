package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	patchwf "github.com/managekube-hue/Kubric-UiDR/03_K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL"
)

const (
	// TaskQueuePatch is the Temporal task queue for patch and remediation workflows.
	TaskQueuePatch = "kubric-patch"
)

func main() {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "temporal-worker: dial failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, TaskQueuePatch, worker.Options{
		MaxConcurrentActivityExecutionSize:      20,
		MaxConcurrentWorkflowTaskExecutionSize:  10,
		MaxConcurrentLocalActivityExecutionSize: 10,
	})

	// Register all K-KAI patch workflows and activities.
	patchwf.RegisterWorkflows(w)

	fmt.Printf("temporal-worker: listening on task-queue=%q namespace=%q host=%s\n",
		TaskQueuePatch, namespace, temporalHost)

	// Start the worker in background; block until SIGINT/SIGTERM.
	if err := w.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "temporal-worker: start failed: %v\n", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("temporal-worker: shutting down")
	w.Stop()
}
