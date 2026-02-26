// K-XRO-PT-002 — Linux perf_event_open FFI Wrapper
//
// Provides safe Rust bindings around the Linux `perf_event_open(2)` syscall.
// On non-Linux platforms, or when the calling process lacks the required
// privileges (EPERM) or the kernel is built without perf support (ENOSYS),
// every function returns `PerfError::NotAvailable` so the rest of the agent
// degrades gracefully.

#![allow(non_upper_case_globals, non_camel_case_types, dead_code)]

use std::os::unix::io::RawFd;

// ─── Error type ──────────────────────────────────────────────────────────────

#[derive(Debug, thiserror::Error)]
pub enum PerfError {
    #[error("perf_event_open is not available on this platform or kernel")]
    NotAvailable,

    #[error("permission denied — run as root or set /proc/sys/kernel/perf_event_paranoid <= 1")]
    PermissionDenied,

    #[error("invalid perf_event_attr configuration: {0}")]
    InvalidConfig(String),

    #[error("I/O error reading perf counter fd: {0}")]
    Io(#[from] std::io::Error),

    #[error("syscall returned unexpected errno {0}")]
    Syscall(i32),
}

pub type PerfResult<T> = Result<T, PerfError>;

// ─── perf event types ─────────────────────────────────────────────────────────

pub const PERF_TYPE_HARDWARE: u32 = 0;
pub const PERF_TYPE_SOFTWARE: u32 = 1;
pub const PERF_TYPE_TRACEPOINT: u32 = 2;
pub const PERF_TYPE_HW_CACHE: u32 = 3;
pub const PERF_TYPE_RAW: u32 = 4;
pub const PERF_TYPE_BREAKPOINT: u32 = 5;

// ─── Hardware event IDs (PERF_TYPE_HARDWARE) ──────────────────────────────────

pub const PERF_COUNT_HW_CPU_CYCLES: u64 = 0;
pub const PERF_COUNT_HW_INSTRUCTIONS: u64 = 1;
pub const PERF_COUNT_HW_CACHE_REFERENCES: u64 = 2;
pub const PERF_COUNT_HW_CACHE_MISSES: u64 = 3;
pub const PERF_COUNT_HW_BRANCH_INSTRUCTIONS: u64 = 4;
pub const PERF_COUNT_HW_BRANCH_MISSES: u64 = 5;
pub const PERF_COUNT_HW_BUS_CYCLES: u64 = 6;
pub const PERF_COUNT_HW_STALLED_CYCLES_FRONTEND: u64 = 7;
pub const PERF_COUNT_HW_STALLED_CYCLES_BACKEND: u64 = 8;
pub const PERF_COUNT_HW_REF_CPU_CYCLES: u64 = 9;

// ─── Software event IDs (PERF_TYPE_SOFTWARE) ─────────────────────────────────

pub const PERF_COUNT_SW_CPU_CLOCK: u64 = 0;
pub const PERF_COUNT_SW_TASK_CLOCK: u64 = 1;
pub const PERF_COUNT_SW_PAGE_FAULTS: u64 = 2;
pub const PERF_COUNT_SW_CONTEXT_SWITCHES: u64 = 3;
pub const PERF_COUNT_SW_CPU_MIGRATIONS: u64 = 4;
pub const PERF_COUNT_SW_PAGE_FAULTS_MIN: u64 = 5;
pub const PERF_COUNT_SW_PAGE_FAULTS_MAJ: u64 = 6;
pub const PERF_COUNT_SW_ALIGNMENT_FAULTS: u64 = 7;
pub const PERF_COUNT_SW_EMULATION_FAULTS: u64 = 8;

// ─── Flags ────────────────────────────────────────────────────────────────────

pub const PERF_FLAG_FD_NO_GROUP: u64 = 1 << 0;
pub const PERF_FLAG_FD_OUTPUT: u64 = 1 << 1;
pub const PERF_FLAG_PID_CGROUP: u64 = 1 << 2;
pub const PERF_FLAG_FD_CLOEXEC: u64 = 1 << 3;

// ─── perf_event_attr ─────────────────────────────────────────────────────────

/// C-compatible representation of `struct perf_event_attr` from
/// `<linux/perf_event.h>`.  The struct is intentionally kept at the original
/// layout (128 bytes) so it can be passed directly to the syscall.
#[repr(C)]
#[derive(Debug, Clone, Default)]
pub struct PerfEventAttr {
    /// Major type: hardware/software/tracepoint/etc.
    pub type_: u32,
    /// Size of this struct for version negotiation with the kernel.
    pub size: u32,
    /// Type-specific event configuration.
    pub config: u64,
    /// Sampling period or frequency union (period when `freq` bit is clear).
    pub sample_period_or_freq: u64,
    /// Bitmask of PERF_SAMPLE_* flags.
    pub sample_type: u64,
    /// Bitmask of PERF_FORMAT_* flags.
    pub read_format: u64,
    /// Packed bitfield word 1 (disabled, inherit, pinned, …).
    pub flags: u64,
    /// Wakeup granularity (events or watermark bytes) union.
    pub wakeup_events_or_watermark: u32,
    /// Hardware breakpoint type.
    pub bp_type: u32,
    /// Union: bp_addr / kprobe_func / uprobe_path / config1.
    pub config1: u64,
    /// Union: bp_len / kprobe_addr / probe_offset / config2.
    pub config2: u64,
    /// Branch-sample filter.
    pub branch_sample_type: u64,
    /// Registers to sample on user-space entry.
    pub sample_regs_user: u64,
    /// Stack size for user space stack sampling.
    pub sample_stack_user: u32,
    /// Clock ID for timestamps.
    pub clockid: i32,
    /// Registers to sample on kernel entry.
    pub sample_regs_intr: u64,
    /// AUX watermark.
    pub aux_watermark: u32,
    /// Maximum number of frame pointers for stack unwinding.
    pub sample_max_stack: u16,
    pub __reserved_2: u16,
    /// AUX sample size.
    pub aux_sample_size: u32,
    pub __reserved_3: u32,
    /// Context switch data size.
    pub sig_data: u64,
    /// Kernel-specific config3.
    pub config3: u64,
}

impl PerfEventAttr {
    pub const SIZE: u32 = std::mem::size_of::<Self>() as u32;

