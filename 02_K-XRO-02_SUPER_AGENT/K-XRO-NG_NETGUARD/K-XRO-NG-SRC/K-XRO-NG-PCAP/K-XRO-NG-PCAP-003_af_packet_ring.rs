//! K-XRO-NG-PCAP-003 — AF_PACKET TPACKET_V3 ring buffer for zero-copy kernel
//! packet capture on Linux.
//!
//! # Architecture
//! Linux `AF_PACKET` sockets with `TPACKET_V3` provide a mmap-based ring buffer
//! that eliminates per-packet system call overhead.  The kernel writes frames into
//! fixed-size blocks; userspace polls with `POLLIN` and walks the ring without
//! performing any extra copies.
//!
//! ## Memory layout (TPACKET_V3)
//! ```text
//! ring_base
//! ├── block[0]  (block_size bytes, default 4 MiB)
//! │   ├── tpacket_block_desc  (status, num_pkts, offset_to_first_pkt …)
//! │   ├── tpacket3_hdr + frame[0]
//! │   ├── tpacket3_hdr + frame[1]
//! │   └── …
//! ├── block[1]
//! └── … block[block_nr-1]
//! ```
//!
//! ## Default RingConfig
//! | Field            | Value     | Rationale                             |
//! |------------------|-----------|---------------------------------------|
//! | `block_size`     | 4 MiB     | Amortises block-retire overhead       |
//! | `block_nr`       | 64        | 256 MiB total ring — fits L3/LLC      |
//! | `frame_size`     | 2048      | Fits jumbo-frame headers              |
//! | `retire_blk_tov` | 60 ms     | Low-latency retirement                |
//!
//! ## Platform
//! Linux only — compile-guarded with `#[cfg(target_os = "linux")]`.
//! Non-Linux builds expose stub types that return `ErrorKind::Unsupported`.

#![allow(dead_code)]

// ─────────────────────────────────────────────────────────────────────────────
// Linux implementation
// ─────────────────────────────────────────────────────────────────────────────
#[cfg(target_os = "linux")]
mod inner {
    use libc::{
        bind, close, mmap, munmap, poll, pollfd, setsockopt, socket, socklen_t, AF_PACKET,
        ETH_P_ALL, MAP_SHARED, POLLIN, PROT_READ, PROT_WRITE, SOCK_NONBLOCK, SOCK_RAW, SOL_PACKET,
    };
    use std::ffi::CString;
    use std::io;
    use std::mem;
    use std::os::raw::{c_int, c_uint, c_void};

    // ── AF_PACKET / TPACKET_V3 numeric constants ──────────────────────────────
    const PACKET_VERSION: c_int = 10;
    const PACKET_RX_RING: c_int = 5;
    const TPACKET_V3: c_int = 2;
    /// Block is owned by the kernel (being filled).
    const TP_STATUS_KERNEL: u32 = 0;
    /// Block is ready for userspace to read.
    const TP_STATUS_USER: u32 = 1;

    // ── C struct mirrors ──────────────────────────────────────────────────────

    /// `tpacket_req3` — passed to `setsockopt(SOL_PACKET, PACKET_RX_RING)`.
    #[repr(C)]
    #[derive(Debug, Default)]
    struct TpacketReq3 {
        /// Block size in bytes (must be power-of-two, >= page size).
        tp_block_size: c_uint,
        /// Number of blocks in the ring.
        tp_block_nr: c_uint,
        /// Frame size (only used as a quota hint in V3, but must be set).
        tp_frame_size: c_uint,
        /// Total frame count across all blocks.
        tp_frame_nr: c_uint,
        /// Block retire timeout in milliseconds (V3-specific).
        tp_retire_blk_tov: c_uint,
        /// Private per-frame space in bytes (unused here).
        tp_sizeof_priv: c_uint,
        /// Feature request bits.
        tp_feature_req_word: c_uint,
    }

