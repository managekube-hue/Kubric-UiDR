package billing

import (
	"testing"
	"time"
)

func TestBillingMerkleRoot(t *testing.T) {
	ts := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	events := []UsageEvent{
		{TenantID: "tenant-1", EventType: "event.processed", Quantity: 100, Timestamp: ts},
		{TenantID: "tenant-1", EventType: "agent.heartbeat", Quantity: 50, Timestamp: ts.Add(time.Hour)},
		{TenantID: "tenant-1", EventType: "scan.completed", Quantity: 10, Timestamp: ts.Add(2 * time.Hour)},
	}

	// First computation
	root1 := MerkleRoot(events)
	if root1 == "" {
		t.Fatal("MerkleRoot returned empty string")
	}

	// Second computation (same data) must be deterministic
	root2 := MerkleRoot(events)
	if root1 != root2 {
		t.Fatalf("MerkleRoot not deterministic: %s != %s", root1, root2)
	}

	// Reversed order must produce same root (sorted internally)
	reversed := []UsageEvent{events[2], events[0], events[1]}
	root3 := MerkleRoot(reversed)
	if root1 != root3 {
		t.Fatalf("MerkleRoot order-dependent: %s != %s", root1, root3)
	}

	// Different data must produce different root
	different := []UsageEvent{
		{TenantID: "tenant-2", EventType: "event.processed", Quantity: 200, Timestamp: ts},
	}
	root4 := MerkleRoot(different)
	if root1 == root4 {
		t.Fatal("MerkleRoot collision: different data produced same root")
	}

	t.Logf("Merkle root (3 events): %s", root1)
}

func TestMerkleRootEmpty(t *testing.T) {
	root := MerkleRoot(nil)
	if root != "" {
		t.Fatalf("expected empty root for nil events, got: %s", root)
	}
}
