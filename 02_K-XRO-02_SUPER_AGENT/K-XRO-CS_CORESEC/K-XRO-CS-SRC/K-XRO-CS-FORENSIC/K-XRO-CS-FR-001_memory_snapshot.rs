//! K-XRO-CS-FR-001 — Memory forensics: process memory-map capture
//!
//! Parses `/proc/{pid}/maps` to enumerate memory regions, identifies
//! suspicious executable-writable-anonymous (RWX / W+X) pages that may
//! indicate injected shellcode, and can dump individual regions to disk by
//! reading `/proc/{pid}/mem`.
//!
//! All functions are Linux-only at the implementation level; the types compile
//! on any platform so they can be used in cross-compiled artefacts.
//!
//! Production counterpart: forensic subsystem (not yet in main coresec tree)
//!
//! Dependencies in Cargo.toml:
//!   anyhow  = "1"
//!   serde   = { version = "1", features = ["derive"] }
//!   tracing = "0.1"
//!
//! Module: K-XRO-CS-FR-001

use std::fs;
use std::io::{Read, Seek, SeekFrom};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use tracing::{debug, warn};

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

/// A single VMA (Virtual Memory Area) parsed from `/proc/{pid}/maps`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemRegion {
    /// Start virtual address (inclusive).
    pub start: u64,
    /// End virtual address (exclusive).
    pub end: u64,
    /// Permission string as reported by the kernel, e.g. `rwxp`.
    pub permissions: String,
    /// File offset (for file-backed mappings).
    pub offset: u64,
    /// `major:minor` device pair as a string.
    pub device: String,
    /// Inode number (0 for anonymous mappings).
    pub inode: u64,
    /// Pathname or pseudo-name (`[heap]`, `[stack]`, empty for anonymous).
    pub path: String,
}

impl MemRegion {
    /// Size of this region in bytes.
    pub fn size(&self) -> u64 {
        self.end.saturating_sub(self.start)
    }

    /// Returns `true` if the region is both writable and executable.
    pub fn is_wx(&self) -> bool {
        self.permissions.contains('w') && self.permissions.contains('x')
    }

    /// Returns `true` for anonymous memory (no backing file, inode == 0).
    pub fn is_anonymous(&self) -> bool {
        self.inode == 0 && self.path.is_empty()
    }

    /// Returns `true` for the classic RWX shellcode indicator.
    pub fn is_rwx(&self) -> bool {
        self.permissions.starts_with("rwx")
    }
}

/// Aggregated memory view of a process.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProcessMemory {
    /// Target process ID.
    pub pid: u32,
    /// All mapped regions from `/proc/{pid}/maps`.
    pub regions: Vec<MemRegion>,
    /// Total bytes of anonymous (unfile-backed) mappings.
    pub total_anonymous: u64,
    /// Total bytes of file-backed mappings.
    pub total_mapped: u64,
}

/// Complete forensic snapshot of a process at a point in time.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemSnapshot {
    /// Target PID.
    pub pid: u32,
    /// Executable path (`/proc/{pid}/exe` symlink target).
    pub executable: String,
    /// All regions.
    pub regions: Vec<MemRegion>,
    /// Regions flagged as suspicious (W+X or RWX anonymous).
    pub suspicious_regions: Vec<MemRegion>,
    /// Unix seconds when the snapshot was taken.
    pub timestamp: u64,
}

// ---------------------------------------------------------------------------
// Core functions
// ---------------------------------------------------------------------------

/// Parse `/proc/{pid}/maps` and return all memory regions.
///
/// Each line in the maps file has the format:
/// ```
/// address           perms offset  dev   inode   pathname
/// 7f0a3c000000-7f0a3c001000 r--p 00000000 fd:01 123456 /usr/lib/libc.so.6
/// ```
pub fn read_proc_maps(pid: u32) -> Result<Vec<MemRegion>> {
    #[cfg(target_os = "linux")]
    {
        let path = format!("/proc/{}/maps", pid);
        let content = fs::read_to_string(&path)
            .with_context(|| format!("cannot read {}", path))?;
        parse_maps(&content)
    }

    #[cfg(not(target_os = "linux"))]
    {
        anyhow::bail!("read_proc_maps is only supported on Linux (pid={})", pid)
    }
}

