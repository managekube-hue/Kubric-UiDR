package schema_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

func TestValidateTenantID_valid(t *testing.T) {
	valid := []string{
		"acme-corp",
		"tenant1",
		"my-org-123",
		"ab",
		"a0",
		"z9",
	}
	for _, id := range valid {
		if err := schema.ValidateTenantID(id); err != nil {
			t.Errorf("ValidateTenantID(%q) unexpected error: %v", id, err)
		}
	}
}

func TestValidateTenantID_empty(t *testing.T) {
	err := schema.ValidateTenantID("")
	if !errors.Is(err, schema.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID, got %v", err)
	}
}

func TestValidateTenantID_invalid(t *testing.T) {
	invalid := []string{
		"UPPERCASE",
		"has space",
		"has_underscore",
		"-starts-with-dash",
		"ends-with-dash-",
		"a",
		strings.Repeat("a", 64),
		"has.dot",
	}
	for _, id := range invalid {
		err := schema.ValidateTenantID(id)
		if err == nil {
			t.Errorf("ValidateTenantID(%q) expected error, got nil", id)
		}
	}
}

func TestNATSSubject(t *testing.T) {
	got := schema.NATSSubject("acme-corp", "endpoint", "process")
	want := "kubric.acme-corp.endpoint.process.v1"
	if got != want {
		t.Errorf("NATSSubject = %q, want %q", got, want)
	}
}

func TestNATSSubscribePattern(t *testing.T) {
	got := schema.NATSSubscribePattern("acme-corp")
	want := "kubric.acme-corp.>"
	if got != want {
		t.Errorf("NATSSubscribePattern = %q, want %q", got, want)
	}
}

func TestClickHousePartitionKey(t *testing.T) {
	got := schema.ClickHousePartitionKey()
	if got == "" {
		t.Error("ClickHousePartitionKey returned empty string")
	}
	if !strings.Contains(got, "tenant_id") {
		t.Errorf("ClickHousePartitionKey %q must contain tenant_id", got)
	}
}