    /// First word of every block — contains status, packet count, and offsets.
    /// This mirrors the kernel's `tpacket_block_desc` + `tpacket_hdr_v1` union.
    #[repr(C)]
    struct TpacketBlockDesc {
        version: u32,
        offset_to_priv: u32,
        hdr: TpacketBDHdr,
    }

    #[repr(C)]
    struct TpacketBDHdr {
        block_status: u32,
        num_pkts: u32,
        offset_to_first_pkt: u32,
        blk_len: u32,
        seq_num: u64,
        ts_first_sec: u32,
        ts_first_nsec: u32,
        ts_last_sec: u32,
        ts_last_nsec: u32,
    }

    /// Per-frame header — written by kernel before each captured frame.
    #[repr(C)]
    struct Tpacket3Hdr {
        /// Offset to the next frame header within this block (0 = last frame).
        tp_next_offset: u32,
        tp_sec: u32,
        tp_nsec: u32,
        /// Bytes captured (may be less than `tp_len` if truncated).
        tp_snaplen: u32,
        /// Wire length of the original packet.
        tp_len: u32,
        tp_status: u32,
        /// Offset from this header to the MAC (Ethernet) header.
        tp_mac: u16,
        /// Offset from this header to the network (IP) header.
        tp_net: u16,
        tp_vlan_tci: u16,
        tp_vlan_tpid: u16,
        _padding: [u8; 4],
    }

    /// `sockaddr_ll` — link-layer address used to bind `AF_PACKET` to an iface.
    #[repr(C)]
    struct SockaddrLl {
        sll_family: u16,
        sll_protocol: u16,
        sll_ifindex: i32,
        sll_hatype: u16,
        sll_pkttype: u8,
        sll_halen: u8,
        sll_addr: [u8; 8],
    }

    extern "C" {
        /// Convert interface name to kernel index.
        fn if_nametoindex(ifname: *const libc::c_char) -> c_uint;
    }

    // ── Ring configuration ────────────────────────────────────────────────────

    /// Tuning parameters for the TPACKET_V3 ring buffer.
    #[derive(Debug, Clone)]
    pub struct RingConfig {
        /// Block size in bytes.  Must be power-of-two and >= 4096.  Default: 4 MiB.
        pub block_size: usize,
        /// Number of blocks in the ring.  Default: 64 (256 MiB total).
        pub block_nr: usize,
        /// Frame size quota hint.  Default: 2048.
        pub frame_size: usize,
        /// Block retire timeout in milliseconds.  Default: 60.
        pub retire_blk_tov: u32,
    }

    impl Default for RingConfig {
        fn default() -> Self {
            Self {
                block_size: 1 << 22, // 4 MiB
                block_nr: 64,
                frame_size: 2048,
                retire_blk_tov: 60,
            }
        }
    }

    // ── AfPacketRing ──────────────────────────────────────────────────────────

    /// Zero-copy AF_PACKET TPACKET_V3 ring buffer capture handle.
    ///
    /// Owns the raw socket fd and the mmap'd ring memory.  Dropped via `close()`.
    pub struct AfPacketRing {
        sock: i32,
        ring: *mut u8,
        ring_size: usize,
        block_size: usize,
        block_nr: usize,
        frame_size: usize,
        current_block: usize,
        /// Current frame walk index within `current_block`.
        current_pkt: u32,
        /// Cumulative frames delivered to callers.
        pub frames_captured: u64,
    }

    // SAFETY: AfPacketRing owns the fd and mmap exclusively.
    unsafe impl Send for AfPacketRing {}

    impl AfPacketRing {
        /// Open a TPACKET_V3 ring on `iface` with `RingConfig::default()`.
        pub fn new(iface: &str) -> io::Result<Self> {
            Self::with_config(iface, RingConfig::default())
        }

