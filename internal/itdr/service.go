package itdr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/managekube-hue/Kubric-UiDR/internal/bloodhound"
	"github.com/managekube-hue/Kubric-UiDR/internal/cortex"
)

// Config controls ITDR runtime integration points and asset roots.
type Config struct {
	SigmaSecurityDir       string
	SigmaPrivEscDir        string
	WazuhRulesDir          string
	MispTaxonomiesDir      string
	BloodHoundCypherDir    string
	IdentityRespondersDir  string
	OTXBaseURL             string
	OTXAPIKey              string
	ResponderCommand       string
}

// Service provides ITDR operations over vendored assets and external integrations.
type Service struct {
	cfg    Config
	bh     *bloodhound.Client
	cortex *cortex.Client
	hc     *http.Client
}

type RuleAsset struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

type AssetInventory struct {
	SigmaADRules      []RuleAsset `json:"sigma_ad_rules"`
	SigmaPrivEscRules []RuleAsset `json:"sigma_privesc_rules"`
	WazuhADRules      []RuleAsset `json:"wazuh_ad_rules"`
	CypherQueries     []RuleAsset `json:"cypher_queries"`
}

type MispTaxonomy struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	Path        string `json:"path"`
}

type ResponderResult struct {
	Name      string                 `json:"name"`
	ExitCode  int                    `json:"exit_code"`
	Stdout    string                 `json:"stdout"`
	Stderr    string                 `json:"stderr"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

func New(cfg Config, bh *bloodhound.Client, cx *cortex.Client) *Service {
	if strings.TrimSpace(cfg.SigmaSecurityDir) == "" {
		cfg.SigmaSecurityDir = filepath.FromSlash("vendor/sigma/rules/windows/builtin/security")
	}
	if strings.TrimSpace(cfg.SigmaPrivEscDir) == "" {
		cfg.SigmaPrivEscDir = filepath.FromSlash("vendor/sigma/rules/windows/builtin/security/privesc")
	}
	if strings.TrimSpace(cfg.WazuhRulesDir) == "" {
		cfg.WazuhRulesDir = filepath.FromSlash("vendor/wazuh-rules")
	}
	if strings.TrimSpace(cfg.MispTaxonomiesDir) == "" {
		cfg.MispTaxonomiesDir = filepath.FromSlash("vendor/misp/taxonomies")
	}
	if strings.TrimSpace(cfg.BloodHoundCypherDir) == "" {
		cfg.BloodHoundCypherDir = filepath.FromSlash("vendor/bloodhound/cypher")
	}
	if strings.TrimSpace(cfg.IdentityRespondersDir) == "" {
		cfg.IdentityRespondersDir = filepath.FromSlash("vendor/cortex/responders/identity")
	}
	if strings.TrimSpace(cfg.OTXBaseURL) == "" {
		cfg.OTXBaseURL = "https://otx.alienvault.com/api/v1/indicators"
	}
	if strings.TrimSpace(cfg.ResponderCommand) == "" {
		cfg.ResponderCommand = "python3"
	}

	return &Service{
		cfg:    cfg,
		bh:     bh,
		cortex: cx,
		hc:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) Health(ctx context.Context) error {
	inventory, err := s.ListAssets()
	if err != nil {
		return err
	}
	totalLocal := len(inventory.SigmaADRules) + len(inventory.SigmaPrivEscRules) + len(inventory.WazuhADRules)
	if totalLocal == 0 && s.bh == nil && s.cortex == nil && strings.TrimSpace(s.cfg.OTXAPIKey) == "" {
		return fmt.Errorf("itdr: no local assets or upstream integrations configured")
	}
	_ = ctx
	return nil
}

func (s *Service) ListAssets() (*AssetInventory, error) {
	inv := &AssetInventory{}

	sigma, _ := globAssets("sigma_ad", s.cfg.SigmaSecurityDir, "*.yml")
	inv.SigmaADRules = sigma

	privEsc, _ := globAssets("sigma_privesc", s.cfg.SigmaPrivEscDir, "*.yml")
	inv.SigmaPrivEscRules = privEsc

	wazuh, _ := findWazuhADAssets(s.cfg.WazuhRulesDir)
	inv.WazuhADRules = wazuh

	cypher, _ := globAssets("bloodhound_cypher", s.cfg.BloodHoundCypherDir, "*.cypher")
	inv.CypherQueries = cypher

	return inv, nil
}

func (s *Service) ListMispTaxonomies() ([]MispTaxonomy, error) {
	if _, err := os.Stat(s.cfg.MispTaxonomiesDir); err != nil {
		if os.IsNotExist(err) {
			return []MispTaxonomy{}, nil
		}
		return nil, fmt.Errorf("misp taxonomy root: %w", err)
	}

	var out []MispTaxonomy
	err := filepath.WalkDir(s.cfg.MispTaxonomiesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".json" {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var parsed struct {
			Name        string `json:"name"`
			Namespace   string `json:"namespace"`
			Description string `json:"description"`
			Version     int    `json:"version"`
		}
		if err := json.Unmarshal(payload, &parsed); err != nil {
			return nil
		}
		if parsed.Name == "" && parsed.Namespace == "" {
			return nil
		}
		out = append(out, MispTaxonomy{
			Name:        parsed.Name,
			Namespace:   parsed.Namespace,
			Description: parsed.Description,
			Version:     parsed.Version,
			Path:        filepath.ToSlash(path),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) RunCypherQueryFile(ctx context.Context, file string, params map[string]interface{}) (*bloodhound.CypherResult, error) {
	if s.bh == nil {
		return nil, fmt.Errorf("bloodhound integration not configured")
	}
	clean := filepath.Clean(file)
	if clean == "." || strings.Contains(clean, "..") {
		return nil, fmt.Errorf("invalid cypher file path")
	}
	fullPath := filepath.Join(s.cfg.BloodHoundCypherDir, clean)
	queryBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read cypher file: %w", err)
	}
	query := strings.TrimSpace(string(queryBytes))
	if query == "" {
		return nil, fmt.Errorf("cypher query is empty")
	}
	return s.bh.RunCypher(ctx, query, params)
}

func (s *Service) LookupOTXIndicator(ctx context.Context, indicatorType, indicator string) (map[string]interface{}, error) {
	if strings.TrimSpace(s.cfg.OTXAPIKey) == "" {
		return nil, fmt.Errorf("otx api key not configured")
	}
	if strings.TrimSpace(indicatorType) == "" || strings.TrimSpace(indicator) == "" {
		return nil, fmt.Errorf("indicator type and value are required")
	}

	endpoint := strings.TrimRight(s.cfg.OTXBaseURL, "/") + "/" +
		url.PathEscape(indicatorType) + "/" + url.PathEscape(indicator) + "/general"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("otx request: %w", err)
	}
	req.Header.Set("X-OTX-API-KEY", s.cfg.OTXAPIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("otx http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("otx read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("otx api returned %d: %s", resp.StatusCode, string(body))
	}

	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("otx decode response: %w", err)
	}
	return out, nil
}

func (s *Service) RunIdentityResponderScript(ctx context.Context, name string, payload map[string]interface{}) (*ResponderResult, error) {
	script := strings.TrimSpace(name)
	if script == "" {
		return nil, fmt.Errorf("responder name is required")
	}
	if !strings.HasSuffix(strings.ToLower(script), ".py") {
		script += ".py"
	}
	clean := filepath.Clean(script)
	if strings.Contains(clean, "..") {
		return nil, fmt.Errorf("invalid responder path")
	}
	scriptPath := filepath.Join(s.cfg.IdentityRespondersDir, clean)
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("identity responder not found: %w", err)
	}

	input, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode responder payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, s.cfg.ResponderCommand, scriptPath)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	res := &ResponderResult{
		Name:     clean,
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
	}
	if res.Stdout != "" {
		var parsed map[string]interface{}
		if json.Unmarshal([]byte(res.Stdout), &parsed) == nil {
			res.Payload = parsed
		}
	}
	if err != nil {
		return res, fmt.Errorf("identity responder execution failed: %w", err)
	}
	return res, nil
}

func (s *Service) RunCortexIdentityResponder(ctx context.Context, responderID, objectType, objectID string, params map[string]interface{}) (*cortex.ResponderAction, error) {
	if s.cortex == nil {
		return nil, fmt.Errorf("cortex integration not configured")
	}
	if responderID == "" || objectType == "" || objectID == "" {
		return nil, fmt.Errorf("responder_id, object_type, and object_id are required")
	}
	return s.cortex.RunResponder(ctx, responderID, objectType, objectID, params)
}

func globAssets(source, root, pattern string) ([]RuleAsset, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return []RuleAsset{}, nil
		}
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return nil, err
	}
	assets := make([]RuleAsset, 0, len(matches))
	for _, m := range matches {
		assets = append(assets, RuleAsset{Source: source, Path: filepath.ToSlash(m), Name: filepath.Base(m)})
	}
	return assets, nil
}

func findWazuhADAssets(root string) ([]RuleAsset, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return []RuleAsset{}, nil
		}
		return nil, err
	}
	assets := make([]RuleAsset, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".xml" {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
		matched, _ := filepath.Match("0200-*_ad_*.xml", base)
		if !matched {
			return nil
		}
		assets = append(assets, RuleAsset{
			Source: "wazuh_ad",
			Path:   filepath.ToSlash(path),
			Name:   filepath.Base(path),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assets, nil
}
