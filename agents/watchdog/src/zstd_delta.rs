//! ZSTD delta compression — compresses agent binary updates using delta
//! encoding to minimize OTA download size.
//!
//! Uses the `zstd` crate for compression with dictionary-based delta:
//! 1. The old binary is used as a dictionary
//! 2. The new binary is compressed against that dictionary
//! 3. Result is a small delta patch instead of the full binary
//!
//! On the agent side, the same dictionary (old binary) + delta produces
//! the new binary.

use anyhow::{Context, Result};
use std::io::{Read, Write};
use std::path::Path;
use tracing::{info, warn};

/// Compression level for delta patches (1-22, higher = smaller but slower).
const COMPRESSION_LEVEL: i32 = 15;

/// Maximum dictionary size (64 MB — covers most agent binaries).
const MAX_DICT_SIZE: usize = 64 * 1024 * 1024;

/// A delta patch that can reconstruct a new binary from an old one.
#[derive(Debug)]
pub struct DeltaPatch {
    pub compressed_data: Vec<u8>,
    pub old_hash: String,
    pub new_hash: String,
    pub original_size: usize,
}

/// Compress `new_data` using `old_data` as a dictionary (delta compression).
///
/// The result is a compressed blob that, when decompressed with `old_data` as
/// the dictionary, produces `new_data`.
pub fn compress_delta(old_data: &[u8], new_data: &[u8]) -> Result<DeltaPatch> {
    let dict = if old_data.len() > MAX_DICT_SIZE {
        &old_data[..MAX_DICT_SIZE]
    } else {
        old_data
    };

    let mut compressor = zstd::bulk::Compressor::new(COMPRESSION_LEVEL)
        .context("create zstd compressor")?;
    compressor.set_dictionary(COMPRESSION_LEVEL, dict)
        .context("set zstd dictionary")?;
    let compressed = compressor.compress(new_data)
        .context("zstd delta compress failed")?;

    let old_hash = blake3::hash(old_data).to_hex().to_string();
    let new_hash = blake3::hash(new_data).to_hex().to_string();

    let ratio = if new_data.is_empty() {
        0.0
    } else {
        compressed.len() as f64 / new_data.len() as f64 * 100.0
    };

    info!(
        old_size = old_data.len(),
        new_size = new_data.len(),
        delta_size = compressed.len(),
        ratio_pct = format!("{:.1}", ratio),
        "delta patch created"
    );

    Ok(DeltaPatch {
        compressed_data: compressed,
        old_hash,
        new_hash,
        original_size: new_data.len(),
    })
}

/// Decompress a delta patch using the old binary as dictionary.
///
/// Verifies the old binary hash matches before decompression and
/// the new binary hash after decompression.
pub fn decompress_delta(
    old_data: &[u8],
    patch: &DeltaPatch,
) -> Result<Vec<u8>> {
    // Verify old binary hash
    let actual_old_hash = blake3::hash(old_data).to_hex().to_string();
    if actual_old_hash != patch.old_hash {
        anyhow::bail!(
            "old binary hash mismatch: expected {} got {}",
            patch.old_hash,
            actual_old_hash
        );
    }

    let dict = if old_data.len() > MAX_DICT_SIZE {
        &old_data[..MAX_DICT_SIZE]
    } else {
        old_data
    };

    let mut decompressor = zstd::bulk::Decompressor::new()
        .context("create zstd decompressor")?;
    decompressor.set_dictionary(dict)
        .context("set zstd dictionary")?;
    let decompressed = decompressor.decompress(
        &patch.compressed_data,
        patch.original_size,
    )
    .context("zstd delta decompress failed")?;

    // Verify new binary hash
    let actual_new_hash = blake3::hash(&decompressed).to_hex().to_string();
    if actual_new_hash != patch.new_hash {
        anyhow::bail!(
            "new binary hash mismatch: expected {} got {}",
            patch.new_hash,
            actual_new_hash
        );
    }

    info!(size = decompressed.len(), "delta patch applied and verified");
    Ok(decompressed)
}

/// Compress a file using ZSTD standard compression (no delta).
pub fn compress_file(input_path: &Path, output_path: &Path) -> Result<()> {
    let data = std::fs::read(input_path)
        .with_context(|| format!("read {}", input_path.display()))?;

    let compressed = zstd::bulk::compress(&data, COMPRESSION_LEVEL)
        .context("zstd compress failed")?;

    std::fs::write(output_path, &compressed)
        .with_context(|| format!("write {}", output_path.display()))?;

    info!(
        input = %input_path.display(),
        output = %output_path.display(),
        original = data.len(),
        compressed = compressed.len(),
        "file compressed"
    );

    Ok(())
}

/// Decompress a ZSTD-compressed file.
pub fn decompress_file(input_path: &Path, output_path: &Path) -> Result<()> {
    let data = std::fs::read(input_path)
        .with_context(|| format!("read {}", input_path.display()))?;

    let decompressed = zstd::bulk::decompress(&data, MAX_DICT_SIZE)
        .context("zstd decompress failed")?;

    std::fs::write(output_path, &decompressed)
        .with_context(|| format!("write {}", output_path.display()))?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn delta_roundtrip() {
        let old = b"The quick brown fox jumps over the lazy dog. Version 1.0.0";
        let new = b"The quick brown fox jumps over the lazy dog. Version 1.1.0 with patches";

        let patch = compress_delta(old, new).unwrap();
        assert!(!patch.compressed_data.is_empty());

        // Delta should be smaller than the new data
        assert!(patch.compressed_data.len() <= new.len());

        let recovered = decompress_delta(old, &patch).unwrap();
        assert_eq!(recovered, new);
    }

    #[test]
    fn delta_identical_data() {
        let data = b"identical binary content v2.0.0";
        let patch = compress_delta(data, data).unwrap();
        let recovered = decompress_delta(data, &patch).unwrap();
        assert_eq!(recovered, data.to_vec());
    }

    #[test]
    fn delta_rejects_wrong_old_hash() {
        let old = b"original binary";
        let new = b"updated binary";
        let patch = compress_delta(old, new).unwrap();

        let wrong_old = b"different binary";
        let result = decompress_delta(wrong_old, &patch);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("hash mismatch"));
    }

    #[test]
    fn delta_large_data() {
        // Simulate a ~1MB binary with small changes
        let mut old = vec![0x42u8; 1024 * 1024];
        let mut new = old.clone();
        // Change 1% of bytes
        for i in (0..new.len()).step_by(100) {
            new[i] = 0xFF;
        }

        let patch = compress_delta(&old, &new).unwrap();
        // Delta should be significantly smaller than the full binary
        assert!(patch.compressed_data.len() < new.len() / 2);

        let recovered = decompress_delta(&old, &patch).unwrap();
        assert_eq!(recovered, new);
    }

    #[test]
    fn standard_compress_decompress() {
        let dir = std::env::temp_dir().join("kubric_zstd_test");
        let _ = std::fs::create_dir_all(&dir);

        let input = dir.join("input.bin");
        let compressed = dir.join("compressed.zst");
        let output = dir.join("output.bin");

        let data = b"test data for compression roundtrip";
        std::fs::write(&input, data).unwrap();

        compress_file(&input, &compressed).unwrap();
        decompress_file(&compressed, &output).unwrap();

        let recovered = std::fs::read(&output).unwrap();
        assert_eq!(recovered, data);

        let _ = std::fs::remove_dir_all(&dir);
    }
}
