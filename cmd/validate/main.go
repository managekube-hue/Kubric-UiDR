package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type summary struct {
	xmlFiles  int
	yamlFiles int
}

type wazuhDecoder struct {
	XMLName xml.Name `xml:"decoder"`
	Name    string   `xml:"name,attr"`
}

type wazuhRule struct {
	XMLName xml.Name `xml:"rule"`
	ID      string   `xml:"id,attr"`
}

type sigmaRule struct {
	Title     string      `yaml:"title"`
	Logsource interface{} `yaml:"logsource"`
	Detection interface{} `yaml:"detection"`
}

func main() {
	upstreamRoot := "third_party/intelligence/upstream"
	if value := strings.TrimSpace(os.Getenv("UPSTREAM_ROOT")); value != "" {
		upstreamRoot = value
	}

	stats, err := validateTree(upstreamRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("validation ok: xml=%d yaml=%d\n", stats.xmlFiles, stats.yamlFiles)
}

func validateTree(root string) (summary, error) {
	stats := summary{}
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
			if err := validateXML(path); err != nil {
				return err
			}
			stats.xmlFiles++
		case ".yaml", ".yml":
			if err := validateYAML(path); err != nil {
				return err
			}
			stats.yamlFiles++
		}
		return nil
	})

	return stats, err
}

func validateXML(path string) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read xml %s: %w", path, err)
	}

	var marker struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(payload, &marker); err != nil {
		return fmt.Errorf("parse xml %s: %w", path, err)
	}

	switch marker.XMLName.Local {
	case "decoder":
		var decoder wazuhDecoder
		if err := xml.Unmarshal(payload, &decoder); err != nil {
			return fmt.Errorf("decode decoder xml %s: %w", path, err)
		}
		if strings.TrimSpace(decoder.Name) == "" {
			return fmt.Errorf("decoder missing name in %s", path)
		}
	case "rule":
		var rule wazuhRule
		if err := xml.Unmarshal(payload, &rule); err != nil {
			return fmt.Errorf("decode rule xml %s: %w", path, err)
		}
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("rule missing id in %s", path)
		}
	}

	return nil
}

func validateYAML(path string) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml %s: %w", path, err)
	}

	if strings.Contains(path, "/sigma/") {
		var rule sigmaRule
		if err := yaml.Unmarshal(payload, &rule); err != nil {
			return fmt.Errorf("parse sigma yaml %s: %w", path, err)
		}
		if strings.TrimSpace(rule.Title) == "" {
			return fmt.Errorf("sigma rule missing title in %s", path)
		}
		if rule.Logsource == nil {
			return fmt.Errorf("sigma rule missing logsource in %s", path)
		}
		if rule.Detection == nil {
			return fmt.Errorf("sigma rule missing detection in %s", path)
		}
		return nil
	}

	var generic any
	if err := yaml.Unmarshal(payload, &generic); err != nil {
		return fmt.Errorf("parse yaml %s: %w", path, err)
	}
	return nil
}
