package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/managekube-hue/Kubric-UiDR/internal/intel/canonical"
	"gopkg.in/yaml.v3"
)

type xmlDecoder struct {
	XMLName     xml.Name `xml:"decoder"`
	Name        string   `xml:"name,attr"`
	ProgramName string   `xml:"program_name"`
	Regex       []string `xml:"regex"`
	Order       string   `xml:"order"`
}

type xmlRule struct {
	XMLName     xml.Name `xml:"rule"`
	ID          string   `xml:"id,attr"`
	Level       string   `xml:"level,attr"`
	Description string   `xml:"description"`
}

type sigmaRule struct {
	Title      string                 `yaml:"title"`
	ID         string                 `yaml:"id"`
	Status     string                 `yaml:"status"`
	References []string               `yaml:"references"`
	Tags       []string               `yaml:"tags"`
	Detection  map[string]interface{} `yaml:"detection"`
}

func main() {
	var upstreamRoot string
	var outFile string

	flag.StringVar(&upstreamRoot, "upstream-root", "third_party/intelligence/upstream", "upstream intelligence root")
	flag.StringVar(&outFile, "out", "third_party/intelligence/normalized/canonical.ndjson", "normalized output file")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	handle, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer handle.Close()

	writer := bufio.NewWriter(handle)
	defer writer.Flush()

	count, err := normalizeTree(upstreamRoot, writer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "normalize failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("normalized records: %d -> %s\n", count, outFile)
}

func normalizeTree(root string, writer *bufio.Writer) (int, error) {
	records := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		extension := strings.ToLower(filepath.Ext(path))
		switch extension {
		case ".xml":
			emitted, emitErr := normalizeXML(path, writer)
			if emitErr != nil {
				return emitErr
			}
			records += emitted
		case ".yaml", ".yml":
			emitted, emitErr := normalizeYAML(path, writer)
			if emitErr != nil {
				return emitErr
			}
			records += emitted
		}

		return nil
	})

	return records, err
}

func normalizeXML(path string, writer *bufio.Writer) (int, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read xml %s: %w", path, err)
	}

	var marker struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(payload, &marker); err != nil {
		return 0, nil
	}

	reference := provenanceFor(path)
	switch marker.XMLName.Local {
	case "decoder":
		var decoder xmlDecoder
		if err := xml.Unmarshal(payload, &decoder); err != nil {
			return 0, fmt.Errorf("unmarshal decoder %s: %w", path, err)
		}
		record := canonical.DecoderSpec{
			ID:          strings.TrimSpace(decoder.Name),
			Name:        strings.TrimSpace(decoder.Name),
			ProgramName: strings.TrimSpace(decoder.ProgramName),
			Regex:       decoder.Regex,
			Order:       splitOrder(decoder.Order),
			Provenance:  reference,
		}
		return 1, writeRecord(writer, record)
	case "rule":
		var rule xmlRule
		if err := xml.Unmarshal(payload, &rule); err != nil {
			return 0, fmt.Errorf("unmarshal rule %s: %w", path, err)
		}
		record := canonical.DetectionSpec{
			ID:          strings.TrimSpace(rule.ID),
			Name:        fmt.Sprintf("wazuh_rule_%s", strings.TrimSpace(rule.ID)),
			Description: strings.TrimSpace(rule.Description),
			Level:       strings.TrimSpace(rule.Level),
			Metadata: map[string]string{
				"source_format": "xml",
			},
			Provenance: reference,
		}
		return 1, writeRecord(writer, record)
	default:
		return 0, nil
	}
}

func normalizeYAML(path string, writer *bufio.Writer) (int, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read yaml %s: %w", path, err)
	}

	if strings.Contains(path, "/sigma/") {
		var rule sigmaRule
		if err := yaml.Unmarshal(payload, &rule); err != nil {
			return 0, nil
		}
		if strings.TrimSpace(rule.Title) == "" {
			return 0, nil
		}
		record := canonical.DetectionSpec{
			ID:          strings.TrimSpace(rule.ID),
			Name:        strings.TrimSpace(rule.Title),
			Description: strings.TrimSpace(rule.Status),
			Tags:        rule.Tags,
			References:  rule.References,
			Condition:   "sigma_detection",
			Metadata: map[string]string{
				"source_format": "yaml",
			},
			Provenance: provenanceFor(path),
		}
		return 1, writeRecord(writer, record)
	}

	return 0, nil
}

func writeRecord(writer *bufio.Writer, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func splitOrder(order string) []string {
	parts := strings.Split(order, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func provenanceFor(path string) canonical.Provenance {
	repo := "unknown"
	reference := "unknown"
	license := "unknown"
	cleanPath := filepath.ToSlash(path)

	switch {
	case strings.Contains(cleanPath, "/wazuh/"):
		repo = "https://github.com/wazuh/wazuh"
		reference = "master"
		license = "GPL-2.0"
	case strings.Contains(cleanPath, "/sigma/"):
		repo = "https://github.com/SigmaHQ/sigma"
		reference = "master"
		license = "Detection-content license (verify upstream)"
	case strings.Contains(cleanPath, "/suricata/"):
		repo = "https://github.com/OISF/suricata"
		reference = "master"
		license = "GPL-2.0"
	case strings.Contains(cleanPath, "/zeek/"):
		repo = "https://github.com/zeek/zeek"
		reference = "master"
		license = "BSD-3-Clause (verify upstream file)"
	}

	return canonical.Provenance{
		SourceSystem: sourceName(cleanPath),
		UpstreamRepo: repo,
		UpstreamRef:  reference,
		UpstreamPath: cleanPath,
		License:      license,
		IngestedAt:   time.Now().UTC(),
	}
}

func sourceName(path string) string {
	switch {
	case strings.Contains(path, "/wazuh/"):
		return "wazuh"
	case strings.Contains(path, "/sigma/"):
		return "sigma"
	case strings.Contains(path, "/suricata/"):
		return "suricata"
	case strings.Contains(path, "/zeek/"):
		return "zeek"
	default:
		return "unknown"
	}
}