/// Parse the text content of a `/proc/{pid}/maps` file.
pub fn parse_maps(content: &str) -> Result<Vec<MemRegion>> {
    let mut regions = Vec::new();

    for line in content.lines() {
        let region = parse_maps_line(line).with_context(|| format!("parse error on: {}", line))?;
        regions.push(region);
    }

    Ok(regions)
}

fn parse_maps_line(line: &str) -> Result<MemRegion> {
    //  7f0a3c000000-7f0a3c001000 r--p 00000000 fd:01 123456 /usr/lib/libc.so.6
    //  ^address range^           ^per ^offset   ^dev  ^inode ^pathname (optional)
    let mut cols = line.splitn(6, ' ');

    let addr_range = cols.next().context("missing address range")?;
    let permissions = cols.next().context("missing permissions")?.to_string();
    let offset_hex  = cols.next().context("missing offset")?;
    let device      = cols.next().context("missing device")?.to_string();
    let inode_str   = cols.next().context("missing inode")?;
    let path        = cols.next().unwrap_or("").trim().to_string();

    let (start_str, end_str) = addr_range
        .split_once('-')
        .context("malformed address range")?;

    Ok(MemRegion {
        start:       u64::from_str_radix(start_str, 16)?,
        end:         u64::from_str_radix(end_str,   16)?,
        permissions,
        offset:      u64::from_str_radix(offset_hex, 16)?,
        device,
        inode:       inode_str.parse::<u64>()?,
        path,
    })
}

/// Return regions that are both writable and executable and anonymous —
/// the classic indicators of injected shellcode or heap spray.
pub fn scan_for_shellcode(pid: u32) -> Vec<MemRegion> {
    match read_proc_maps(pid) {
        Ok(regions) => {
            regions
                .into_iter()
                .filter(|r| r.is_wx() && r.is_anonymous())
                .collect()
        }
        Err(e) => {
            warn!(pid, err = %e, "scan_for_shellcode: cannot read maps");
            Vec::new()
        }
    }
}

/// Build a `ProcessMemory` aggregating all regions and accounting totals.
pub fn capture_snapshot(pid: u32) -> Result<ProcessMemory> {
    let regions = read_proc_maps(pid)?;

    let mut total_anonymous: u64 = 0;
    let mut total_mapped:    u64 = 0;

    for r in &regions {
        if r.is_anonymous() {
            total_anonymous += r.size();
        } else {
            total_mapped += r.size();
        }
    }

    Ok(ProcessMemory { pid, regions, total_anonymous, total_mapped })
}

/// Dump a single memory region to a file by reading `/proc/{pid}/mem`.
///
/// Returns the number of bytes written.
///
/// # Safety
/// Reading another process's memory via `/proc/{pid}/mem` requires:
/// - `CAP_SYS_PTRACE` or `PTRACE_ATTACH`
/// - The target process must still be running (not in zombie state)
///
/// Unreadable pages (holes, guard pages) will return errors; partial reads
/// are reflected in the returned byte count.
pub fn dump_region(pid: u32, region: &MemRegion, output_path: &Path) -> Result<usize> {
    #[cfg(target_os = "linux")]
    {
        let mem_path = format!("/proc/{}/mem", pid);
        let mut mem_file = fs::OpenOptions::new()
            .read(true)
            .open(&mem_path)
            .with_context(|| format!("open {}", mem_path))?;

        mem_file
            .seek(SeekFrom::Start(region.start))
            .with_context(|| format!("seek to {:#x}", region.start))?;

        let region_size = region.size() as usize;
        let mut buf = vec![0u8; region_size];

        let mut total_read = 0usize;
        loop {
            match mem_file.read(&mut buf[total_read..]) {
                Ok(0) => break,
                Ok(n) => { total_read += n; }
                Err(e) if e.kind() == std::io::ErrorKind::UnexpectedEof => break,
                Err(e) => {
                    // Partial reads are normal for mixed-protection regions
                    debug!(
                        pid, addr = %format!("{:#x}", region.start + total_read as u64),
                        err = %e,
                        "partial dump — truncating at readable boundary"
                    );
                    break;
                }
            }
        }

        fs::write(output_path, &buf[..total_read])
            .with_context(|| format!("write dump to {}", output_path.display()))?;

        Ok(total_read)
    }

    #[cfg(not(target_os = "linux"))]
    {
        anyhow::bail!("dump_region is only supported on Linux")
    }
}

