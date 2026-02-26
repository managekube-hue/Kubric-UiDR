// Package soc provides SOC-level detection orchestration.
// K-SOC-DET-001 — Sigma rule compiler: loads, compiles, and manages Sigma YAML rules
// for the SOC detection pipeline, bridging to CoreSec agent sigma evaluator.
package soc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// SigmaRule is the structured representation of a Sigma detection rule.
// Fields map directly to the Sigma specification v1.
type SigmaRule struct {
	// Core identity
	ID          string   `yaml:"id"          json:"id"`
	Title       string   `yaml:"title"       json:"title"`
	Status      string   `yaml:"status"      json:"status"` // stable, test, experimental, deprecated, unsupported
	Level       string   `yaml:"level"       json:"level"`  // informational, low, medium, high, critical
	Author      string   `yaml:"author"      json:"author"`
	Description string   `yaml:"description" json:"description"`
	References  []string `yaml:"references"  json:"references,omitempty"`

	// MITRE ATT&CK and Sigma tags
	Tags []string `yaml:"tags" json:"tags,omitempty"`

	// Sigma detection block — raw map to preserve complex condition trees.
	Detection map[string]interface{} `yaml:"detection" json:"detection,omitempty"`

	// Logsource identifies the product/service/category this rule targets.
	Logsource map[string]string `yaml:"logsource" json:"logsource,omitempty"`

	// Optional fields
	Falsepositives []string `yaml:"falsepositives" json:"falsepositives,omitempty"`
	Date           string   `yaml:"date"           json:"date,omitempty"`
	Modified       string   `yaml:"modified"       json:"modified,omitempty"`
	License        string   `yaml:"license"        json:"license,omitempty"`

	// Internal bookkeeping — not serialised from YAML.
	FilePath string `yaml:"-" json:"file_path,omitempty"`
}

// sigmaRuleMessage is the NATS wire format for bulk rule publication.
type sigmaRuleMessage struct {
	TenantID    string      `json:"tenant_id"`
	PublishedAt time.Time   `json:"published_at"`
	Count       int         `json:"count"`
	Rules       []SigmaRule `json:"rules"`
}

// ---------------------------------------------------------------------------
// SigmaCompiler
// ---------------------------------------------------------------------------

// SigmaCompiler loads, indexes, and manages Sigma YAML rules from a directory
// tree.  It is safe for concurrent read access after the initial load.
type SigmaCompiler struct {
	mu       sync.RWMutex
	rules    map[string]*SigmaRule // keyed by rule ID
	loadedAt time.Time
	watchDir string
}

// NewSigmaCompiler creates a new compiler and immediately walks rulesDir,
// loading every .yml / .yaml file found.  If rulesDir is empty the compiler
// is created without any rules and rules can be loaded later.
func NewSigmaCompiler(rulesDir string) (*SigmaCompiler, error) {
	sc := &SigmaCompiler{
		rules:    make(map[string]*SigmaRule),
		watchDir: rulesDir,
	}
	if rulesDir == "" {
		return sc, nil
	}
	if _, err := os.Stat(rulesDir); err != nil {
		return nil, fmt.Errorf("sigma: rules dir %q: %w", rulesDir, err)
	}
	if _, errs := sc.CompileAll(); len(errs) > 0 {
		// Log but do not fail — partial load is valid.
		_ = errs
	}
	return sc, nil
}

// LoadRule parses a single Sigma YAML file and returns the populated rule.
// It does NOT add the rule to the compiler's internal index; use compileOne
// for that.
func LoadRule(path string) (*SigmaRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sigma: read %q: %w", path, err)
	}
	var rule SigmaRule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("sigma: parse %q: %w", path, err)
	}
	rule.FilePath = path
	if rule.ID == "" {
		// Fall back to the filename without extension as a synthetic ID.
		base := filepath.Base(path)
		rule.ID = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return &rule, nil
}

// CompileAll walks the configured rulesDir, loads every .yml / .yaml file, and
// updates the internal rule index.  It returns the count of successfully loaded
// rules and any per-file parse errors encountered.  Existence of errors does
// not prevent the successfully parsed rules from being indexed.
func (sc *SigmaCompiler) CompileAll() (int, []error) {
	if sc.watchDir == "" {
		return 0, nil
	}

	var (
		errs   []error
		loaded int
	)

	err := filepath.WalkDir(sc.watchDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, fmt.Errorf("sigma: walk %q: %w", path, walkErr))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		rule, err := LoadRule(path)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		sc.mu.Lock()
		sc.rules[rule.ID] = rule
		sc.mu.Unlock()
		loaded++
		return nil
	})
	if err != nil {
		errs = append(errs, fmt.Errorf("sigma: walk dir: %w", err))
	}

	sc.mu.Lock()
	sc.loadedAt = time.Now()
	sc.mu.Unlock()

	return loaded, errs
}

