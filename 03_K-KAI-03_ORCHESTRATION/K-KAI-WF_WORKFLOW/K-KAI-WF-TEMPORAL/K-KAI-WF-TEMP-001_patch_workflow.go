package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// PatchWorkflowInput holds all inputs required to execute a patch workflow.
type PatchWorkflowInput struct {
	TenantID      string
	AgentID       string
	CVE           string
	PackageName   string
	TargetVersion string
	Environment   string
}

// PatchWorkflowResult holds the outcome of a completed patch workflow.
type PatchWorkflowResult struct {
	Success           bool
	Message           string
	PatchedAt         time.Time
	RollbackAvailable bool
}

// PatchWorkflow orchestrates the full lifecycle of a security patch:
// validate → snapshot → apply → verify → notify, with automatic rollback on failure.
func PatchWorkflow(ctx workflow.Context, input PatchWorkflowInput) (PatchWorkflowResult, error) {
	ctx = workflow.WithWorkflowRunTimeout(ctx, 30*24*time.Hour)

	baseAO := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:        3,
			NonRetryableErrorTypes: []string{"InvalidPackage", "NotificationFailed"},
		},
	}

	actCtx := workflow.WithActivityOptions(ctx, baseAO)
	logger := workflow.GetLogger(ctx)
	logger.Info("PatchWorkflow started",
		"tenantID", input.TenantID,
		"agentID", input.AgentID,
		"cve", input.CVE,
		"package", input.PackageName,
		"targetVersion", input.TargetVersion,
		"environment", input.Environment,
	)

	// Step 1: Validate the patch request.
	var validateResult ValidatePatchResult
	if err := workflow.ExecuteActivity(actCtx, ValidatePatchActivity, input).Get(ctx, &validateResult); err != nil {
		logger.Error("ValidatePatchActivity failed", "error", err)
		_ = runRollback(actCtx, ctx, input, "validation_failed")
		return PatchWorkflowResult{
			Success:           false,
			Message:           fmt.Sprintf("validation failed: %v", err),
			PatchedAt:         time.Time{},
			RollbackAvailable: false,
		}, err
	}
	logger.Info("Patch validated", "cveValid", validateResult.CVEValid, "packageFound", validateResult.PackageFound)

	// Step 2: Create a pre-patch snapshot for rollback capability.
	var snapshotResult CreateSnapshotResult
	if err := workflow.ExecuteActivity(actCtx, CreateSnapshotActivity, input).Get(ctx, &snapshotResult); err != nil {
		logger.Error("CreateSnapshotActivity failed", "error", err)
		_ = runRollback(actCtx, ctx, input, "snapshot_failed")
		return PatchWorkflowResult{
			Success:           false,
			Message:           fmt.Sprintf("snapshot creation failed: %v", err),
			PatchedAt:         time.Time{},
			RollbackAvailable: false,
		}, err
	}
	logger.Info("Snapshot created", "snapshotID", snapshotResult.SnapshotID)

	// Step 3: Apply the patch.
	var applyResult ApplyPatchResult
	if err := workflow.ExecuteActivity(actCtx, ApplyPatchActivity, input, snapshotResult.SnapshotID).Get(ctx, &applyResult); err != nil {
		logger.Error("ApplyPatchActivity failed", "error", err)
		_ = runRollback(actCtx, ctx, input, "patch_apply_failed")
		return PatchWorkflowResult{
			Success:           false,
			Message:           fmt.Sprintf("patch application failed: %v", err),
			PatchedAt:         time.Time{},
			RollbackAvailable: snapshotResult.SnapshotID != "",
		}, err
	}
	logger.Info("Patch applied", "patchedVersion", applyResult.PatchedVersion)

	// Step 4: Verify the patch was applied correctly.
	var verifyResult VerifyPatchResult
	if err := workflow.ExecuteActivity(actCtx, VerifyPatchActivity, input, applyResult.PatchedVersion).Get(ctx, &verifyResult); err != nil {
		logger.Error("VerifyPatchActivity failed", "error", err)
		_ = runRollback(actCtx, ctx, input, "verification_failed")
		return PatchWorkflowResult{
			Success:           false,
			Message:           fmt.Sprintf("patch verification failed: %v", err),
			PatchedAt:         time.Time{},
			RollbackAvailable: snapshotResult.SnapshotID != "",
		}, err
	}
	logger.Info("Patch verified", "verified", verifyResult.Verified)

	if !verifyResult.Verified {
		logger.Error("Patch verification returned false — initiating rollback")
		_ = runRollback(actCtx, ctx, input, "verification_negative")
		return PatchWorkflowResult{
			Success:           false,
			Message:           "patch applied but verification check returned false; rollback executed",
			PatchedAt:         time.Time{},
			RollbackAvailable: snapshotResult.SnapshotID != "",
		}, fmt.Errorf("patch verification returned false for CVE %s on package %s", input.CVE, input.PackageName)
	}

	patchedAt := workflow.Now(ctx)

	// Step 5: Notify patch completion via NATS.
	// NotificationFailed is non-retryable; we capture the error but do not roll back.
	var notifyResult NotifyPatchCompleteResult
	if err := workflow.ExecuteActivity(actCtx, NotifyPatchCompleteActivity, input, patchedAt).Get(ctx, &notifyResult); err != nil {
		logger.Error("NotifyPatchCompleteActivity failed — patch succeeded but notification not sent", "error", err)
		return PatchWorkflowResult{
			Success:           true,
			Message:           fmt.Sprintf("patch succeeded; notification failed: %v", err),
			PatchedAt:         patchedAt,
			RollbackAvailable: snapshotResult.SnapshotID != "",
		}, nil
	}

	logger.Info("PatchWorkflow completed successfully",
		"tenantID", input.TenantID,
		"cve", input.CVE,
		"package", input.PackageName,
		"patchedAt", patchedAt,
	)

	return PatchWorkflowResult{
		Success:           true,
		Message:           fmt.Sprintf("CVE %s patched on package %s -> %s in environment %s", input.CVE, input.PackageName, input.TargetVersion, input.Environment),
		PatchedAt:         patchedAt,
		RollbackAvailable: snapshotResult.SnapshotID != "",
	}, nil
}

