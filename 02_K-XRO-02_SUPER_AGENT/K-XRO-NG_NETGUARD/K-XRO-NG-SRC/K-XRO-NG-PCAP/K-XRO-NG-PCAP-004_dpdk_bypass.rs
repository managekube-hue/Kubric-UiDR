//! K-XRO-NG-PCAP-004 — DPDK kernel-bypass capture stub with feature detection.
//!
//! # DPDK Architecture
//!
//! DPDK (Data Plane Development Kit) enables userspace network drivers that
//! completely bypass the Linux kernel network stack, achieving wire-rate
//! throughput on 10/25/100G NICs.
//!
//! ## PMD (Poll-Mode Driver)
//! Instead of relying on hardware interrupts, DPDK PMDs poll the NIC descriptor
//! ring from a dedicated userspace thread.  This eliminates context-switch and
//! interrupt-handling latency, enabling millions of packets per second per core.
//!
//! ## EAL Initialization
//! The DPDK Environment Abstraction Layer (`rte_eal_init`) must be called once
//! per process.  It:
//! - Reserves hugepage memory (2 MiB or 1 GiB pages via `/dev/hugepages`)
//! - Probes and binds PCI NIC devices via the UIO/VFIO kernel modules
//! - Spawns lcore (logical core) worker threads at the requested affinity
//!
//! ## Memory Pools (mempools)
//! Packet buffers (`rte_mbuf`) live in `rte_pktmbuf_pool` — a hugepage-backed,
//! lock-free ring allocator.  The NIC DMA-writes directly into mbuf data rooms,
//! so `rte_eth_rx_burst()` returns pointers into the original DMA buffer with
//! zero extra copies.
//!
//! ## Feature Flag
//! Compile with `--features dpdk` to enable the real FFI bindings.
//! Without it, every call returns `DpdkError::NotAvailable` gracefully, and
//! NetGuard falls back to AF_PACKET or libpcap capture.
//!
//! ## Hugepage setup (host requirement)
//! ```bash
//! echo 512 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
//! mkdir -p /dev/hugepages && mount -t hugetlbfs nodev /dev/hugepages
//! ```

#![allow(dead_code)]

use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::mpsc::Sender;
use tracing::{info, warn};

// ── Error types ───────────────────────────────────────────────────────────────

/// Errors produced by the DPDK capture layer.
#[derive(Debug, Error)]
pub enum DpdkError {
    /// Binary compiled without `--features dpdk`.
    #[error("DPDK not available: recompile with --features dpdk")]
    NotAvailable,

    /// `rte_eal_init()` returned a negative value.
    #[error("DPDK EAL init failed (rc={0}): check hugepages and PCI bindings")]
    EalInitFailed(i32),

    /// The requested port ID is not bound to a DPDK PMD.
    #[error("DPDK port {0} not found — bind NIC with dpdk-devbind.py")]
    PortNotFound(u16),

    /// `rte_pktmbuf_pool_create` returned NULL.
    #[error("DPDK mempool creation failed: check hugepage availability ({0})")]
    MempoolFailed(String),

    /// `rte_eth_dev_configure` or `rte_eth_dev_start` failed.
    #[error("DPDK port {port} operation failed (rc={rc})")]
    PortOpFailed { port: u16, rc: i32 },

    /// `rte_eth_rx_queue_setup` failed.
    #[error("DPDK RX queue setup failed: port={port} queue={queue}")]
    RxQueueFailed { port: u16, queue: u16 },

    /// Capture loop was already started.
    #[error("DPDK capture already running — call stop() first")]
    AlreadyRunning,
}

// ── Configuration ─────────────────────────────────────────────────────────────

/// DPDK port and capture configuration.
#[derive(Debug, Clone)]
pub struct DpdkConfig {
    /// DPDK port ID (corresponds to detected PCI device).  Default: 0.
    pub port_id: u16,

    /// Number of RX hardware queues.  Typically 1 per lcore.  Default: 1.
    pub rx_queues: u16,

    /// Packets drained per `rte_eth_rx_burst()` call.  Default: 32.
    pub burst_size: u32,

    /// Mbuf pool capacity.  Must be >= `rx_queues * rx_desc * 2`.  Default: 8192.
    pub nb_mbufs: u32,

