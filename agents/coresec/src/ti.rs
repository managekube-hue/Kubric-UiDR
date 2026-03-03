//! Threat Intelligence module - MalwareBazaar API integration.
//! Threat Intelligence module - MalwareBazaar API integration.
//! Queries the MalwareBazaar API (https://mb-api.abuse.ch/api/v1/) to enrich
//! file hash observables with malware family, tags, and threat intel data.
//!
//! All data from MalwareBazaar is published under CC0 (public domain).
//!
//! # NATS Subject
//! Enrichment results are published to:
//!   `kubric.<tenant_id>.endpoint.ti.v1`
//!
//! # Rate limiting
//! MalwareBazaar allows ~10 req/s unauthenticated. This module caches hashes
//! locally to avoid redundant lookups within a 24-hour window.

use std::collections::HashMap;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tracing::{debug, warn};

const MALWAREBAZAAR_URL: &str = "https://mb-api.abuse.ch/api/v1/";
const CACHE_TTL_SECS: u64 = 86_400; // 24 hours
const REQUEST_TIMEOUT_SECS: u64 = 10;

// ?? Public types ??????????????????????????????????????????????????????????????

/// Enriched threat intel result for a file hash.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TiResult {
    pub hash:           String,
    pub hash_type:      String,
    pub found:          bool,
    pub malware_family: Option<String>,
    pub tags:           Vec<String>,
    pub signature:      Option<String>,
    pub first_seen:     Option<String>,
    pub file_name:      Option<String>,
    pub file_type:      Option<String>,
    pub reporter:       Option<String>,
    pub confidence:     u8,
}

impl TiResult {
    pub fn not_found(hash: String, hash_type: String) -> Self {
        TiResult {
            hash,
            hash_type,
            found: false,
            malware_family: None,
            tags: vec![],
            signature: None,
            first_seen: None,
            file_name: None,
            file_type: None,
            reporter: None,
            confidence: 0,
        }
    }

    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.endpoint.ti.v1", tenant_id)
    }
}

// ?? MalwareBazaar API response types ?????????????????????????????????????????

#[derive(Debug, Deserialize)]
struct MbResponse {
    query_status: String,
    data:         Option<Vec<MbEntry>>,
}

#[derive(Debug, Deserialize)]
struct MbEntry {
    sha256_hash:    Option<String>,
    md5_hash:       Option<String>,
    signature:      Option<String>,
    file_name:      Option<String>,
    file_type:      Option<String>,
    first_seen:     Option<String>,
    reporter:       Option<String>,
    tags:           Option<Vec<String>>,
}

// ?? Cache entry ???????????????????????????????????????????????????????????????

struct CacheEntry {
    result:    TiResult,
    cached_at: u64,
}

// ?? TiEngine ??????????????????????????????????????????????????????????????????

/// MalwareBazaar threat intelligence engine with local result cache.
pub struct TiEngine {
    client: Client,
    cache:  HashMap<String, CacheEntry>,
}

impl TiEngine {
    pub fn new() -> Result<Self> {
        let client = Client::builder()
            .timeout(Duration::from_secs(REQUEST_TIMEOUT_SECS))
            .user_agent("Kubric-CoreSec/0.2")
            .build()?;
        Ok(TiEngine {
            client,
            cache: HashMap::new(),
        })
    }

    /// Lookup a SHA-256 hash in MalwareBazaar.
    ///
    /// Returns a cached result if available and within TTL.
    pub async fn lookup_sha256(&mut self, hash: &str) -> TiResult {
        self.lookup(hash, "sha256").await
    }

    /// Lookup an MD5 hash in MalwareBazaar.
    pub async fn lookup_md5(&mut self, hash: &str) -> TiResult {
        self.lookup(hash, "md5").await
    }

    async fn lookup(&mut self, hash: &str, hash_type: &str) -> TiResult {
        let key = format!("{}:{}", hash_type, hash);

        // Return cached result if still fresh
        let now = now_secs();
        if let Some(entry) = self.cache.get(&key) {
            if now.saturating_sub(entry.cached_at) < CACHE_TTL_SECS {
                debug!(hash = %hash, "TI cache hit");
                return entry.result.clone();
            }
        }

        let result = match self.query_api(hash, hash_type).await {
            Ok(r)  => r,
            Err(e) => {
                warn!(hash = %hash, error = %e, "MalwareBazaar query failed");
                TiResult::not_found(hash.to_string(), hash_type.to_string())
            }
        };

        self.cache.insert(key, CacheEntry { result: result.clone(), cached_at: now });
        result
    }

    async fn query_api(&self, hash: &str, hash_type: &str) -> Result<TiResult> {
        let query_key = match hash_type {
            "md5"    => "get_info",
            "sha256" => "get_info",
            _        => "get_info",
        };

        let params = [
            ("query", query_key),
            ("hash", hash),
        ];

        let resp = self.client
            .post(MALWAREBAZAAR_URL)
            .form(&params)
            .send()
            .await?;

        if !resp.status().is_success() {
            anyhow::bail!("MalwareBazaar HTTP {}", resp.status());
        }

        let mb: MbResponse = resp.json().await?;

        if mb.query_status != "ok" || mb.data.is_none() {
            return Ok(TiResult::not_found(hash.to_string(), hash_type.to_string()));
        }

        let entries = mb.data.unwrap_or_default();
        if entries.is_empty() {
            return Ok(TiResult::not_found(hash.to_string(), hash_type.to_string()));
        }

        let entry = &entries[0];
        Ok(TiResult {
            hash:           hash.to_string(),
            hash_type:      hash_type.to_string(),
            found:          true,
            malware_family: entry.signature.clone(),
            tags:           entry.tags.clone().unwrap_or_default(),
            signature:      entry.signature.clone(),
            first_seen:     entry.first_seen.clone(),
            file_name:      entry.file_name.clone(),
            file_type:      entry.file_type.clone(),
            reporter:       entry.reporter.clone(),
            confidence:     90,
        })
    }

    /// Evict stale cache entries older than TTL.
    pub fn prune_cache(&mut self) {
        let now = now_secs();
        self.cache.retain(|_, v| now.saturating_sub(v.cached_at) < CACHE_TTL_SECS);
    }
}

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ti_result_not_found() {
        let r = TiResult::not_found("abc123".into(), "sha256".into());
        assert!(!r.found);
        assert_eq!(r.confidence, 0);
        assert!(r.malware_family.is_none());
    }

    #[test]
    fn nats_subject_format() {
        assert_eq!(
            TiResult::nats_subject("acme"),
            "kubric.acme.endpoint.ti.v1"
        );
    }
}