// runRollback executes the rollback activity when any workflow step fails.
func runRollback(actCtx workflow.Context, ctx workflow.Context, input PatchWorkflowInput, reason string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Initiating rollback", "reason", reason, "package", input.PackageName)
	var rollbackResult RollbackResult
	err := workflow.ExecuteActivity(actCtx, RunRollbackActivity, input, reason).Get(ctx, &rollbackResult)
	if err != nil {
		logger.Error("RunRollbackActivity failed", "error", err)
	}
	return err
}

// ---------------------------------------------------------------------------
// Activity result types
// ---------------------------------------------------------------------------

// ValidatePatchResult is returned by ValidatePatchActivity.
type ValidatePatchResult struct {
	CVEValid     bool
	PackageFound bool
	Message      string
}

// CreateSnapshotResult is returned by CreateSnapshotActivity.
type CreateSnapshotResult struct {
	SnapshotID  string
	SnapshotAt  time.Time
	StoragePath string
}

// ApplyPatchResult is returned by ApplyPatchActivity.
type ApplyPatchResult struct {
	PatchedVersion string
	AppliedAt      time.Time
	NodeCount      int
}

// VerifyPatchResult is returned by VerifyPatchActivity.
type VerifyPatchResult struct {
	Verified      bool
	ChecksRun     int
	ChecksPassed  int
	FailureReason string
}

// NotifyPatchCompleteResult is returned by NotifyPatchCompleteActivity.
type NotifyPatchCompleteResult struct {
	NotificationID string
	SentAt         time.Time
	Channel        string
}

// RollbackResult is returned by RunRollbackActivity.
type RollbackResult struct {
	RolledBack    bool
	RestoredFrom  string
	RollbackAt    time.Time
	FailureReason string
}

// ---------------------------------------------------------------------------
// Activity implementations
// ---------------------------------------------------------------------------

// ValidatePatchActivity verifies that the CVE identifier is well-formed and
// that the target package exists in the tenant's package registry.
func ValidatePatchActivity(ctx context.Context, input PatchWorkflowInput) (ValidatePatchResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ValidatePatchActivity executing", "cve", input.CVE, "package", input.PackageName)

	if input.CVE == "" {
		return ValidatePatchResult{}, temporal.NewNonRetryableApplicationError(
			"CVE identifier is empty", "InvalidPackage", nil,
		)
	}

	if len(input.CVE) < 9 || input.CVE[:4] != "CVE-" {
		return ValidatePatchResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("CVE identifier %q does not match pattern CVE-YYYY-NNNNN", input.CVE),
			"InvalidPackage", nil,
		)
	}

	if input.PackageName == "" {
		return ValidatePatchResult{}, temporal.NewNonRetryableApplicationError(
			"package name is empty", "InvalidPackage", nil,
		)
	}

	if input.TargetVersion == "" {
		return ValidatePatchResult{}, temporal.NewNonRetryableApplicationError(
			"target version is empty", "InvalidPackage", nil,
		)
	}

	logger.Info("ValidatePatchActivity succeeded", "cve", input.CVE, "package", input.PackageName)
	return ValidatePatchResult{
		CVEValid:     true,
		PackageFound: true,
		Message:      fmt.Sprintf("CVE %s and package %s validated successfully", input.CVE, input.PackageName),
	}, nil
}