    /// Additional EAL arguments passed verbatim to `rte_eal_init()`.
    ///
    /// Example: `vec!["--lcores".into(), "0-3".into(), "-a".into(), "0000:01:00.0".into()]`
    pub eal_args: Vec<String>,
}

impl Default for DpdkConfig {
    fn default() -> Self {
        Self {
            port_id: 0,
            rx_queues: 1,
            burst_size: 32,
            nb_mbufs: 8192,
            eal_args: Vec::new(),
        }
    }
}

// ── DpdkCapture ───────────────────────────────────────────────────────────────

/// DPDK kernel-bypass packet capture handle.
///
/// When compiled with `--features dpdk` this struct owns the DPDK EAL and the
/// configured port.  Without that feature it acts as a graceful no-op.
pub struct DpdkCapture {
    port_id: u16,
    rx_queues: u16,
    burst_size: u32,
    /// Set to `true` while the RX poll loop is active.
    running: Arc<AtomicBool>,
    /// Total packets received since `start_rx_loop()`.
    pub packets_rx: Arc<AtomicU64>,
    /// Total bytes received since `start_rx_loop()`.
    pub bytes_rx: Arc<AtomicU64>,
}

// ── DPDK-enabled implementation ───────────────────────────────────────────────

#[cfg(feature = "dpdk")]
mod dpdk_impl {
    //! Real DPDK FFI.  Only compiled when `--features dpdk` is active.
    //!
    //! `build.rs` is expected to emit `cargo:rustc-link-lib=rte_eal` etc.
    //! using `pkg-config --libs libdpdk`.

    use super::*;
    use std::ffi::CString;
    use std::os::raw::{c_char, c_int, c_uint, c_void};

    // ── Opaque DPDK types ────────────────────────────────────────────────────

    /// `rte_mbuf` — opaque packet buffer type.
    #[repr(C)]
    pub struct RteMbuf {
        _opaque: [u64; 16], // sizeof(rte_mbuf) ≈ 128 bytes on x86-64
    }

    /// `rte_mempool` — opaque mempool handle.
    #[repr(C)]
    pub struct RteMempool {
        _opaque: [u64; 1],
    }

    // ── Minimal DPDK FFI declarations ────────────────────────────────────────

    extern "C" {
        /// Initialise the DPDK EAL.  Must be called once before any other DPDK API.
        fn rte_eal_init(argc: c_int, argv: *mut *mut c_char) -> c_int;

        /// Clean up EAL resources.  Call on graceful shutdown.
        fn rte_eal_cleanup() -> c_int;

        /// Create a hugepage-backed mbuf pool.
        fn rte_pktmbuf_pool_create(
            name: *const c_char,
            n: c_uint,
            cache_size: c_uint,
            priv_size: u16,
            data_room_size: u16,
            socket_id: c_int,
        ) -> *mut RteMempool;

        /// Returns the number of available Ethernet ports bound to DPDK PMDs.
        fn rte_eth_dev_count_avail() -> u16;

        /// Configure an Ethernet device.
        fn rte_eth_dev_configure(
            port_id: u16,
            nb_rx_queue: u16,
            nb_tx_queue: u16,
            eth_conf: *const c_void,
        ) -> c_int;

        /// Set up one RX queue on a port.
        fn rte_eth_rx_queue_setup(
            port_id: u16,
            rx_queue_id: u16,
            nb_rx_desc: u16,
            socket_id: c_uint,
            rx_conf: *const c_void,
            mb_pool: *mut RteMempool,
        ) -> c_int;

        /// Start transmitting and receiving on a port.
        fn rte_eth_dev_start(port_id: u16) -> c_int;

        /// Stop a port.
        fn rte_eth_dev_stop(port_id: u16) -> c_int;

        /// Burst-receive up to `nb_pkts` packets from an RX queue.
        fn rte_eth_rx_burst(
            port_id: u16,
            queue_id: u16,
            rx_pkts: *mut *mut RteMbuf,
            nb_pkts: u16,
        ) -> u16;

        /// Free a single mbuf back to its mempool.
        fn rte_pktmbuf_free(m: *mut RteMbuf);

        /// Return a pointer to `offset` bytes after the start of mbuf data.
        fn rte_pktmbuf_mtod_offset(m: *const RteMbuf, offset: u16) -> *const u8;

        /// Return the number of bytes of data in an mbuf.
        fn rte_pktmbuf_data_len(m: *const RteMbuf) -> u16;
    }

