//! Response Action module - Cortex Responders + TheHive Case Templates.
//! Response Action module - Cortex Responders + TheHive Case Templates
//!
//! # Cortex Responders
//! Cortex responders are Python scripts vendored at `vendor/cortex/responders/`.
//! This module executes them as child processes (subprocess model) - the AGPL 3.0
//! code is never linked into the binary, only invoked via process boundary.
//!
//! # TheHive Case Templates
//! JSON templates from `vendor/thehive/templates/case/` are loaded at startup
//! and used to open structured cases via the TheHive v5 REST API.
//!
//! # NATS Subject
//! Response action results are published to:
//!   `kubric.<tenant_id>.endpoint.response.v1`

use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::Duration;

use anyhow::{Context, Result};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use tokio::process::Command;
use tracing::{error, info, warn};

const RESPONSE_TIMEOUT_SECS: u64 = 30;

// ?? Public types ??????????????????????????????????????????????????????????????

/// Outcome of a response action execution.
#[derive(Debug, Clone, Serialize)]
pub struct ResponseResult {
    pub action:    String,
    pub success:   bool,
    pub output:    String,
    pub error_msg: Option<String>,
}

impl ResponseResult {
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.endpoint.response.v1", tenant_id)
    }
}

/// A TheHive case template loaded from JSON.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaseTemplate {
    pub name:        String,
    pub title_prefix: String,
    pub severity:    u8,
    pub tags:        Vec<String>,
    pub tasks:       Vec<CaseTask>,
    pub custom_fields: Option<Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaseTask {
    pub title:  String,
    pub group:  Option<String>,
    pub description: Option<String>,
}

// ?? Cortex Responder runner ???????????????????????????????????????????????????

/// Executes a Cortex responder Python script as a child process.
///
/// The responder receives a JSON job payload on stdin and emits a JSON
/// result on stdout - this is the standard Cortex Analyzer/Responder protocol.
pub async fn run_cortex_responder(
    responder_script: &Path,
    job_payload: &Value,
) -> Result<ResponseResult> {
    let script_name = responder_script
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("unknown")
        .to_string();

    if !responder_script.exists() {
        warn!(script = %responder_script.display(), "Cortex responder script not found");
        return Ok(ResponseResult {
            action:    script_name.clone(),
            success:   false,
            output:    String::new(),
            error_msg: Some(format!("script not found: {}", responder_script.display())),
        });
    }

    let payload_bytes = serde_json::to_vec(job_payload)?;

    let mut child = Command::new("python3")
        .arg(responder_script)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .context("spawn python3 for Cortex responder")?;

    // Write job payload to stdin
    if let Some(mut stdin) = child.stdin.take() {
        use tokio::io::AsyncWriteExt;
        stdin.write_all(&payload_bytes).await?;
    }

    let result = tokio::time::timeout(
        Duration::from_secs(RESPONSE_TIMEOUT_SECS),
        child.wait_with_output(),
    )
    .await
    .context("Cortex responder timed out")?
    .context("Cortex responder process error")?;

    let stdout = String::from_utf8_lossy(&result.stdout).to_string();
    let stderr = String::from_utf8_lossy(&result.stderr).to_string();
    let success = result.status.success();

    if success {
        info!(script = %script_name, "Cortex responder completed successfully");
    } else {
        error!(script = %script_name, stderr = %stderr, "Cortex responder failed");
    }

    Ok(ResponseResult {
        action:    script_name,
        success,
        output:    stdout,
        error_msg: if stderr.is_empty() { None } else { Some(stderr) },
    })
}

// ?? TheHive case management ???????????????????????????????????????????????????

/// TheHive v5 API client for case creation.
pub struct TheHiveClient {
    client:   Client,
    base_url: String,
    api_key:  String,
    templates: Vec<CaseTemplate>,
}

impl TheHiveClient {
    /// Create a new client. Loads case templates from the given directory.
    pub fn new(base_url: &str, api_key: &str, template_dir: &str) -> Result<Self> {
        let client = Client::builder()
            .timeout(Duration::from_secs(30))
            .build()?;

        let templates = load_case_templates(Path::new(template_dir));
        info!(count = templates.len(), dir = %template_dir, "TheHive case templates loaded");

        Ok(TheHiveClient {
            client,
            base_url: base_url.to_string(),
            api_key: api_key.to_string(),
            templates,
        })
    }