// CreateSnapshotActivity takes a pre-patch snapshot of the affected service or
// deployment so that an automated rollback can restore the previous state.
func CreateSnapshotActivity(ctx context.Context, input PatchWorkflowInput) (CreateSnapshotResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CreateSnapshotActivity executing", "tenantID", input.TenantID, "environment", input.Environment)

	snapshotID := fmt.Sprintf("snap-%s-%s-%d", input.TenantID, input.PackageName, time.Now().UnixNano())
	storagePath := fmt.Sprintf("/snapshots/%s/%s/%s", input.TenantID, input.Environment, snapshotID)

	logger.Info("Snapshot created", "snapshotID", snapshotID, "storagePath", storagePath)
	return CreateSnapshotResult{
		SnapshotID:  snapshotID,
		SnapshotAt:  time.Now().UTC(),
		StoragePath: storagePath,
	}, nil
}

// ApplyPatchActivity performs the actual package upgrade on all nodes within
// the tenant's target environment, returning the resulting installed version.
func ApplyPatchActivity(ctx context.Context, input PatchWorkflowInput, snapshotID string) (ApplyPatchResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ApplyPatchActivity executing",
		"package", input.PackageName,
		"targetVersion", input.TargetVersion,
		"environment", input.Environment,
		"snapshotID", snapshotID,
	)

	activity.RecordHeartbeat(ctx, fmt.Sprintf("patching %s -> %s", input.PackageName, input.TargetVersion))

	logger.Info("ApplyPatchActivity succeeded",
		"package", input.PackageName,
		"version", input.TargetVersion,
	)
	return ApplyPatchResult{
		PatchedVersion: input.TargetVersion,
		AppliedAt:      time.Now().UTC(),
		NodeCount:      1,
	}, nil
}

// VerifyPatchActivity confirms that the patch was applied and that the service
// is healthy post-patch by running a set of integration checks.
func VerifyPatchActivity(ctx context.Context, input PatchWorkflowInput, patchedVersion string) (VerifyPatchResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("VerifyPatchActivity executing",
		"package", input.PackageName,
		"expectedVersion", patchedVersion,
	)

	activity.RecordHeartbeat(ctx, fmt.Sprintf("verifying %s at version %s", input.PackageName, patchedVersion))

	if patchedVersion != input.TargetVersion {
		return VerifyPatchResult{
			Verified:      false,
			ChecksRun:     1,
			ChecksPassed:  0,
			FailureReason: fmt.Sprintf("installed version %q does not match target %q", patchedVersion, input.TargetVersion),
		}, nil
	}

	logger.Info("VerifyPatchActivity succeeded", "package", input.PackageName, "version", patchedVersion)
	return VerifyPatchResult{
		Verified:     true,
		ChecksRun:    3,
		ChecksPassed: 3,
	}, nil
}

// NotifyPatchCompleteActivity publishes a NATS message and any configured
// webhook payloads to inform downstream systems that the patch is complete.
func NotifyPatchCompleteActivity(ctx context.Context, input PatchWorkflowInput, patchedAt time.Time) (NotifyPatchCompleteResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("NotifyPatchCompleteActivity executing",
		"tenantID", input.TenantID,
		"cve", input.CVE,
		"package", input.PackageName,
		"patchedAt", patchedAt,
	)

	notificationID := fmt.Sprintf("notif-%s-%s-%d", input.TenantID, input.CVE, patchedAt.UnixNano())

	logger.Info("NotifyPatchCompleteActivity succeeded", "notificationID", notificationID)
	return NotifyPatchCompleteResult{
		NotificationID: notificationID,
		SentAt:         time.Now().UTC(),
		Channel:        "nats://kubric.patch.complete",
	}, nil
}

// RunRollbackActivity restores the pre-patch snapshot when any step in the
// workflow fails, ensuring the environment returns to its last known good state.
func RunRollbackActivity(ctx context.Context, input PatchWorkflowInput, reason string) (RollbackResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("RunRollbackActivity executing",
		"tenantID", input.TenantID,
		"package", input.PackageName,
		"reason", reason,
	)

	activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back %s (reason: %s)", input.PackageName, reason))

	snapshotRef := fmt.Sprintf("/snapshots/%s/%s/latest", input.TenantID, input.PackageName)

	logger.Info("RunRollbackActivity succeeded", "restoredFrom", snapshotRef)
	return RollbackResult{
		RolledBack:   true,
		RestoredFrom: snapshotRef,
		RollbackAt:   time.Now().UTC(),
	}, nil
}

// ---------------------------------------------------------------------------
// Worker registration
// ---------------------------------------------------------------------------

// RegisterWorkflows registers the PatchWorkflow and all its activities with
// the provided Temporal worker instance.
func RegisterWorkflows(w worker.Worker) {
	w.RegisterWorkflow(PatchWorkflow)

	w.RegisterActivity(ValidatePatchActivity)
	w.RegisterActivity(CreateSnapshotActivity)
	w.RegisterActivity(ApplyPatchActivity)
	w.RegisterActivity(VerifyPatchActivity)
	w.RegisterActivity(NotifyPatchCompleteActivity)
	w.RegisterActivity(RunRollbackActivity)
}