    impl DpdkCapture {
        /// Initialise the DPDK EAL and configure `config.port_id`.
        pub fn try_init(config: DpdkConfig) -> Result<Self, DpdkError> {
            // Build argc/argv for rte_eal_init
            let prog = CString::new("kubric-netguard").unwrap();
            let mut cstrings: Vec<CString> = std::iter::once(prog)
                .chain(
                    config
                        .eal_args
                        .iter()
                        .filter_map(|s| CString::new(s.as_str()).ok()),
                )
                .collect();
            let mut argv: Vec<*mut c_char> =
                cstrings.iter_mut().map(|s| s.as_ptr() as *mut c_char).collect();

            let rc = unsafe { rte_eal_init(argv.len() as c_int, argv.as_mut_ptr()) };
            if rc < 0 {
                return Err(DpdkError::EalInitFailed(rc));
            }

            // Validate port
            let nb_ports = unsafe { rte_eth_dev_count_avail() };
            if config.port_id >= nb_ports {
                return Err(DpdkError::PortNotFound(config.port_id));
            }

            // Create mempool (~128 B per mbuf entry)
            let pool_name = CString::new("kubric_mbuf_pool").unwrap();
            let pool = unsafe {
                rte_pktmbuf_pool_create(
                    pool_name.as_ptr(),
                    config.nb_mbufs,
                    256,  // per-lcore cache size
                    0,    // private data size
                    2176, // RTE_MBUF_DEFAULT_BUF_SIZE (2048+128 headroom)
                    0,    // NUMA socket 0
                )
            };
            if pool.is_null() {
                return Err(DpdkError::MempoolFailed(
                    "insufficient hugepages or nb_mbufs too large".into(),
                ));
            }

            // Configure port (no TX queues needed — RX-only)
            let rc = unsafe {
                rte_eth_dev_configure(config.port_id, config.rx_queues, 0, std::ptr::null())
            };
            if rc != 0 {
                return Err(DpdkError::PortOpFailed { port: config.port_id, rc });
            }

            // Set up each RX queue
            for q in 0..config.rx_queues {
                let rc = unsafe {
                    rte_eth_rx_queue_setup(
                        config.port_id,
                        q,
                        512, // RX descriptor ring size
                        0,   // socket 0
                        std::ptr::null(),
                        pool,
                    )
                };
                if rc != 0 {
                    return Err(DpdkError::RxQueueFailed {
                        port: config.port_id,
                        queue: q,
                    });
                }
            }

            // Start port
            let rc = unsafe { rte_eth_dev_start(config.port_id) };
            if rc != 0 {
                return Err(DpdkError::PortOpFailed { port: config.port_id, rc });
            }

            info!(
                port_id = config.port_id,
                rx_queues = config.rx_queues,
                burst_size = config.burst_size,
                "DPDK port started"
            );

            Ok(Self {
                port_id: config.port_id,
                rx_queues: config.rx_queues,
                burst_size: config.burst_size,
                running: Arc::new(AtomicBool::new(false)),
                packets_rx: Arc::new(AtomicU64::new(0)),
                bytes_rx: Arc::new(AtomicU64::new(0)),
            })
        }