        /// Open a TPACKET_V3 ring with a custom `RingConfig`.
        ///
        /// Sequence:
        /// 1. `socket(AF_PACKET, SOCK_RAW|SOCK_NONBLOCK, ETH_P_ALL)`
        /// 2. `setsockopt(PACKET_VERSION, TPACKET_V3)`
        /// 3. `setsockopt(PACKET_RX_RING, tpacket_req3{…})`
        /// 4. `mmap(ring_size, MAP_SHARED)`
        /// 5. `bind(sockaddr_ll{ifindex})`
        pub fn with_config(iface: &str, cfg: RingConfig) -> io::Result<Self> {
            if !cfg.block_size.is_power_of_two() || cfg.block_size < 4096 {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidInput,
                    "block_size must be a power of two >= 4096",
                ));
            }
            if cfg.frame_size == 0 || cfg.frame_size > cfg.block_size {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidInput,
                    "frame_size must be > 0 and <= block_size",
                ));
            }
            if cfg.block_nr == 0 {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidInput,
                    "block_nr must be > 0",
                ));
            }

            let frame_nr = (cfg.block_size / cfg.frame_size) * cfg.block_nr;

            unsafe {
                // ── Step 1: create AF_PACKET socket ──────────────────────────
                let protocol = (ETH_P_ALL as u16).to_be() as c_int;
                let sock = socket(AF_PACKET, SOCK_RAW | SOCK_NONBLOCK, protocol);
                if sock < 0 {
                    return Err(io::Error::last_os_error());
                }

                // ── Step 2: require TPACKET_V3 ───────────────────────────────
                let version: c_int = TPACKET_V3;
                let rc = setsockopt(
                    sock,
                    SOL_PACKET,
                    PACKET_VERSION,
                    &version as *const c_int as *const c_void,
                    mem::size_of::<c_int>() as socklen_t,
                );
                if rc < 0 {
                    close(sock);
                    return Err(io::Error::last_os_error());
                }

                // ── Step 3: configure RX ring ────────────────────────────────
                let req = TpacketReq3 {
                    tp_block_size: cfg.block_size as c_uint,
                    tp_block_nr: cfg.block_nr as c_uint,
                    tp_frame_size: cfg.frame_size as c_uint,
                    tp_frame_nr: frame_nr as c_uint,
                    tp_retire_blk_tov: cfg.retire_blk_tov,
                    tp_sizeof_priv: 0,
                    tp_feature_req_word: 0,
                };
                let rc = setsockopt(
                    sock,
                    SOL_PACKET,
                    PACKET_RX_RING,
                    &req as *const TpacketReq3 as *const c_void,
                    mem::size_of::<TpacketReq3>() as socklen_t,
                );
                if rc < 0 {
                    close(sock);
                    return Err(io::Error::last_os_error());
                }

                // ── Step 4: mmap the ring ────────────────────────────────────
                let ring_size = cfg.block_size * cfg.block_nr;
                let ring = mmap(
                    std::ptr::null_mut(),
                    ring_size,
                    PROT_READ | PROT_WRITE,
                    MAP_SHARED,
                    sock,
                    0,
                );
                if ring == libc::MAP_FAILED {
                    close(sock);
                    return Err(io::Error::last_os_error());
                }

                // ── Step 5: bind to interface ────────────────────────────────
                let c_iface = CString::new(iface).map_err(|_| {
                    io::Error::new(io::ErrorKind::InvalidInput, "interface name contains NUL")
                })?;
                let ifindex = if_nametoindex(c_iface.as_ptr());
                if ifindex == 0 {
                    munmap(ring, ring_size);
                    close(sock);
                    return Err(io::Error::new(
                        io::ErrorKind::NotFound,
                        format!("interface '{}' not found (if_nametoindex=0)", iface),
                    ));
                }

                let sa = SockaddrLl {
                    sll_family: AF_PACKET as u16,
                    sll_protocol: (ETH_P_ALL as u16).to_be(),
                    sll_ifindex: ifindex as i32,
                    sll_hatype: 0,
                    sll_pkttype: 0,
                    sll_halen: 0,
                    sll_addr: [0u8; 8],
                };
                let rc = bind(
                    sock,
                    &sa as *const SockaddrLl as *const libc::sockaddr,
                    mem::size_of::<SockaddrLl>() as socklen_t,
                );
                if rc < 0 {
                    munmap(ring, ring_size);
                    close(sock);
                    return Err(io::Error::last_os_error());
                }

                Ok(Self {
                    sock,
                    ring: ring as *mut u8,
                    ring_size,
                    block_size: cfg.block_size,
                    block_nr: cfg.block_nr,
                    frame_size: cfg.frame_size,
                    current_block: 0,
                    current_pkt: 0,
                    frames_captured: 0,
                })
            }
        }

        /// Return the next captured packet as a zero-copy slice into mmap memory.
        ///
        /// Polls with a 100 ms timeout when no block is currently user-owned.
        /// Returns `None` on timeout, EINTR, or ring temporarily empty.
        ///
        /// The returned slice is valid only until the *next* call to `next_packet()`.
        /// The caller must copy data out if it needs to survive beyond that point.
        pub fn next_packet(&mut self) -> Option<&[u8]> {
            unsafe {
                loop {
                    let block_base = self.ring.add(self.current_block * self.block_size);
                    let bd = &*(block_base as *const TpacketBlockDesc);

                    // If the kernel still owns this block, poll for up to 100 ms
                    if bd.hdr.block_status & TP_STATUS_USER == 0 {
                        let mut pfd = pollfd {
                            fd: self.sock,
                            events: POLLIN,
                            revents: 0,
                        };
                        let ready = poll(&mut pfd as *mut pollfd, 1, 100);
                        if ready <= 0 {
                            return None;
                        }
                        if bd.hdr.block_status & TP_STATUS_USER == 0 {
                            return None;
                        }
                    }

                    let num_pkts = bd.hdr.num_pkts;
                    if self.current_pkt < num_pkts {
                        // Walk the singly-linked frame chain inside this block
                        let mut off = bd.hdr.offset_to_first_pkt as usize;
                        for _ in 0..self.current_pkt {
                            let hdr = &*(block_base.add(off) as *const Tpacket3Hdr);
                            if hdr.tp_next_offset == 0 {
                                break;
                            }
                            off = off.saturating_add(hdr.tp_next_offset as usize);
                        }

                        let frame_hdr = &*(block_base.add(off) as *const Tpacket3Hdr);
                        let mac_off = off + frame_hdr.tp_mac as usize;
                        let snaplen = frame_hdr.tp_snaplen as usize;

                        // Bounds-check before exposing a slice
                        if mac_off.saturating_add(snaplen) > self.block_size {
                            // Kernel wrote a corrupt frame — abandon this block
                            self.release_block(block_base);
                            self.current_pkt = 0;
                            self.current_block = (self.current_block + 1) % self.block_nr;
                            continue;
                        }

                        self.current_pkt += 1;
                        self.frames_captured += 1;
                        return Some(std::slice::from_raw_parts(
                            block_base.add(mac_off),
                            snaplen,
                        ));
                    }

                    // All frames in this block consumed — retire it back to kernel
                    self.release_block(block_base);
                    self.current_pkt = 0;
                    self.current_block = (self.current_block + 1) % self.block_nr;
                }
            }
        }

        /// Hand a consumed block back to the kernel (set `TP_STATUS_KERNEL`).
        ///
        /// A full memory barrier ensures the kernel does not observe a partial write.
        unsafe fn release_block(&self, block_base: *mut u8) {
            std::sync::atomic::fence(std::sync::atomic::Ordering::SeqCst);
            let bd = &mut *(block_base as *mut TpacketBlockDesc);
            bd.hdr.block_status = TP_STATUS_KERNEL;
        }

        /// Close the socket and unmap ring memory.
        pub fn close(&mut self) {
            if !self.ring.is_null() {
                unsafe { munmap(self.ring as *mut c_void, self.ring_size) };
                self.ring = std::ptr::null_mut();
            }
            if self.sock >= 0 {
                unsafe { close(self.sock) };
                self.sock = -1;
            }
        }

        /// Total size of the mmap'd ring in bytes.
        pub fn ring_size(&self) -> usize { self.ring_size }
        /// Number of blocks in the ring.
        pub fn block_nr(&self) -> usize { self.block_nr }
        /// Configured per-block size.
        pub fn block_size(&self) -> usize { self.block_size }
    }

    impl Drop for AfPacketRing {
        fn drop(&mut self) { self.close(); }
    }
} // mod inner