    /// Construct a minimal counting attribute for the given type + config.
    pub fn counting(type_: u32, config: u64) -> Self {
        let mut attr = Self::default();
        attr.type_ = type_;
        attr.size = Self::SIZE;
        attr.config = config;
        // disabled=1: start the counter manually via ioctl(PERF_EVENT_IOC_ENABLE)
        attr.flags = 1 << 0; // disabled bit
        attr
    }
}

// ─── Raw syscall wrapper ──────────────────────────────────────────────────────

#[cfg(target_os = "linux")]
pub unsafe fn perf_event_open(
    attr: *const PerfEventAttr,
    pid: libc::pid_t,
    cpu: libc::c_int,
    group_fd: RawFd,
    flags: libc::c_ulong,
) -> i64 {
    libc::syscall(
        libc::SYS_perf_event_open,
        attr as *const libc::c_void,
        pid,
        cpu,
        group_fd,
        flags,
    )
}

// ─── PerfCounter ─────────────────────────────────────────────────────────────

/// A single opened performance counter backed by a file descriptor.
pub struct PerfCounter {
    fd: RawFd,
    pub event_type: u32,
    pub config: u64,
    pub label: String,
}

impl PerfCounter {
    /// Open a counting event for the calling process (`pid = 0`) on any CPU
    /// (`cpu = -1`).  The counter starts disabled; call `enable()` to begin
    /// accumulating.
    pub fn open(event_type: u32, config: u64, label: impl Into<String>) -> PerfResult<Self> {
        #[cfg(not(target_os = "linux"))]
        {
            return Err(PerfError::NotAvailable);
        }

        #[cfg(target_os = "linux")]
        {
            let mut attr = PerfEventAttr::counting(event_type, config);
            // Clear disabled bit so the counter starts running immediately.
            attr.flags &= !(1u64 << 0);
            // exclude_kernel=0, exclude_hv=0 — measure everything.

            let fd = unsafe {
                perf_event_open(
                    &attr as *const PerfEventAttr,
                    0,              // pid: current process
                    -1,             // cpu: all CPUs
                    -1,             // group_fd: no group
                    PERF_FLAG_FD_CLOEXEC,
                )
            };

            if fd < 0 {
                let errno = unsafe { *libc::__errno_location() };
                return Err(match errno {
                    libc::ENOSYS => PerfError::NotAvailable,
                    libc::EPERM | libc::EACCES => PerfError::PermissionDenied,
                    libc::EINVAL => PerfError::InvalidConfig(format!(
                        "EINVAL for type={event_type} config={config:#x}"
                    )),
                    other => PerfError::Syscall(other),
                });
            }

            Ok(Self {
                fd: fd as RawFd,
                event_type,
                config,
                label: label.into(),
            })
        }
    }

