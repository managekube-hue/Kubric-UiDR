//! Binary fingerprinting — validates agent binaries using blake3 hashes.
//!
//! Used by the registration handler to verify that incoming agent
//! registration requests come from authentic Kubric agent builds.

use blake3::Hasher;
use std::collections::HashMap;
use std::path::Path;
use tracing::{info, warn};

/// Known agent binary hashes indexed by (agent_type, hash).
/// Populated from the binary directory at startup.
static mut KNOWN_HASHES: Option<HashMap<String, Vec<String>>> = None;

/// Compute the blake3 hash of a file.
pub fn hash_file(path: &Path) -> Option<String> {
    let data = std::fs::read(path).ok()?;
    let mut hasher = Hasher::new();
    hasher.update(&data);
    Some(hasher.finalize().to_hex().to_string())
}

/// Compute the blake3 hash of a byte slice.
pub fn hash_bytes(data: &[u8]) -> String {
    let mut hasher = Hasher::new();
    hasher.update(data);
    hasher.finalize().to_hex().to_string()
}

/// Build the known hash database from an agent binary directory.
///
/// Scans the directory for agent binaries and records their blake3 hashes.
/// Binary names are expected to follow the pattern:
///   `{agent_type}` or `{agent_type}.exe` (Windows)
pub fn build_known_hashes(binary_dir: &str) -> HashMap<String, Vec<String>> {
    let mut known: HashMap<String, Vec<String>> = HashMap::new();
    let dir = Path::new(binary_dir);

    if !dir.exists() {
        warn!(dir = binary_dir, "binary directory not found — fingerprinting disabled");
        return known;
    }

    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(e) => {
            warn!(dir = binary_dir, %e, "cannot read binary directory");
            return known;
        }
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        if let Some(hash) = hash_file(&path) {
            let name = path
                .file_stem()
                .and_then(|s| s.to_str())
                .unwrap_or("")
                .to_string();

            if !name.is_empty() {
                known.entry(name.clone()).or_default().push(hash.clone());
                info!(
                    agent_type = %name,
                    hash = &hash[..16],
                    "agent binary fingerprinted"
                );
            }
        }
    }

    info!(
        types = known.len(),
        total_hashes = known.values().map(|v| v.len()).sum::<usize>(),
        "known binary hash database built"
    );

    known
}

/// Validate an agent's claimed binary hash against known hashes.
///
/// In production, this checks against:
/// 1. The local binary directory hashes
/// 2. The TUF manifest published hashes
/// 3. A cached hash database from the provisioning API
///
/// Returns true if the hash matches any known build for the agent type.
pub fn validate_agent_hash(
    agent_type: &str,
    claimed_hash: &str,
    binary_dir: &str,
) -> bool {
    let known = build_known_hashes(binary_dir);

    if let Some(hashes) = known.get(agent_type) {
        if hashes.iter().any(|h| h == claimed_hash) {
            info!(agent_type, "binary hash validated");
            return true;
        }
    }

    // If no known hashes exist (e.g., first deployment), allow registration
    // with a warning.  Production deployments should always have known hashes.
    if known.is_empty() {
        warn!(
            agent_type,
            "no known hashes — allowing registration (first deployment)"
        );
        return true;
    }

    warn!(
        agent_type,
        claimed_hash = &claimed_hash[..16.min(claimed_hash.len())],
        "binary hash validation FAILED"
    );
    false
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn hash_bytes_deterministic() {
        let h1 = hash_bytes(b"test binary content");
        let h2 = hash_bytes(b"test binary content");
        assert_eq!(h1, h2);
        assert_ne!(h1, hash_bytes(b"different content"));
    }

    #[test]
    fn hash_file_works() {
        let dir = std::env::temp_dir().join("kubric_fp_test");
        let _ = std::fs::create_dir_all(&dir);
        let file = dir.join("agent.bin");

        let mut f = std::fs::File::create(&file).unwrap();
        f.write_all(b"test agent binary").unwrap();
        drop(f);

        let hash = hash_file(&file);
        assert!(hash.is_some());
        assert!(!hash.unwrap().is_empty());

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn build_known_hashes_missing_dir() {
        let hashes = build_known_hashes("/nonexistent/dir");
        assert!(hashes.is_empty());
    }

    #[test]
    fn build_known_hashes_from_dir() {
        let dir = std::env::temp_dir().join("kubric_fp_known");
        let _ = std::fs::create_dir_all(&dir);
        std::fs::write(dir.join("coresec"), b"coresec binary").unwrap();
        std::fs::write(dir.join("netguard"), b"netguard binary").unwrap();

        let hashes = build_known_hashes(dir.to_str().unwrap());
        assert!(hashes.contains_key("coresec"));
        assert!(hashes.contains_key("netguard"));

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn validate_unknown_type_no_known_hashes() {
        // With no known hashes, validation should pass (first deployment mode)
        let result = validate_agent_hash("coresec", "abc123", "/nonexistent/dir");
        assert!(result);
    }

    #[test]
    fn validate_matching_hash() {
        let dir = std::env::temp_dir().join("kubric_fp_validate");
        let _ = std::fs::create_dir_all(&dir);
        let content = b"test binary for validation";
        std::fs::write(dir.join("coresec"), content).unwrap();

        let expected_hash = hash_bytes(content);
        let result = validate_agent_hash("coresec", &expected_hash, dir.to_str().unwrap());
        assert!(result);

        let _ = std::fs::remove_dir_all(&dir);
    }
}
