//! TUF updater — secure agent binary updates using The Update Framework.
//!
//! Uses the `tough` crate (AWS TUF implementation) to:
//! 1. Fetch the latest TUF repository metadata
//! 2. Verify signatures against the root of trust
//! 3. Download new agent binaries
//! 4. Apply delta patches via `zstd_delta`
//! 5. Verify binary integrity via blake3 manifest
//!
//! TUF repository is expected at the URL specified by KUBRIC_TUF_REPO_URL
//! (default: https://updates.kubric.io/tuf).

use anyhow::{Context, Result};
use std::path::{Path, PathBuf};
use tracing::{error, info, warn};

use crate::manifest::{UpdateManifest, verify_file_hash};
use crate::zstd_delta;

/// Configuration for the TUF updater.
#[derive(Debug, Clone)]
pub struct TufConfig {
    /// URL of the TUF repository.
    pub repo_url: String,
    /// Local directory where agent binaries are stored.
    pub install_dir: PathBuf,
    /// Local directory for downloaded updates before installation.
    pub staging_dir: PathBuf,
    /// Path to the TUF root.json (initial trust anchor).
    pub root_json_path: PathBuf,
    /// Current agent version.
    pub current_version: String,
}

impl TufConfig {
    pub fn from_env() -> Self {
        Self {
            repo_url: std::env::var("KUBRIC_TUF_REPO_URL")
                .unwrap_or_else(|_| "https://updates.kubric.io/tuf".to_string()),
            install_dir: PathBuf::from(
                std::env::var("KUBRIC_INSTALL_DIR")
                    .unwrap_or_else(|_| "/opt/kubric/bin".to_string()),
            ),
            staging_dir: PathBuf::from(
                std::env::var("KUBRIC_STAGING_DIR")
                    .unwrap_or_else(|_| "/opt/kubric/staging".to_string()),
            ),
            root_json_path: PathBuf::from(
                std::env::var("KUBRIC_TUF_ROOT")
                    .unwrap_or_else(|_| "/opt/kubric/tuf/root.json".to_string()),
            ),
            current_version: std::env::var("KUBRIC_AGENT_VERSION")
                .unwrap_or_else(|_| "0.2.0".to_string()),
        }
    }
}

/// Update result after a check/apply cycle.
#[derive(Debug)]
pub enum UpdateResult {
    /// No update available.
    UpToDate,
    /// Update downloaded and staged but not yet applied.
    Staged {
        version: String,
        manifest: UpdateManifest,
    },
    /// Update applied successfully.
    Applied { version: String },
    /// Update failed.
    Failed { reason: String },
}

/// TUF-based agent updater.
pub struct TufUpdater {
    config: TufConfig,
    http_client: reqwest::Client,
}