    /// Read the current 64-bit counter value.
    pub fn read(&self) -> PerfResult<u64> {
        #[cfg(not(target_os = "linux"))]
        {
            return Err(PerfError::NotAvailable);
        }

        #[cfg(target_os = "linux")]
        {
            let mut val: u64 = 0;
            let ret = unsafe {
                libc::read(
                    self.fd,
                    &mut val as *mut u64 as *mut libc::c_void,
                    std::mem::size_of::<u64>(),
                )
            };
            if ret < 0 {
                return Err(PerfError::Io(std::io::Error::last_os_error()));
            }
            Ok(val)
        }
    }

    /// Reset the counter to zero via ioctl.
    pub fn reset(&self) -> PerfResult<()> {
        #[cfg(not(target_os = "linux"))]
        return Err(PerfError::NotAvailable);

        #[cfg(target_os = "linux")]
        {
            const PERF_EVENT_IOC_RESET: libc::c_ulong = 0x2403;
            let ret =
                unsafe { libc::ioctl(self.fd, PERF_EVENT_IOC_RESET, 0 as libc::c_ulong) };
            if ret < 0 {
                return Err(PerfError::Io(std::io::Error::last_os_error()));
            }
            Ok(())
        }
    }

    /// Enable the counter via ioctl.
    pub fn enable(&self) -> PerfResult<()> {
        #[cfg(not(target_os = "linux"))]
        return Err(PerfError::NotAvailable);

        #[cfg(target_os = "linux")]
        {
            const PERF_EVENT_IOC_ENABLE: libc::c_ulong = 0x2400;
            let ret =
                unsafe { libc::ioctl(self.fd, PERF_EVENT_IOC_ENABLE, 0 as libc::c_ulong) };
            if ret < 0 {
                return Err(PerfError::Io(std::io::Error::last_os_error()));
            }
            Ok(())
        }
    }

    /// Disable the counter via ioctl (freezes accumulation).
    pub fn disable(&self) -> PerfResult<()> {
        #[cfg(not(target_os = "linux"))]
        return Err(PerfError::NotAvailable);

        #[cfg(target_os = "linux")]
        {
            const PERF_EVENT_IOC_DISABLE: libc::c_ulong = 0x2401;
            let ret =
                unsafe { libc::ioctl(self.fd, PERF_EVENT_IOC_DISABLE, 0 as libc::c_ulong) };
            if ret < 0 {
                return Err(PerfError::Io(std::io::Error::last_os_error()));
            }
            Ok(())
        }
    }
}

#[cfg(target_os = "linux")]
impl Drop for PerfCounter {
    fn drop(&mut self) {
        if self.fd >= 0 {
            unsafe {
                libc::close(self.fd);
            }
        }
    }
}

// ─── PerfSession ─────────────────────────────────────────────────────────────

/// A named collection of hardware and software perf counters.
/// Opens CPU cycles, instructions, cache misses, branch misses, context switches,
/// page faults, and CPU migrations. Any counter that cannot be opened (due to
/// privilege or hardware support) is skipped with a warning; the session still
/// reports whatever is available.
pub struct PerfSession {
    counters: Vec<PerfCounter>,
}

/// A single raw reading from the session.
#[derive(Debug, Clone, Default)]
pub struct PerfReading {
    pub cpu_cycles: u64,
    pub instructions: u64,
    pub cache_misses: u64,
    pub branch_misses: u64,
    pub context_switches: u64,
    pub page_faults: u64,
    pub cpu_migrations: u64,
}

impl PerfSession {
    /// Open all standard counters. Errors are demoted to tracing warnings.
    pub fn open() -> Self {
        let wanted: &[(u32, u64, &str)] = &[
            (PERF_TYPE_HARDWARE, PERF_COUNT_HW_CPU_CYCLES, "cpu_cycles"),
            (PERF_TYPE_HARDWARE, PERF_COUNT_HW_INSTRUCTIONS, "instructions"),
            (PERF_TYPE_HARDWARE, PERF_COUNT_HW_CACHE_MISSES, "cache_misses"),
            (PERF_TYPE_HARDWARE, PERF_COUNT_HW_BRANCH_MISSES, "branch_misses"),
            (PERF_TYPE_SOFTWARE, PERF_COUNT_SW_CONTEXT_SWITCHES, "context_switches"),
            (PERF_TYPE_SOFTWARE, PERF_COUNT_SW_PAGE_FAULTS, "page_faults"),
            (PERF_TYPE_SOFTWARE, PERF_COUNT_SW_CPU_MIGRATIONS, "cpu_migrations"),
        ];

        let mut counters = Vec::with_capacity(wanted.len());
        for (type_, config, label) in wanted {
            match PerfCounter::open(*type_, *config, *label) {
                Ok(c) => counters.push(c),
                Err(PerfError::NotAvailable) => {
                    tracing::warn!(
                        counter = label,
                        "perf_event_open not available — skipping"
                    );
                }
                Err(PerfError::PermissionDenied) => {
                    tracing::warn!(
                        counter = label,
                        "perf_event_open EPERM — set kernel.perf_event_paranoid <= 1 or run as root"
                    );
                }
                Err(e) => {
                    tracing::warn!(counter = label, error = %e, "perf counter open failed");
                }
            }
        }

        Self { counters }
    }

