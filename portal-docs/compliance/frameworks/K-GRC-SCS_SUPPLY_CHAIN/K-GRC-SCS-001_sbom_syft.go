package grc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SBOMCompEntry is a single software component inside an SBOM.
// NOTE: named SBOMCompEntry (not SBOMComponent) to avoid collision with
// the soc package in K-SOC-VULN-006 which owns the SBOMComponent type.
type SBOMCompEntry struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Type     string   `json:"type"`
	Language string   `json:"language"`
	PURL     string   `json:"purl"`
	Licenses []string `json:"licenses"`
	CPEs     []string `json:"cpes"`
}

// SBOMStoreEntry represents a stored SBOM record, including parsed component list.
// NOTE: named SBOMStoreEntry (not SBOMEntry) to avoid collision with other packages.
type SBOMStoreEntry struct {
	Serial      string          `json:"serialNumber"`
	TargetName  string          `json:"targetName"`
	TargetType  string          `json:"targetType"`
	Components  []SBOMCompEntry `json:"components"`
	GeneratedAt time.Time       `json:"generatedAt"`
	Source      string          `json:"source"` // "syft"
}

// SBOMDiffResult holds the delta between two SBOMs.
// NOTE: named SBOMDiffResult (not SBOMDiff) to avoid collision with other packages.
type SBOMDiffResult struct {
	Added   []SBOMCompEntry `json:"added"`
	Removed []SBOMCompEntry `json:"removed"`
	Changed []string        `json:"changed"` // PURL strings whose version changed
}

// SBOMStore generates SBOMs via the syft CLI and persists them to PostgreSQL.
type SBOMStore struct {
	pgPool      *pgxpool.Pool
	StoragePath string
	syftBin     string
}

// NewSBOMStore constructs an SBOMStore from environment variables.
// DATABASE_URL (required), SBOM_STORAGE_PATH (default: /tmp/sboms).
func NewSBOMStore(ctx context.Context) (*SBOMStore, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("sbom_store: DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("sbom_store: pgxpool: %w", err)
	}

	storagePath := os.Getenv("SBOM_STORAGE_PATH")
	if storagePath == "" {
		storagePath = "/tmp/sboms"
	}
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		pool.Close()
		return nil, fmt.Errorf("sbom_store: mkdir %s: %w", storagePath, err)
	}

	syftBin := os.Getenv("SYFT_BIN")
	if syftBin == "" {
		syftBin = "syft"
	}

	return &SBOMStore{
		pgPool:      pool,
		StoragePath: storagePath,
		syftBin:     syftBin,
	}, nil
}

// cycloneDXDoc is a minimal CycloneDX JSON BOM shape used for parsing.
type cycloneDXDoc struct {
	SerialNumber string `json:"serialNumber"`
	Metadata     struct {
		Component struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"component"`
	} `json:"metadata"`
	Components []struct {
		Name     string   `json:"name"`
		Version  string   `json:"version"`
		Type     string   `json:"type"`
		Language string   `json:"language"`
		PURL     string   `json:"purl"`
		CPE      string   `json:"cpe"`
		Licenses []struct {
			License struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"license"`
		} `json:"licenses"`
		Hashes []struct {
			Alg     string `json:"alg"`
			Content string `json:"content"`
		} `json:"hashes"`
	} `json:"components"`
}

// parseCycloneDX turns raw CycloneDX JSON into an SBOMStoreEntry.
func parseCycloneDX(data []byte, target, targetType string) (*SBOMStoreEntry, error) {
	var doc cycloneDXDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse cyclonedx: %w", err)
	}

	targetName := target
	if doc.Metadata.Component.Name != "" {
		targetName = doc.Metadata.Component.Name
	}
	if targetType == "" {
		targetType = doc.Metadata.Component.Type
	}

	comps := make([]SBOMCompEntry, 0, len(doc.Components))
	for _, c := range doc.Components {
		var licenses []string
		for _, l := range c.Licenses {
			id := l.License.ID
			if id == "" {
				id = l.License.Name
			}
			if id != "" {
				licenses = append(licenses, id)
			}
		}
		var cpes []string
		if c.CPE != "" {
			cpes = append(cpes, c.CPE)
		}
		comps = append(comps, SBOMCompEntry{
			Name:     c.Name,
			Version:  c.Version,
			Type:     c.Type,
			Language: c.Language,
			PURL:     c.PURL,
			Licenses: licenses,
			CPEs:     cpes,
		})
	}

	return &SBOMStoreEntry{
		Serial:      doc.SerialNumber,
		TargetName:  targetName,
		TargetType:  targetType,
		Components:  comps,
		GeneratedAt: time.Now().UTC(),
		Source:      "syft",
	}, nil
}

// GenerateSBOM runs `syft <target> -o cyclonedx-json --quiet` and returns a parsed SBOMStoreEntry.
func (s *SBOMStore) GenerateSBOM(ctx context.Context, target, targetType string) (*SBOMStoreEntry, error) {
	outFile := filepath.Join(s.StoragePath, sbomFileName(target)+".cdx.json")

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, s.syftBin,
		target,
		"-o", "cyclonedx-json="+outFile,
		"--quiet",
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("syft %s: %w: %s", target, err, stderr.String())
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("read syft output %s: %w", outFile, err)
	}

	entry, err := parseCycloneDX(data, target, targetType)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// StoreSBOM writes the SBOM JSON to disk and upserts the record into sbom_inventory.
