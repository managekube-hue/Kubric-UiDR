package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	patchwf "github.com/managekube-hue/Kubric-UiDR/03_K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL"
)

const (
	// TaskQueuePatch is the Temporal task queue for patch workflows (Go activities).
	TaskQueuePatch = "kubric-patch"

	// TaskQueueRemediation is the Temporal task queue for remediation workflows.
	// The Go worker registers RemediationWorkflow on this queue; Python workers
	// on the same queue register the individual activities (validate_finding,
	// run_ansible, verify_remediation, close_finding) using the temporalio SDK.
	TaskQueueRemediation = "kubric-remediation"
)

// ─── RemediationWorkflow types ─────────────────────────────────────────────────

// RemediationInput carries the finding to be remediated.
type RemediationInput struct {
	FindingID  string            `json:"finding_id"`
	TenantID   string            `json:"tenant_id"`
	AssetID    string            `json:"asset_id"`
	CVEIDs     []string          `json:"cve_ids"`
	AutoApply  bool              `json:"auto_apply"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// RemediationResult is the final outcome of a remediation run.
type RemediationResult struct {
	FindingID   string    `json:"finding_id"`
	TenantID    string    `json:"tenant_id"`
	Status      string    `json:"status"` // completed|failed|rolled_back
	AnsibleJob  string    `json:"ansible_job,omitempty"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	ErrorDetail string    `json:"error_detail,omitempty"`
}

// activityResult is a small helper used by all cross-language activity calls.
type activityResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ─── RemediationWorkflow ────────────────────────────────────────────────────────
//
// Orchestrates remediation of a single finding by calling Python-registered
// activities on the kubric-remediation task queue (cross-language Temporal):
//
//  1. validate_finding   — confirm the finding still exists and is actionable
//  2. run_ansible        — apply Ansible playbook / patch
//  3. verify_remediation — smoke-test the fix
//  4. close_finding      — mark the finding resolved in all downstream systems
//
// All activities are executed with per-step timeouts and a single retry on
// transient failures.  If run_ansible or verify_remediation fails, the
// workflow returns an error so the caller (KAI-KEEPER) can decide whether to
// escalate via TheHive / Shuffle.
func RemediationWorkflow(ctx workflow.Context, input RemediationInput) (RemediationResult, error) {
	log := workflow.GetLogger(ctx)
	log.Info("RemediationWorkflow.started",
		"finding_id", input.FindingID,
		"tenant_id", input.TenantID,
		"auto_apply", input.AutoApply,
	)

	ao := workflow.ActivityOptions{
		TaskQueue:           TaskQueueRemediation,
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:        2,
			InitialInterval:        10 * time.Second,
			BackoffCoefficient:     2.0,
			NonRetryableErrorTypes: []string{"NonRetryableError"},
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, ao)

	// ── Step 1: validate_finding ──────────────────────────────────────────────
	var validateResult activityResult
	if err := workflow.ExecuteActivity(actCtx,
		"validate_finding", input,
	).Get(ctx, &validateResult); err != nil {
		log.Error("RemediationWorkflow.validate_failed", "error", err)
		return RemediationResult{
			FindingID:   input.FindingID,
			TenantID:    input.TenantID,
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("validate_finding: %v", err),
		}, err
	}
	if !validateResult.OK {
		return RemediationResult{
			FindingID:   input.FindingID,
			TenantID:    input.TenantID,
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("finding validation rejected: %s", validateResult.Message),
		}, temporal.NewNonRetryableApplicationError(validateResult.Message, "NonRetryableError", nil)
	}

	// ── Step 2: run_ansible ───────────────────────────────────────────────────
	if !input.AutoApply {
		log.Info("RemediationWorkflow.auto_apply_disabled_skipping_ansible",
			"finding_id", input.FindingID)
		return RemediationResult{
			FindingID: input.FindingID,
			TenantID:  input.TenantID,
			Status:    "completed",
			ClosedAt:  workflow.Now(ctx),
		}, nil
	}

	var ansibleResult activityResult
	if err := workflow.ExecuteActivity(actCtx,
		"run_ansible", input,
	).Get(ctx, &ansibleResult); err != nil {
		log.Error("RemediationWorkflow.ansible_failed", "error", err)
		return RemediationResult{
			FindingID:   input.FindingID,
			TenantID:    input.TenantID,
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("run_ansible: %v", err),
		}, err
	}

	// ── Step 3: verify_remediation ────────────────────────────────────────────
	var verifyResult activityResult
	if err := workflow.ExecuteActivity(actCtx,
		"verify_remediation", input,
	).Get(ctx, &verifyResult); err != nil {
		log.Error("RemediationWorkflow.verify_failed", "error", err)
		return RemediationResult{
			FindingID:   input.FindingID,
			TenantID:    input.TenantID,
			Status:      "failed",
			AnsibleJob:  ansibleResult.Message,
			ErrorDetail: fmt.Sprintf("verify_remediation: %v", err),
		}, err
	}

	// ── Step 4: close_finding ─────────────────────────────────────────────────
	var closeResult activityResult
	if err := workflow.ExecuteActivity(actCtx,
		"close_finding", input,
	).Get(ctx, &closeResult); err != nil {
		// Non-fatal: finding is remediated even if the close call fails.
		log.Warn("RemediationWorkflow.close_finding_failed", "error", err)
	}

	log.Info("RemediationWorkflow.completed", "finding_id", input.FindingID)
	return RemediationResult{
		FindingID: input.FindingID,
		TenantID:  input.TenantID,
		Status:    "completed",
		AnsibleJob: ansibleResult.Message,
		ClosedAt:   workflow.Now(ctx),
	}, nil
}