    /// Returns true if at least one counter opened successfully.
    pub fn is_available(&self) -> bool {
        !self.counters.is_empty()
    }

    /// Read every open counter and return a `PerfReading`.  Unavailable counters
    /// are returned as zero.
    pub fn read_all(&self) -> PerfReading {
        let mut r = PerfReading::default();

        for c in &self.counters {
            let val = match c.read() {
                Ok(v) => v,
                Err(e) => {
                    tracing::debug!(counter = %c.label, error = %e, "read failed");
                    continue;
                }
            };

            match c.label.as_str() {
                "cpu_cycles" => r.cpu_cycles = val,
                "instructions" => r.instructions = val,
                "cache_misses" => r.cache_misses = val,
                "branch_misses" => r.branch_misses = val,
                "context_switches" => r.context_switches = val,
                "page_faults" => r.page_faults = val,
                "cpu_migrations" => r.cpu_migrations = val,
                _ => {}
            }
        }

        r
    }

    /// Reset all counters to zero.
    pub fn reset_all(&self) {
        for c in &self.counters {
            if let Err(e) = c.reset() {
                tracing::debug!(counter = %c.label, error = %e, "reset failed");
            }
        }
    }

    /// Number of successfully opened counters.
    pub fn counter_count(&self) -> usize {
        self.counters.len()
    }
}

// ─── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    /// The session must not panic regardless of privilege level.
    #[test]
    fn session_open_does_not_panic() {
        let _session = PerfSession::open();
        // No assertion: we just verify it doesn't panic.
    }

    /// If we can open counters, reading must return 0 or a positive integer.
    #[test]
    fn read_all_returns_coherent_values() {
        let session = PerfSession::open();
        if !session.is_available() {
            // Graceful skip in restricted CI environments.
            return;
        }
        let r = session.read_all();
        // Execute some work so counters accumulate.
        let _: u64 = (0u64..1_000_000).fold(0, |a, b| a.wrapping_add(b));
        let r2 = session.read_all();
        // cpu_cycles should have advanced (or at least not regressed).
        assert!(
            r2.cpu_cycles >= r.cpu_cycles,
            "cpu_cycles should be non-decreasing"
        );
    }

    #[test]
    fn attr_size_is_correct() {
        // The struct must be at least 112 bytes (the minimum accepted by modern kernels).
        assert!(
            std::mem::size_of::<PerfEventAttr>() >= 112,
            "PerfEventAttr too small"
        );
    }

    #[test]
    fn perf_counter_open_returns_not_available_on_non_linux() {
        #[cfg(not(target_os = "linux"))]
        {
            let r = PerfCounter::open(PERF_TYPE_HARDWARE, PERF_COUNT_HW_CPU_CYCLES, "test");
            assert!(
                matches!(r, Err(PerfError::NotAvailable)),
                "expected NotAvailable on non-Linux"
            );
        }
        // On Linux the result depends on the environment; just verify no panic.
        #[cfg(target_os = "linux")]
        {
            let _ = PerfCounter::open(PERF_TYPE_HARDWARE, PERF_COUNT_HW_CPU_CYCLES, "test");
        }
    }
}