/// Build a complete `MemSnapshot` for a process including suspicious region detection.
pub fn build_mem_snapshot(pid: u32) -> Result<MemSnapshot> {
    let pm = capture_snapshot(pid)?;

    let suspicious_regions: Vec<MemRegion> = pm
        .regions
        .iter()
        .filter(|r| r.is_wx() && r.is_anonymous())
        .cloned()
        .collect();

    let executable = {
        #[cfg(target_os = "linux")]
        {
            fs::read_link(format!("/proc/{}/exe", pid))
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_else(|_| "<unknown>".into())
        }
        #[cfg(not(target_os = "linux"))]
        { "<unknown>".into() }
    };

    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();

    Ok(MemSnapshot {
        pid,
        executable,
        regions: pm.regions,
        suspicious_regions,
        timestamp,
    })
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    const SAMPLE_MAPS: &str = "\
7f0a3c000000-7f0a3c001000 r--p 00000000 fd:01 123456 /usr/lib/libc.so.6
7f0a3c001000-7f0a3c002000 r-xp 00001000 fd:01 123456 /usr/lib/libc.so.6
7f0a40000000-7f0a40010000 rwxp 00000000 00:00 0
7f0a50000000-7f0a50008000 rwxp 00000000 00:00 0 [heap]
55c0a0000000-55c0a0001000 r--p 00000000 fd:01 999 /usr/bin/bash";

    #[test]
    fn parse_maps_basic() {
        let regions = parse_maps(SAMPLE_MAPS).unwrap();
        assert_eq!(regions.len(), 5);
    }

    #[test]
    fn region_permissions_rwx() {
        let regions = parse_maps(SAMPLE_MAPS).unwrap();
        let rwx: Vec<_> = regions.iter().filter(|r| r.is_rwx()).collect();
        assert_eq!(rwx.len(), 2, "expected 2 rwx regions");
    }

    #[test]
    fn region_is_anonymous() {
        let regions = parse_maps(SAMPLE_MAPS).unwrap();
        // Third region: rwxp, inode 0, empty path → anonymous
        let anon_rwx: Vec<_> = regions.iter().filter(|r| r.is_wx() && r.is_anonymous()).collect();
        assert_eq!(anon_rwx.len(), 1, "only fully anonymous W+X region");
    }

    #[test]
    fn region_size_calculation() {
        let r = MemRegion {
            start: 0x1000,
            end:   0x2000,
            permissions: "rw-p".into(),
            offset: 0,
            device: "fd:01".into(),
            inode: 0,
            path: String::new(),
        };
        assert_eq!(r.size(), 0x1000);
    }

    #[test]
    fn parse_maps_line_with_no_path() {
        let line = "7fff00000000-7fff00001000 rw-p 00000000 00:00 0";
        let r = parse_maps_line(line).unwrap();
        assert!(r.path.is_empty());
        assert_eq!(r.inode, 0);
        assert!(r.is_anonymous());
    }

    #[test]
    fn parse_maps_line_exec_stack() {
        let line = "7fff12340000-7fff12360000 rwxp 00000000 00:00 0 [stack]";
        let r = parse_maps_line(line).unwrap();
        assert_eq!(r.path, "[stack]");
        // [stack] has a path so is NOT anonymous
        assert!(!r.is_anonymous());
        assert!(r.is_rwx());
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn capture_snapshot_current_process() {
        // Should succeed for the current process (pid == own pid)
        let pid = std::process::id();
        let snap = build_mem_snapshot(pid).unwrap();
        assert!(snap.pid == pid);
        assert!(!snap.regions.is_empty(), "current process must have mapped regions");
    }
}