// re-export Linux types at crate level
#[cfg(target_os = "linux")]
pub use inner::{AfPacketRing, RingConfig};

// ─────────────────────────────────────────────────────────────────────────────
// Non-Linux stubs
// ─────────────────────────────────────────────────────────────────────────────

/// Placeholder for non-Linux platforms.
#[cfg(not(target_os = "linux"))]
pub struct AfPacketRing;

#[cfg(not(target_os = "linux"))]
impl AfPacketRing {
    pub fn new(_iface: &str) -> std::io::Result<Self> {
        Err(std::io::Error::new(
            std::io::ErrorKind::Unsupported,
            "AF_PACKET ring buffer is Linux-only",
        ))
    }
    pub fn next_packet(&mut self) -> Option<&[u8]> { None }
    pub fn close(&mut self) {}
    pub fn ring_size(&self) -> usize { 0 }
    pub fn block_nr(&self) -> usize { 0 }
    pub fn block_size(&self) -> usize { 0 }
    pub fn frames_captured(&self) -> u64 { 0 }
}

/// Placeholder `RingConfig` for non-Linux builds.
#[cfg(not(target_os = "linux"))]
#[derive(Debug, Clone)]
pub struct RingConfig {
    pub block_size: usize,
    pub block_nr: usize,
    pub frame_size: usize,
    pub retire_blk_tov: u32,
}

