//! Manifest signer & verifier — validates agent binary integrity using
//! blake3 checksums and ed25519 signatures (via ring).
//!
//! Update manifests are fetched from the TUF repository.  Each manifest
//! contains a list of (filename, blake3_hash, size) entries.

use blake3::Hasher;
use ring::signature::{UnparsedPublicKey, ED25519};
use serde::{Deserialize, Serialize};
use std::path::Path;
use tracing::{info, warn};

/// A single entry in an update manifest.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ManifestEntry {
    pub filename: String,
    pub blake3_hash: String,
    pub size: u64,
}

/// Signed update manifest.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateManifest {
    pub version: String,
    pub entries: Vec<ManifestEntry>,
    /// Ed25519 signature of the canonical JSON of entries (hex-encoded).
    pub signature: String,
}

/// Verify a file's blake3 hash matches the expected value.
pub fn verify_file_hash(path: &Path, expected_hash: &str) -> bool {
    let data = match std::fs::read(path) {
        Ok(d) => d,
        Err(e) => {
            warn!(path = %path.display(), %e, "failed to read file for hash verification");
            return false;
        }
    };
    let mut hasher = Hasher::new();
    hasher.update(&data);
    let actual = hasher.finalize().to_hex().to_string();
    if actual == expected_hash {
        info!(path = %path.display(), "hash verified");
        true
    } else {
        warn!(
            path = %path.display(),
            expected = expected_hash,
            actual = actual,
            "hash mismatch"
        );
        false
    }
}

/// Verify an ed25519 signature over manifest data.
pub fn verify_signature(public_key_bytes: &[u8], message: &[u8], signature_bytes: &[u8]) -> bool {
    let public_key = UnparsedPublicKey::new(&ED25519, public_key_bytes);
    public_key.verify(message, signature_bytes).is_ok()
}

/// Generate a blake3 hash for a byte slice.
pub fn blake3_hash(data: &[u8]) -> String {
    let mut hasher = Hasher::new();
    hasher.update(data);
    hasher.finalize().to_hex().to_string()
}

/// Verify all entries in a manifest against files in a directory.
pub fn verify_manifest(manifest: &UpdateManifest, base_dir: &Path) -> Vec<(String, bool)> {
    manifest
        .entries
        .iter()
        .map(|entry| {
            let path = base_dir.join(&entry.filename);
            let ok = verify_file_hash(&path, &entry.blake3_hash);
            (entry.filename.clone(), ok)
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn blake3_hash_deterministic() {
        let h1 = blake3_hash(b"test data");
        let h2 = blake3_hash(b"test data");
        assert_eq!(h1, h2);
        assert_ne!(h1, blake3_hash(b"different data"));
    }

    #[test]
    fn verify_file_hash_works() {
        let dir = std::env::temp_dir().join("kubric_manifest_test");
        let _ = std::fs::create_dir_all(&dir);
        let file = dir.join("agent.bin");
        let mut f = std::fs::File::create(&file).unwrap();
        f.write_all(b"agent binary content").unwrap();
        drop(f);

        let expected = blake3_hash(b"agent binary content");
        assert!(verify_file_hash(&file, &expected));
        assert!(!verify_file_hash(&file, "wrong_hash"));

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn verify_manifest_checks_entries() {
        let dir = std::env::temp_dir().join("kubric_manifest_verify");
        let _ = std::fs::create_dir_all(&dir);

        let content = b"binary content v2";
        std::fs::write(dir.join("agent"), content).unwrap();

        let manifest = UpdateManifest {
            version: "1.0.0".to_string(),
            entries: vec![
                ManifestEntry {
                    filename: "agent".to_string(),
                    blake3_hash: blake3_hash(content),
                    size: content.len() as u64,
                },
                ManifestEntry {
                    filename: "missing_file".to_string(),
                    blake3_hash: "deadbeef".to_string(),
                    size: 0,
                },
            ],
            signature: String::new(),
        };

        let results = verify_manifest(&manifest, &dir);
        assert_eq!(results.len(), 2);
        assert!(results[0].1);  // agent verified
        assert!(!results[1].1); // missing_file failed

        let _ = std::fs::remove_dir_all(&dir);
    }
}