// NoopActivity is a placeholder registered alongside RemediationWorkflow so
// the kubric-remediation task queue always has at least one Go activity.
// Real remediation activities are registered by the Python temporalio worker
// (kai/workers/temporal_worker.py).
func NoopActivity(ctx context.Context) error {
	_ = activity.GetInfo(ctx)
	return nil
}

// ─── main ────────────────────────────────────────────────────────────────────

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

	// ── Worker 1: kubric-patch ────────────────────────────────────────────────
	// Runs PatchWorkflow and all its Go activities (validate, snapshot, apply,
	// verify, notify, rollback).  Defined in K-KAI-WF-TEMP-001_patch_workflow.go.
	patchWorker := worker.New(c, TaskQueuePatch, worker.Options{
		MaxConcurrentActivityExecutionSize:      20,
		MaxConcurrentWorkflowTaskExecutionSize:  10,
		MaxConcurrentLocalActivityExecutionSize: 10,
	})
	patchwf.RegisterWorkflows(patchWorker) // registers PatchWorkflow + activities

	// ── Worker 2: kubric-remediation ─────────────────────────────────────────
	// Runs RemediationWorkflow (Go).  Python temporalio worker registers the
	// individual activities (validate_finding, run_ansible, verify_remediation,
	// close_finding) on the same task queue, enabling cross-language execution.
	remediationWorker := worker.New(c, TaskQueueRemediation, worker.Options{
		MaxConcurrentActivityExecutionSize:      10,
		MaxConcurrentWorkflowTaskExecutionSize:  5,
		MaxConcurrentLocalActivityExecutionSize: 5,
	})
	remediationWorker.RegisterWorkflow(RemediationWorkflow)
	remediationWorker.RegisterActivity(NoopActivity)

	fmt.Printf(
		"temporal-worker: listening on task-queues=[%q, %q] namespace=%q host=%s\n",
		TaskQueuePatch, TaskQueueRemediation, namespace, temporalHost,
	)

	// Start both workers; if either fails to start, abort.
	if err := patchWorker.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "temporal-worker: patch worker start failed: %v\n", err)
		os.Exit(1)
	}
	if err := remediationWorker.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "temporal-worker: remediation worker start failed: %v\n", err)
		patchWorker.Stop()
		os.Exit(1)
	}

	// Block until SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("temporal-worker: shutting down")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); patchWorker.Stop() }()
	go func() { defer wg.Done(); remediationWorker.Stop() }()
	wg.Wait()
	fmt.Println("temporal-worker: stopped")
}