#[cfg(not(target_os = "linux"))]
impl Default for RingConfig {
    fn default() -> Self {
        Self {
            block_size: 1 << 22,
            block_nr: 64,
            frame_size: 2048,
            retire_blk_tov: 60,
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ring_config_defaults_match_spec() {
        let cfg = RingConfig::default();
        assert_eq!(cfg.block_size, 1 << 22);
        assert_eq!(cfg.block_nr, 64);
        assert_eq!(cfg.frame_size, 2048);
        assert_eq!(cfg.retire_blk_tov, 60);
    }

    #[test]
    fn ring_total_size() {
        let cfg = RingConfig::default();
        assert_eq!(cfg.block_size * cfg.block_nr, 256 * 1024 * 1024);
    }

    #[test]
    #[cfg(not(target_os = "linux"))]
    fn non_linux_returns_unsupported() {
        let r = AfPacketRing::new("eth0");
        assert!(r.is_err());
        assert_eq!(r.unwrap_err().kind(), std::io::ErrorKind::Unsupported);
    }

    #[test]
    #[cfg(not(target_os = "linux"))]
    fn non_linux_stub_methods() {
        let mut ring = AfPacketRing;
        assert_eq!(ring.ring_size(), 0);
        assert_eq!(ring.block_nr(), 0);
        assert!(ring.next_packet().is_none());
        ring.close(); // must not panic
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn linux_bad_iface_returns_not_found() {
        let r = AfPacketRing::new("no_such_iface_99");
        assert!(r.is_err());
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn linux_bad_block_size_non_power_of_two() {
        use inner::RingConfig;
        let cfg = RingConfig {
            block_size: 3000,
            ..Default::default()
        };
        let r = AfPacketRing::with_config("lo", cfg);
        assert!(r.is_err());
        assert!(r.unwrap_err().to_string().contains("power of two"));
    }
}