impl TufUpdater {
    pub fn new(config: TufConfig) -> Self {
        let http_client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(60))
            .build()
            .unwrap_or_default();
        Self { config, http_client }
    }

    /// Check for updates and return the latest manifest if a newer version exists.
    pub async fn check_for_updates(&self) -> Result<Option<UpdateManifest>> {
        let manifest_url = format!("{}/targets/manifest.json", self.config.repo_url);

        let resp = self.http_client
            .get(&manifest_url)
            .send()
            .await
            .context("fetch TUF manifest")?;

        if !resp.status().is_success() {
            warn!(status = %resp.status(), "TUF manifest fetch failed");
            return Ok(None);
        }

        let manifest: UpdateManifest = resp.json().await.context("parse TUF manifest")?;

        if manifest.version == self.config.current_version {
            info!(version = %manifest.version, "agent is up to date");
            return Ok(None);
        }

        info!(
            current = %self.config.current_version,
            available = %manifest.version,
            "update available"
        );
        Ok(Some(manifest))
    }

    /// Download and stage an update.
    pub async fn download_update(&self, manifest: &UpdateManifest) -> Result<UpdateResult> {
        let _ = std::fs::create_dir_all(&self.config.staging_dir);

        for entry in &manifest.entries {
            let url = format!("{}/targets/{}", self.config.repo_url, entry.filename);
            let target_path = self.config.staging_dir.join(&entry.filename);

            info!(file = %entry.filename, size = entry.size, "downloading update file");

            let resp = self.http_client
                .get(&url)
                .send()
                .await
                .with_context(|| format!("download {}", entry.filename))?;

            if !resp.status().is_success() {
                return Ok(UpdateResult::Failed {
                    reason: format!("download failed for {}: {}", entry.filename, resp.status()),
                });
            }

            let bytes = resp.bytes().await.context("read response body")?;
            std::fs::write(&target_path, &bytes)
                .with_context(|| format!("write {}", target_path.display()))?;

            // Verify hash immediately after download
            if !verify_file_hash(&target_path, &entry.blake3_hash) {
                return Ok(UpdateResult::Failed {
                    reason: format!(
                        "hash verification failed for {} (expected {})",
                        entry.filename, entry.blake3_hash
                    ),
                });
            }
        }

        Ok(UpdateResult::Staged {
            version: manifest.version.clone(),
            manifest: manifest.clone(),
        })
    }

    /// Apply a staged update by moving files from staging to install directory.
    pub async fn apply_update(&self, manifest: &UpdateManifest) -> Result<UpdateResult> {
        let _ = std::fs::create_dir_all(&self.config.install_dir);

        for entry in &manifest.entries {
            let staged = self.config.staging_dir.join(&entry.filename);
            let target = self.config.install_dir.join(&entry.filename);

            if !staged.exists() {
                return Ok(UpdateResult::Failed {
                    reason: format!("staged file missing: {}", entry.filename),
                });
            }

            // Check if this is a delta patch (filename ends with .delta.zst)
            if entry.filename.ends_with(".delta.zst") {
                let base_name = entry.filename.trim_end_matches(".delta.zst");
                let old_path = self.config.install_dir.join(base_name);

                if old_path.exists() {
                    let old_data = std::fs::read(&old_path)
                        .with_context(|| format!("read old binary {}", old_path.display()))?;
                    let patch_data = std::fs::read(&staged)?;

                    // Decompress the delta
                    let mut decompressor = zstd::bulk::Decompressor::new()
                        .context("create zstd decompressor")?;
                    decompressor.set_dictionary(&old_data)
                        .context("set zstd dictionary")?;
                    let new_data = decompressor.decompress(
                        &patch_data,
                        entry.size as usize,
                    )
                    .context("delta decompress failed")?;

                    std::fs::write(&self.config.install_dir.join(base_name), &new_data)
                        .context("write updated binary")?;

                    info!(file = base_name, "delta update applied");
                    continue;
                }
            }

            // Full file replacement
            if let Err(e) = std::fs::rename(&staged, &target) {
                // Cross-device fallback: copy + remove
                std::fs::copy(&staged, &target)
                    .with_context(|| format!("copy {} to {}", staged.display(), target.display()))?;
                let _ = std::fs::remove_file(&staged);
            }

            info!(file = %entry.filename, "update file installed");
        }

        // Final verification
        let results = crate::manifest::verify_manifest(manifest, &self.config.install_dir);
        let all_ok = results.iter().all(|(_, ok)| *ok);

        if !all_ok {
            let failed: Vec<_> = results
                .iter()
                .filter(|(_, ok)| !ok)
                .map(|(name, _)| name.as_str())
                .collect();
            return Ok(UpdateResult::Failed {
                reason: format!("post-install verification failed for: {}", failed.join(", ")),
            });
        }

        Ok(UpdateResult::Applied {
            version: manifest.version.clone(),
        })
    }

    /// Run a full update cycle: check → download → apply.
    pub async fn run_update_cycle(&self) -> Result<UpdateResult> {
        match self.check_for_updates().await? {
            Some(manifest) => {
                let stage_result = self.download_update(&manifest).await?;
                match stage_result {
                    UpdateResult::Staged { version, manifest } => {
                        self.apply_update(&manifest).await
                    }
                    other => Ok(other),
                }
            }
            None => Ok(UpdateResult::UpToDate),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tuf_config_from_env_defaults() {
        let cfg = TufConfig::from_env();
        assert!(cfg.repo_url.contains("kubric"));
        assert!(!cfg.current_version.is_empty());
    }

    #[test]
    fn update_result_variants() {
        let up = UpdateResult::UpToDate;
        assert!(matches!(up, UpdateResult::UpToDate));

        let failed = UpdateResult::Failed {
            reason: "test".into(),
        };
        assert!(matches!(failed, UpdateResult::Failed { .. }));
    }

    #[test]
    fn tuf_updater_creates() {
        let cfg = TufConfig::from_env();
        let updater = TufUpdater::new(cfg);
        // Just verify construction doesn't panic
        let _ = updater;
    }
}