// FindByTag returns all loaded rules that carry the given tag string.
// The comparison is case-insensitive.  Tag format follows Sigma convention,
// e.g. "attack.t1059", "attack.execution", "attack.t1055.001".
func (sc *SigmaCompiler) FindByTag(tag string) []SigmaRule {
	tag = strings.ToLower(strings.TrimSpace(tag))
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var out []SigmaRule
	for _, r := range sc.rules {
		for _, t := range r.Tags {
			if strings.ToLower(t) == tag {
				out = append(out, *r)
				break
			}
		}
	}
	return out
}

// FindByLevel returns all loaded rules at the given severity level.
// Valid values: informational, low, medium, high, critical.
// The comparison is case-insensitive.
func (sc *SigmaCompiler) FindByLevel(level string) []SigmaRule {
	level = strings.ToLower(strings.TrimSpace(level))
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var out []SigmaRule
	for _, r := range sc.rules {
		if strings.ToLower(r.Level) == level {
			out = append(out, *r)
		}
	}
	return out
}

// Get returns a single rule by its ID, or false if not found.
func (sc *SigmaCompiler) Get(id string) (*SigmaRule, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	r, ok := sc.rules[id]
	if !ok {
		return nil, false
	}
	cp := *r
	return &cp, true
}

// All returns a snapshot of all currently loaded rules.
func (sc *SigmaCompiler) All() []SigmaRule {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	out := make([]SigmaRule, 0, len(sc.rules))
	for _, r := range sc.rules {
		out = append(out, *r)
	}
	return out
}

// RuleStats returns a map of severity level → rule count for all loaded rules.
func (sc *SigmaCompiler) RuleStats() map[string]int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := map[string]int{
		"informational": 0,
		"low":           0,
		"medium":        0,
		"high":          0,
		"critical":      0,
		"unknown":       0,
	}
	for _, r := range sc.rules {
		lvl := strings.ToLower(r.Level)
		if _, known := stats[lvl]; known {
			stats[lvl]++
		} else {
			stats["unknown"]++
		}
	}
	return stats
}

// ExportToNATS publishes all loaded rules as a single JSON message to the
// subject:  kubric.{tenantID}.detection.sigma_rules.v1
//
// The message is an encoded sigmaRuleMessage with the full rule list.
func (sc *SigmaCompiler) ExportToNATS(nc *nats.Conn, tenantID string) error {
	if nc == nil {
		return fmt.Errorf("sigma: nats connection is nil")
	}
	rules := sc.All()
	msg := sigmaRuleMessage{
		TenantID:    tenantID,
		PublishedAt: time.Now().UTC(),
		Count:       len(rules),
		Rules:       rules,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sigma: marshal rules for NATS: %w", err)
	}
	subject := fmt.Sprintf("kubric.%s.detection.sigma_rules.v1", tenantID)
	if err := nc.Publish(subject, payload); err != nil {
		return fmt.Errorf("sigma: publish to %q: %w", subject, err)
	}
	return nc.Flush()
}

// WatchForChanges polls the rules directory every 30 seconds and fires
// callback with the sets of added and removed rule IDs whenever the on-disk
// state diverges from the in-memory index.  It runs until ctx is cancelled.
//
// Note: polling is used intentionally to avoid pulling the fsnotify dependency
// into the module for a single consumer.  Interval can be tuned if needed.
func (sc *SigmaCompiler) WatchForChanges(ctx context.Context, callback func(added, removed []string)) {
	if sc.watchDir == "" || callback == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.mu.RLock()
			before := make(map[string]struct{}, len(sc.rules))
			for id := range sc.rules {
				before[id] = struct{}{}
			}
			sc.mu.RUnlock()

			// Re-compile from disk.
			fresh := &SigmaCompiler{
				rules:    make(map[string]*SigmaRule),
				watchDir: sc.watchDir,
			}
			fresh.CompileAll() //nolint:errcheck

			fresh.mu.RLock()
			after := make(map[string]struct{}, len(fresh.rules))
			for id := range fresh.rules {
				after[id] = struct{}{}
			}
			fresh.mu.RUnlock()

			var added, removed []string
			for id := range after {
				if _, exists := before[id]; !exists {
					added = append(added, id)
				}
			}
			for id := range before {
				if _, exists := after[id]; !exists {
					removed = append(removed, id)
				}
			}

			if len(added) > 0 || len(removed) > 0 {
				// Atomically swap in the new rule set.
				sc.mu.Lock()
				sc.rules = fresh.rules
				sc.loadedAt = time.Now()
				sc.mu.Unlock()
				callback(added, removed)
			}
		}
	}
}

// LoadedAt returns the time of the last successful CompileAll run.
func (sc *SigmaCompiler) LoadedAt() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.loadedAt
}

// Count returns the total number of currently loaded rules.
func (sc *SigmaCompiler) Count() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.rules)
}