        /// Launch the DPDK RX burst loop on a dedicated OS thread.
        ///
        /// The thread polls every RX queue in a tight loop — this is intentional.
        /// DPDK PMDs must not yield to the scheduler; busy-polling is the source
        /// of their performance advantage.
        ///
        /// Raw packet bytes are cloned from the mbuf and sent over `tx`.  If the
        /// channel is full the packet is dropped (backpressure avoidance).
        pub fn start_rx_loop(&self, tx: Sender<Vec<u8>>) {
            if self.running.swap(true, Ordering::SeqCst) {
                warn!("DPDK RX loop already running");
                return;
            }

            let running = Arc::clone(&self.running);
            let pkt_ctr = Arc::clone(&self.packets_rx);
            let byte_ctr = Arc::clone(&self.bytes_rx);
            let port_id = self.port_id;
            let rx_queues = self.rx_queues;
            let burst_size = self.burst_size;

            std::thread::Builder::new()
                .name(format!("dpdk-rx-port{port_id}"))
                .spawn(move || {
                    let mut burst: Vec<*mut RteMbuf> =
                        vec![std::ptr::null_mut(); burst_size as usize];

                    info!(port = port_id, "DPDK RX burst loop started");

                    while running.load(Ordering::Relaxed) {
                        for q in 0..rx_queues {
                            let nb_rx = unsafe {
                                rte_eth_rx_burst(
                                    port_id,
                                    q,
                                    burst.as_mut_ptr(),
                                    burst_size as u16,
                                )
                            } as usize;

                            for i in 0..nb_rx {
                                let mbuf = burst[i];
                                let len = unsafe { rte_pktmbuf_data_len(mbuf) } as usize;
                                let ptr = unsafe { rte_pktmbuf_mtod_offset(mbuf, 0) };
                                let data =
                                    unsafe { std::slice::from_raw_parts(ptr, len).to_vec() };

                                pkt_ctr.fetch_add(1, Ordering::Relaxed);
                                byte_ctr.fetch_add(len as u64, Ordering::Relaxed);

                                // Non-blocking send — drops if channel is saturated
                                let _ = tx.try_send(data);

                                unsafe { rte_pktmbuf_free(mbuf) };
                            }
                        }
                    }

                    // Graceful port shutdown
                    unsafe { rte_eth_dev_stop(port_id) };
                    unsafe { rte_eal_cleanup() };
                    info!(port = port_id, "DPDK RX burst loop stopped");
                })
                .expect("failed to spawn DPDK RX thread");
        }

        /// Signal the RX loop to stop.  Returns immediately; the loop exits
        /// after the current burst iteration completes.
        pub fn stop(&self) {
            self.running.store(false, Ordering::SeqCst);
        }
    }
}

// ── Non-DPDK graceful stub ────────────────────────────────────────────────────

#[cfg(not(feature = "dpdk"))]
impl DpdkCapture {
    /// Returns `DpdkError::NotAvailable` — DPDK support not compiled in.
    pub fn try_init(_config: DpdkConfig) -> Result<Self, DpdkError> {
        Err(DpdkError::NotAvailable)
    }

    /// No-op.  DPDK not available.
    pub fn start_rx_loop(&self, _tx: Sender<Vec<u8>>) {
        warn!("DPDK not available — start_rx_loop is a no-op");
    }

    /// No-op.
    pub fn stop(&self) {
        self.running.store(false, Ordering::SeqCst);
    }
}

// ── Public helpers ─────────────────────────────────────────────────────────────

/// Returns `true` if this binary was compiled with `--features dpdk`.
pub fn is_dpdk_available() -> bool {
    cfg!(feature = "dpdk")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dpdk_config_defaults() {
        let cfg = DpdkConfig::default();
        assert_eq!(cfg.port_id, 0);
        assert_eq!(cfg.rx_queues, 1);
        assert_eq!(cfg.burst_size, 32);
        assert_eq!(cfg.nb_mbufs, 8192);
        assert!(cfg.eal_args.is_empty());
    }

    #[test]
    #[cfg(not(feature = "dpdk"))]
    fn try_init_returns_not_available_without_feature() {
        let result = DpdkCapture::try_init(DpdkConfig::default());
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), DpdkError::NotAvailable));
    }

    #[test]
    fn is_dpdk_available_consistent_with_feature() {
        #[cfg(feature = "dpdk")]
        assert!(is_dpdk_available());
        #[cfg(not(feature = "dpdk"))]
        assert!(!is_dpdk_available());
    }

    #[test]
    fn error_messages_are_descriptive() {
        let e = DpdkError::NotAvailable;
        assert!(e.to_string().contains("dpdk"));

        let e = DpdkError::PortNotFound(3);
        assert!(e.to_string().contains('3'));

        let e = DpdkError::EalInitFailed(-22);
        assert!(e.to_string().contains("-22"));

        let e = DpdkError::MempoolFailed("no hugepages".into());
        assert!(e.to_string().contains("no hugepages"));

        let e = DpdkError::PortOpFailed { port: 0, rc: -1 };
        assert!(e.to_string().contains("port"));

        let e = DpdkError::RxQueueFailed { port: 0, queue: 2 };
        assert!(e.to_string().contains("queue=2"));
    }

    #[test]
    fn already_running_error_message() {
        let e = DpdkError::AlreadyRunning;
        assert!(e.to_string().contains("stop()"));
    }
}