    /// Open a new TheHive case for the given alert.
    pub async fn create_case(
        &self,
        template_name: &str,
        title: &str,
        description: &str,
        severity: u8,
        tags: Vec<String>,
        observables: Vec<Value>,
    ) -> Result<Value> {
        let template = self.templates
            .iter()
            .find(|t| t.name == template_name);

        let mut case_body = serde_json::json!({
            "title":       title,
            "description": description,
            "severity":    severity,
            "tags":        tags,
            "flag":        false,
            "tlp":         2,
            "pap":         2,
        });

        // Merge template tasks if template found
        if let Some(tmpl) = template {
            case_body["tasks"] = serde_json::to_value(&tmpl.tasks)?;
            if let Some(cf) = &tmpl.custom_fields {
                case_body["customFields"] = cf.clone();
            }
        }

        let resp = self.client
            .post(format!("{}/api/v1/case", self.base_url))
            .header("Authorization", format!("Bearer {}", self.api_key))
            .json(&case_body)
            .send()
            .await
            .context("TheHive create case request failed")?;

        if !resp.status().is_success() {
            let status = resp.status();
            let body = resp.text().await.unwrap_or_default();
            anyhow::bail!("TheHive API error {}: {}", status, body);
        }

        // Add observables to the case
        let case: Value = resp.json().await?;
        let case_id = case["_id"].as_str().unwrap_or("").to_string();

        for obs in &observables {
            if let Err(e) = self.add_observable(&case_id, obs).await {
                warn!(case_id = %case_id, error = %e, "failed to add observable");
            }
        }

        info!(case_id = %case_id, title = %title, "TheHive case created");
        Ok(case)
    }

    async fn add_observable(&self, case_id: &str, observable: &Value) -> Result<()> {
        let resp = self.client
            .post(format!("{}/api/v1/case/{}/observable", self.base_url, case_id))
            .header("Authorization", format!("Bearer {}", self.api_key))
            .json(observable)
            .send()
            .await?;

        if !resp.status().is_success() {
            anyhow::bail!("add observable failed: {}", resp.status());
        }
        Ok(())
    }

    /// Return names of loaded templates.
    pub fn template_names(&self) -> Vec<&str> {
        self.templates.iter().map(|t| t.name.as_str()).collect()
    }
}

// ?? Cortex responder catalog ??????????????????????????????????????????????????

/// Lists all Python responder scripts available in the given directory.
pub fn list_responders(dir: &str) -> Vec<PathBuf> {
    let path = Path::new(dir);
    if !path.exists() {
        return vec![];
    }
    let Ok(entries) = std::fs::read_dir(path) else {
        return vec![];
    };
    entries
        .flatten()
        .map(|e| e.path())
        .filter(|p| p.extension().and_then(|e| e.to_str()) == Some("py"))
        .collect()
}

// ?? Internal helpers ??????????????????????????????????????????????????????????

fn load_case_templates(dir: &Path) -> Vec<CaseTemplate> {
    if !dir.exists() {
        return vec![];
    }
    let Ok(entries) = std::fs::read_dir(dir) else {
        return vec![];
    };
    let mut templates = Vec::new();
    for entry in entries.flatten() {
        let p = entry.path();
        if p.extension().and_then(|e| e.to_str()) == Some("json") {
            match std::fs::read_to_string(&p) {
                Ok(content) => match serde_json::from_str::<CaseTemplate>(&content) {
                    Ok(t)  => templates.push(t),
                    Err(e) => warn!(path = %p.display(), error = %e, "invalid case template"),
                },
                Err(e) => warn!(path = %p.display(), error = %e, "cannot read case template"),
            }
        }
    }
    templates
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn response_result_nats_subject() {
        assert_eq!(
            ResponseResult::nats_subject("acme"),
            "kubric.acme.endpoint.response.v1"
        );
    }

    #[test]
    fn list_responders_missing_dir() {
        let result = list_responders("/nonexistent/path/to/responders");
        assert!(result.is_empty());
    }

    #[test]
    fn load_case_templates_missing_dir() {
        let templates = load_case_templates(Path::new("/nonexistent"));
        assert!(templates.is_empty());
    }
}
