package canonical

import "time"

type Provenance struct {
	SourceSystem string    `json:"source_system"`
	UpstreamRepo string    `json:"upstream_repo"`
	UpstreamRef  string    `json:"upstream_ref"`
	UpstreamPath string    `json:"upstream_path"`
	License      string    `json:"license"`
	IngestedAt   time.Time `json:"ingested_at"`
}

type DecoderSpec struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ProgramName string            `json:"program_name,omitempty"`
	Regex       []string          `json:"regex,omitempty"`
	Order       []string          `json:"order,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
	Provenance  Provenance        `json:"provenance"`
}

type DetectionSpec struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Level       string            `json:"level,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Condition   string            `json:"condition,omitempty"`
	References  []string          `json:"references,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Provenance  Provenance        `json:"provenance"`
}

type ConfigCheckSpec struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Command     string            `json:"command,omitempty"`
	Expected    string            `json:"expected,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Provenance  Provenance        `json:"provenance"`
}