// Returns the PostgreSQL UUID of the newly created or updated row.
func (s *SBOMStore) StoreSBOM(ctx context.Context, sbom *SBOMStoreEntry, tenantID, assetID string) (string, error) {
	jsonData, err := json.Marshal(sbom)
	if err != nil {
		return "", fmt.Errorf("StoreSBOM: marshal: %w", err)
	}

	// Write SBOM JSON file.
	filePath := filepath.Join(s.StoragePath, sbom.Serial+".cdx.json")
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("StoreSBOM: write file: %w", err)
	}

	var id string
	err = s.pgPool.QueryRow(ctx, `
		INSERT INTO sbom_inventory
			(tenant_id, asset_id, target_name, target_type, serial_number,
			 component_count, sbom_path, sbom_json, source, generated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10)
		ON CONFLICT (tenant_id, asset_id, serial_number) DO UPDATE SET
			component_count = EXCLUDED.component_count,
			sbom_path       = EXCLUDED.sbom_path,
			sbom_json       = EXCLUDED.sbom_json,
			generated_at    = EXCLUDED.generated_at
		RETURNING id`,
		tenantID, assetID,
		sbom.TargetName, sbom.TargetType, sbom.Serial,
		len(sbom.Components), filePath, string(jsonData), sbom.Source, sbom.GeneratedAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("StoreSBOM: upsert: %w", err)
	}
	return id, nil
}

// GetSBOMHistory returns all SBOM records for a tenant/asset pair, newest first.
func (s *SBOMStore) GetSBOMHistory(ctx context.Context, tenantID, assetID string) ([]map[string]any, error) {
	rows, err := s.pgPool.Query(ctx, `
		SELECT id, tenant_id, asset_id, target_name, target_type,
		       serial_number, component_count, sbom_path, source, generated_at
		FROM sbom_inventory
		WHERE tenant_id = $1 AND asset_id = $2
		ORDER BY generated_at DESC`,
		tenantID, assetID)
	if err != nil {
		return nil, fmt.Errorf("GetSBOMHistory: %w", err)
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var (
			id, tenID, aID, targetName, targetType, serial, sbomPath, src string
			componentCount                                                  int
			generatedAt                                                     time.Time
		)
		if err := rows.Scan(&id, &tenID, &aID, &targetName, &targetType,
			&serial, &componentCount, &sbomPath, &src, &generatedAt); err != nil {
			return nil, fmt.Errorf("GetSBOMHistory scan: %w", err)
		}
		results = append(results, map[string]any{
			"id":              id,
			"tenant_id":       tenID,
			"asset_id":        aID,
			"target_name":     targetName,
			"target_type":     targetType,
			"serial_number":   serial,
			"component_count": componentCount,
			"sbom_path":       sbomPath,
			"source":          src,
			"generated_at":    generatedAt,
		})
	}
	return results, rows.Err()
}

// CompareSBOMs diffs two stored SBOMs by their PostgreSQL IDs.
func (s *SBOMStore) CompareSBOMs(ctx context.Context, id1, id2 string) (*SBOMDiffResult, error) {
	loadEntry := func(id string) (*SBOMStoreEntry, error) {
		var raw string
		if err := s.pgPool.QueryRow(ctx,
			`SELECT sbom_json FROM sbom_inventory WHERE id = $1`, id,
		).Scan(&raw); err != nil {
			return nil, fmt.Errorf("load sbom %s: %w", id, err)
		}
		var entry SBOMStoreEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			return nil, fmt.Errorf("parse sbom %s: %w", id, err)
		}
		return &entry, nil
	}

	e1, err := loadEntry(id1)
	if err != nil {
		return nil, err
	}
	e2, err := loadEntry(id2)
	if err != nil {
		return nil, err
	}

	// Build PURL-keyed maps (fall back to name@version).
	key := func(c SBOMCompEntry) string {
		if c.PURL != "" {
			return c.PURL
		}
		return c.Name + "@" + c.Version
	}

	map1 := make(map[string]SBOMCompEntry, len(e1.Components))
	for _, c := range e1.Components {
		map1[key(c)] = c
	}
	map2 := make(map[string]SBOMCompEntry, len(e2.Components))
	for _, c := range e2.Components {
		map2[key(c)] = c
	}

	diff := &SBOMDiffResult{}
	for k, c2 := range map2 {
		c1, exists := map1[k]
		if !exists {
			diff.Added = append(diff.Added, c2)
		} else if c1.Version != c2.Version {
			diff.Changed = append(diff.Changed, fmt.Sprintf("%s: %s -> %s", c2.Name, c1.Version, c2.Version))
		}
	}
	for k, c1 := range map1 {
		if _, exists := map2[k]; !exists {
			diff.Removed = append(diff.Removed, c1)
		}
	}
	return diff, nil
}

// Close releases the underlying PostgreSQL pool.
func (s *SBOMStore) Close() {
	s.pgPool.Close()
}

// sbomFileName replaces characters that are unsafe in filenames.
func sbomFileName(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' || c == ':' || c == '@' || c == ' ' {
			out[i] = '_'
		} else {
			out[i] = c
		}
	}
	return string(out)
}
