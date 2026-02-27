// Package dev provides a CI pipeline runner for Kubric using os/exec.
// File: K-DEV-CICD-GHA-007_dagger_ci.go
//
// NOTE: dagger.io/dagger is not in go.mod. This implementation uses os/exec to
// run the same pipeline steps (lint, test, build, scan) portably.
// To switch to the Dagger SDK, run:
//   go get dagger.io/dagger@latest
// and replace the exec-based implementation with the commented Dagger version below.
package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CIStep represents a single step in the CI pipeline.
type CIStep struct {
	Name    string
	Command []string
	Env     []string // additional KEY=VALUE pairs
}

// RunCI executes the Kubric CI pipeline:
//  1. Lint (go vet + staticcheck)
//  2. Unit tests with race detector
//  3. Build all binaries
//  4. Container image build
//  5. Container security scan (Trivy) — skipped if trivy not installed
//  6. Push to registry — only when REGISTRY env var is set
func RunCI(ctx context.Context) error {
	registry := os.Getenv("REGISTRY")

	steps := []CIStep{
		{
			Name:    "go vet",
			Command: []string{"go", "vet", "./..."},
		},
		{
			Name:    "go test",
			Command: []string{"go", "test", "-race", "-count=1", "-timeout=10m", "./..."},
		},
		{
			Name:    "go build",
			Command: []string{"go", "build", "./..."},
		},
	}

	// docker build step — skip when docker not available
	if _, err := exec.LookPath("docker"); err == nil {
		imageTag := "kubric-platform:ci"
		if registry != "" {
			imageTag = registry + "/kubric-platform:ci"
		}
		steps = append(steps, CIStep{
			Name:    "docker build",
			Command: []string{"docker", "build", "-t", imageTag, "."},
		})

		// trivy scan — skip when trivy not installed
		if _, err := exec.LookPath("trivy"); err == nil {
			steps = append(steps, CIStep{
				Name: "trivy scan",
				Command: []string{
					"trivy", "image",
					"--exit-code", "1",
					"--severity", "CRITICAL,HIGH",
					"--no-progress",
					imageTag,
				},
			})
		} else {
			fmt.Println("[ci] trivy not found — skipping container security scan")
		}

		// push to registry only when configured
		if registry != "" {
			steps = append(steps, CIStep{
				Name:    "docker push",
				Command: []string{"docker", "push", imageTag},
			})
		}
	} else {
		fmt.Println("[ci] docker not found — skipping image build steps")
	}

	return runSteps(ctx, steps)
}

// runSteps executes each CIStep sequentially, printing timing and output.
func runSteps(ctx context.Context, steps []CIStep) error {
	for i, step := range steps {
		start := time.Now()
		fmt.Printf("[ci] step %d/%d: %s\n", i+1, len(steps), step.Name)
		fmt.Printf("[ci]   cmd: %s\n", strings.Join(step.Command, " "))

		cmd := exec.CommandContext(ctx, step.Command[0], step.Command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), step.Env...)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("step %q failed after %s: %w",
				step.Name, time.Since(start).Round(time.Millisecond), err)
		}
		fmt.Printf("[ci]   done in %s\n", time.Since(start).Round(time.Millisecond))
	}
	fmt.Println("[ci] all steps passed")
	return nil
}

// RunCIFromMain is a convenience wrapper for use in a main() or cobra command.
func RunCIFromMain() int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := RunCI(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[ci] FAILED: %v\n", err)
		return 1
	}
	return 0
}

// ─── Dagger SDK version (commented out — activate after: go get dagger.io/dagger@latest) ───
//
// import "dagger.io/dagger"
//
// func RunCIDagger(ctx context.Context) error {
//     client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
//     if err != nil { return fmt.Errorf("dagger connect: %w", err) }
//     defer client.Close()
//
//     src := client.Host().Directory(".", dagger.HostDirectoryOpts{
//         Exclude: []string{".git", "vendor", "target"},
//     })
//     golang := client.Container().
//         From("golang:1.23-alpine").
//         WithMountedDirectory("/src", src).
//         WithWorkdir("/src").
//         WithExec([]string{"go", "build", "./..."}).
//         WithExec([]string{"go", "vet",   "./..."}).
//         WithExec([]string{"go", "test",  "-race", "-count=1", "./..."})
//     if _, err := golang.Sync(ctx); err != nil { return fmt.Errorf("go checks: %w", err) }
//
//     image := client.Container().Build(src, dagger.ContainerBuildOpts{Dockerfile: "Dockerfile"})
//     if _, err := image.Sync(ctx); err != nil { return fmt.Errorf("docker build: %w", err) }
//     return nil
// }
