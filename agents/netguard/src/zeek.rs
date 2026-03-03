//! Zeek integration for NetGuard.
//!
//! Executes Zeek scripts from vendored directories against an optional pcap
//! source when enabled via environment variables.

use std::path::{Path, PathBuf};
use std::process::Command;
use serde::Deserialize;
use tracing::{info, warn};

#[derive(Debug, Clone)]
pub struct Ja3Event {
    pub src_ip: String,
    pub dst_ip: String,
    pub ja3: String,
    pub ja3s: String,
    pub server_name: String,
}

pub struct ZeekRunner {
    zeek_bin: String,
    scripts: Vec<PathBuf>,
    pcap_path: PathBuf,
}

impl ZeekRunner {
    pub fn from_env() -> Option<Self> {
        let enabled = std::env::var("KUBRIC_ENABLE_ZEEK")
            .map(|v| v.eq_ignore_ascii_case("true") || v == "1")
            .unwrap_or(false);
        if !enabled {
            return None;
        }

        let zeek_bin = std::env::var("KUBRIC_ZEEK_BIN").unwrap_or_else(|_| "zeek".to_string());
        let script_dir = std::env::var("KUBRIC_ZEEK_SCRIPT_DIR")
            .unwrap_or_else(|_| "vendor/zeek/scripts".to_string());
        let pcap_path = std::env::var("KUBRIC_ZEEK_PCAP")
            .unwrap_or_else(|_| "".to_string());
        if pcap_path.is_empty() {
            warn!("KUBRIC_ENABLE_ZEEK=true but KUBRIC_ZEEK_PCAP is not set; Zeek execution disabled");
            return None;
        }

        let scripts = collect_zeek_scripts(Path::new(&script_dir));
        Some(Self {
            zeek_bin,
            scripts,
            pcap_path: PathBuf::from(pcap_path),
        })
    }

    pub fn script_count(&self) -> usize {
        self.scripts.len()
    }

    pub fn run_once(&self) -> Vec<Ja3Event> {
        if !self.pcap_path.exists() {
            warn!(pcap = %self.pcap_path.display(), "Zeek pcap file does not exist");
            return Vec::new();
        }

        let mut cmd = Command::new(&self.zeek_bin);
        cmd.arg("-r").arg(&self.pcap_path);
        for script in &self.scripts {
            cmd.arg(script);
        }

        match cmd.output() {
            Ok(out) => {
                if out.status.success() {
                    info!(scripts = self.scripts.len(), pcap = %self.pcap_path.display(), "Zeek execution completed");
                    return read_ja3_events(Path::new("ssl.log"));
                } else {
                    let stderr = String::from_utf8_lossy(&out.stderr);
                    warn!(status = %out.status, stderr = %stderr, "Zeek execution failed");
                }
            }
            Err(e) => {
                warn!(error = %e, bin = %self.zeek_bin, "Failed to launch Zeek");
            }
        }
        Vec::new()
    }
}

#[derive(Deserialize)]
struct ZeekSSL {
    #[serde(default)]
    id_orig_h: String,
    #[serde(default)]
    id_resp_h: String,
    #[serde(default)]
    ja3: String,
    #[serde(default)]
    ja3s: String,
    #[serde(default)]
    server_name: String,
}

fn read_ja3_events(path: &Path) -> Vec<Ja3Event> {
    let Ok(content) = std::fs::read_to_string(path) else {
        return Vec::new();
    };
    let mut out = Vec::new();
    for line in content.lines() {
        let line = line.trim();
        if line.is_empty() || !line.starts_with('{') {
            continue;
        }
        if let Ok(parsed) = serde_json::from_str::<ZeekSSL>(line) {
            if parsed.ja3.is_empty() && parsed.ja3s.is_empty() {
                continue;
            }
            out.push(Ja3Event {
                src_ip: parsed.id_orig_h,
                dst_ip: parsed.id_resp_h,
                ja3: parsed.ja3,
                ja3s: parsed.ja3s,
                server_name: parsed.server_name,
            });
        }
    }
    out
}

fn collect_zeek_scripts(root: &Path) -> Vec<PathBuf> {
    let mut out = Vec::new();
    if !root.exists() {
        return out;
    }
    walk_scripts(root, &mut out);
    out
}

fn walk_scripts(dir: &Path, out: &mut Vec<PathBuf>) {
    let Ok(entries) = std::fs::read_dir(dir) else {
        return;
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            walk_scripts(&p, out);
            continue;
        }
        if p.extension().and_then(|e| e.to_str()) == Some("zeek") {
            out.push(p);
        }
    }
}
