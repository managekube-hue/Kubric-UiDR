// Package noc provides NOC operations tooling.
// K-NOC-PT-001 — Patch Delta Generator: compare installed vs required package versions.
package noc

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InstalledPkg represents a package installed on an asset.
type InstalledPkg struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Architecture string    `json:"architecture"`
	InstallDate  time.Time `json:"install_date"`
}

// RequiredPkg represents a package version prescribed by the patch catalog.
type RequiredPkg struct {
	Name          string   `json:"name"`
	TargetVersion string   `json:"target_version"`
	Severity      string   `json:"severity"`
	CVEs          []string `json:"cves"`
}

// PatchDelta describes the gap between the installed version and the required version.
type PatchDelta struct {
	PackageName      string `json:"package_name"`
	InstalledVersion string `json:"installed_version"`
	TargetVersion    string `json:"target_version"`
	Severity         string `json:"severity"`
	Upgrade          bool   `json:"upgrade"`
}

// PatchScore summarises a set of deltas by severity.
type PatchScore struct {
	Critical  int     `json:"critical"`
	High      int     `json:"high"`
	Medium    int     `json:"medium"`
	Low       int     `json:"low"`
	RiskScore float64 `json:"risk_score"`
}

// DeltaGenerator produces patch delta objects for assets.
type DeltaGenerator struct{}

// GetInstalledPackages queries the database for packages installed on an asset.
func (g *DeltaGenerator) GetInstalledPackages(ctx context.Context, assetID string, dbPool *pgxpool.Pool) ([]InstalledPkg, error) {
	rows, err := dbPool.Query(ctx,
		`SELECT name, version, COALESCE(architecture,''), COALESCE(install_date, NOW())
		 FROM asset_packages WHERE asset_id=$1`, assetID)
	if err != nil {
		return nil, fmt.Errorf("query installed packages for %s: %w", assetID, err)
	}
	defer rows.Close()

	var pkgs []InstalledPkg
	for rows.Next() {
		var p InstalledPkg
		if scanErr := rows.Scan(&p.Name, &p.Version, &p.Architecture, &p.InstallDate); scanErr != nil {
			return nil, fmt.Errorf("scan installed package: %w", scanErr)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, rows.Err()
}

// GetRequiredVersions queries the patch catalog for the given package names.
func (g *DeltaGenerator) GetRequiredVersions(ctx context.Context, pkgNames []string, dbPool *pgxpool.Pool) ([]RequiredPkg, error) {
	rows, err := dbPool.Query(ctx,
		`SELECT name, target_version, severity, COALESCE(cves, ARRAY[]::text[])
		 FROM patch_catalog WHERE name=ANY($1)`, pkgNames)
	if err != nil {
		return nil, fmt.Errorf("query patch catalog: %w", err)
	}
	defer rows.Close()

	var required []RequiredPkg
	for rows.Next() {
		var r RequiredPkg
		if scanErr := rows.Scan(&r.Name, &r.TargetVersion, &r.Severity, &r.CVEs); scanErr != nil {
			return nil, fmt.Errorf("scan required package: %w", scanErr)
		}
		required = append(required, r)
	}
	return required, rows.Err()
}

// GenerateDelta computes the set of packages that need upgrading.
func (g *DeltaGenerator) GenerateDelta(installed []InstalledPkg, required []RequiredPkg) []PatchDelta {
	reqMap := make(map[string]RequiredPkg, len(required))
	for _, r := range required {
		reqMap[r.Name] = r
	}

	var deltas []PatchDelta
	for _, inst := range installed {
		req, ok := reqMap[inst.Name]
		if !ok {
			continue
		}
		if inst.Version == req.TargetVersion {
			continue
		}
		deltas = append(deltas, PatchDelta{
			PackageName:      inst.Name,
			InstalledVersion: inst.Version,
			TargetVersion:    req.TargetVersion,
			Severity:         req.Severity,
			Upgrade:          true,
		})
	}
	return deltas
}

// ScoreDelta counts deltas by severity and computes a weighted risk score.
func (g *DeltaGenerator) ScoreDelta(deltas []PatchDelta) PatchScore {
	score := PatchScore{}
	for _, d := range deltas {
		switch d.Severity {
		case "critical":
			score.Critical++
		case "high":
			score.High++
		case "medium":
			score.Medium++
		default:
			score.Low++
		}
	}
	// Weighted: critical=10, high=5, medium=2, low=1; normalised to 0-100.
	raw := float64(score.Critical*10 + score.High*5 + score.Medium*2 + score.Low)
	total := float64(len(deltas))
	if total > 0 {
		score.RiskScore = (raw / (total * 10)) * 100
		if score.RiskScore > 100 {
			score.RiskScore = 100
		}
	}
	return score
}

// SaveDelta upserts patch delta records into the patch_deltas table using pgx batching.
func (g *DeltaGenerator) SaveDelta(ctx context.Context, assetID string, deltas []PatchDelta, dbPool *pgxpool.Pool) error {
	if len(deltas) == 0 {
		return nil
	}

	now := time.Now().UTC()
	batch := &pgx.Batch{}
	for _, d := range deltas {
		batch.Queue(
			`INSERT INTO patch_deltas
			    (asset_id, package_name, installed_version, target_version, severity, upgrade, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)
			 ON CONFLICT (asset_id, package_name) DO UPDATE
			     SET installed_version=EXCLUDED.installed_version,
			         target_version=EXCLUDED.target_version,
			         severity=EXCLUDED.severity,
			         updated_at=EXCLUDED.updated_at`,
			assetID, d.PackageName, d.InstalledVersion, d.TargetVersion, d.Severity, d.Upgrade, now,
		)
	}

	conn, err := dbPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire db conn: %w", err)
	}
	defer conn.Release()

	br := conn.SendBatch(ctx, batch)
	defer br.Close()

	for range deltas {
		if _, execErr := br.Exec(); execErr != nil {
			return fmt.Errorf("upsert patch delta: %w", execErr)
		}
	}
	return nil
}
