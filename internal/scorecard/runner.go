// Package scorecard wraps the OpenSSF Scorecard library to run supply-chain
// security assessments against Git repositories.  Results feed into KIC
// compliance posture as framework "OpenSSF-Scorecard".
//
// Depends on: github.com/ossf/scorecard/v4
package scorecard

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ossf/scorecard/v4/clients"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	sclog "github.com/ossf/scorecard/v4/log"
)

// CheckResult holds a single scorecard check result.
type CheckResult struct {
	Name    string  `json:"name"`
	Score   int     `json:"score"`   // 0-10, -1 = error
	Reason  string  `json:"reason"`
	Details []string `json:"details,omitempty"`
}

// RepoScore holds the aggregate scorecard result for a repository.
type RepoScore struct {
	Repo       string        `json:"repo"`
	CommitSHA  string        `json:"commit_sha"`
	ScoredAt   time.Time     `json:"scored_at"`
	Checks     []CheckResult `json:"checks"`
	Aggregate  float64       `json:"aggregate"` // weighted avg 0–10
}

// Runner executes OpenSSF Scorecard assessments.
type Runner struct {
	ghToken string
}

// NewRunner creates a scorecard Runner.  ghToken is a GitHub PAT for API access.
// Falls back to GITHUB_AUTH_TOKEN env var if token is empty.
func NewRunner(ghToken string) *Runner {
	if ghToken == "" {
		ghToken = os.Getenv("GITHUB_AUTH_TOKEN")
	}
	return &Runner{ghToken: ghToken}
}

// Score runs all default scorecard checks against the given repo
// (format: "github.com/owner/repo").
func (r *Runner) Score(ctx context.Context, repoURI string) (*RepoScore, error) {
	logger := sclog.NewLogger(sclog.WarnLevel)

	repo, repoClient, ossFuzzClient, ciiClient, vulnsClient, err := checker.GetClients(ctx, repoURI, "", logger)
	if err != nil {
		return nil, fmt.Errorf("scorecard init clients for %q: %w", repoURI, err)
	}
	defer repoClient.Close()
	if ossFuzzClient != nil {
		defer ossFuzzClient.Close()
	}

	if err := repoClient.InitRepo(repo, clients.HeadSHA, 0); err != nil {
		return nil, fmt.Errorf("scorecard init repo %q: %w", repoURI, err)
	}

	if r.ghToken != "" {
		if err := os.Setenv("GITHUB_AUTH_TOKEN", r.ghToken); err != nil {
			return nil, fmt.Errorf("scorecard set GITHUB_AUTH_TOKEN: %w", err)
		}
	}

	allChecks := checks.GetAll()
	out := &RepoScore{
		Repo:      repoURI,
		CommitSHA: clients.HeadSHA,
		ScoredAt:  time.Now().UTC(),
	}

	var total float64
	var count int
	for name, checkDef := range allChecks {
		req := &checker.CheckRequest{
			Ctx:                   ctx,
			RepoClient:            repoClient,
			CIIClient:             ciiClient,
			OssFuzzRepo:           ossFuzzClient,
			Repo:                  repo,
			VulnerabilitiesClient: vulnsClient,
			RawResults:            &checker.RawResults{},
		}
		runner := checker.NewRunner(name, repo.String(), req)
		cr := runner.Run(ctx, checkDef)

		check := CheckResult{
			Name:   cr.Name,
			Score:  cr.Score,
			Reason: cr.Reason,
		}
		if cr.Error != nil {
			check.Score = -1
			if check.Reason == "" {
				check.Reason = cr.Error.Error()
			}
		}
		for _, d := range cr.Details {
			if d.Msg.Text != "" {
				check.Details = append(check.Details, d.Msg.Text)
			}
		}

		out.Checks = append(out.Checks, check)
		if check.Score >= 0 {
			total += float64(check.Score)
			count++
		}
	}

	if count > 0 {
		out.Aggregate = total / float64(count)
	}

	return out, nil
}

// ScoreMultiple runs scorecard against multiple repos sequentially.
func (r *Runner) ScoreMultiple(ctx context.Context, repos []string) ([]RepoScore, error) {
	var results []RepoScore
	var errs []string
	for _, repo := range repos {
		score, err := r.Score(ctx, repo)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", repo, err))
			continue
		}
		results = append(results, *score)
	}
	if len(errs) > 0 {
		return results, fmt.Errorf("scorecard errors: %s", strings.Join(errs, "; "))
	}
	return results, nil
}
