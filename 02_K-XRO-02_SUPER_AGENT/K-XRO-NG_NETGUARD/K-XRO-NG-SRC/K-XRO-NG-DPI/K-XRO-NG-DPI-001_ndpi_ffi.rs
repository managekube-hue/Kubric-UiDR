//! K-XRO-NG-DPI-001 — nDPI FFI bindings for Layer-7 protocol classification.
//!
//! # License Boundary
//! nDPI is LGPL-3.0.  To maintain license isolation, this module loads
//! `libndpi.so` (Linux), `libndpi.dylib` (macOS), or `ndpi.dll` (Windows)
//! **at runtime** via `dlopen` / `LoadLibraryA`.  There is NO static link.
//!
//! If the shared library is absent at runtime the module logs a warning and
//! returns `None` from `NdpiLib::load()`.  All callers must handle this
//! gracefully — the L7 classifier falls back to heuristic port matching.
//!
//! # nDPI API (v4.x)
//! | Function                          | Purpose                              |
//! |-----------------------------------|--------------------------------------|
//! | `ndpi_init_detection_module`      | Allocate & initialise module handle  |
//! | `ndpi_set_protocol_detection_bitmask2` | Enable all protocols            |
//! | `ndpi_detection_process_packet`   | Classify one packet                  |
//! | `ndpi_get_proto_name`             | Protocol ID → human-readable name   |
//! | `ndpi_exit_detection_module`      | Free the module handle               |
//! | `ndpi_flow_malloc` / `ndpi_flow_free` | Per-flow struct lifecycle        |
//!
//! # Usage
//! ```rust,ignore
//! if let Some(lib) = NdpiLib::load() {
//!     let wrapper = NdpiWrapper::new(lib)?;
//!     let result = wrapper.classify(flow_ptr, pkt_data, pkt_len, tick, src, dst);
//!     println!("{:?}", result);
//! }
//! ```

#![allow(non_camel_case_types, non_snake_case, dead_code)]

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int, c_void};
use std::sync::atomic::{AtomicBool, Ordering};
use tracing::{info, warn};

// ── Opaque nDPI type aliases ──────────────────────────────────────────────────

/// Opaque detection module — allocated by nDPI, freed by `ndpi_exit_detection_module`.
pub type ndpi_detection_module_struct = c_void;

/// Opaque per-flow state — allocated by `ndpi_flow_malloc`.
pub type ndpi_flow_struct = c_void;

/// nDPI dual-layer protocol result.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default)]
pub struct ndpi_protocol {
    /// Encapsulating (outer) protocol ID.
    pub master_protocol: u16,
    /// Inner application protocol ID.
    pub app_protocol: u16,
    /// Application category bitmask.
    pub category: u32,
}

/// Protocol enable/disable bitmask (one bit per protocol ID, 512 protocols max).
#[repr(C)]
pub struct ndpi_protocol_bitmask {
    pub fds_bits: [u32; 16],
}

impl ndpi_protocol_bitmask {
    /// Create a bitmask with all protocols enabled.
    pub fn all_enabled() -> Self {
        Self {
            fds_bits: [u32::MAX; 16],
        }
    }
}

// ── Function pointer type aliases ─────────────────────────────────────────────

type FnInit =
    unsafe extern "C" fn(prefs: *const c_void) -> *mut ndpi_detection_module_struct;
type FnSetBitmask = unsafe extern "C" fn(
    module: *mut ndpi_detection_module_struct,
    bitmask: *const ndpi_protocol_bitmask,
);
type FnProcessPacket = unsafe extern "C" fn(
    module: *mut ndpi_detection_module_struct,
    flow: *mut ndpi_flow_struct,
    packet: *const u8,
    packetlen: u16,
    current_tick: u64,
    src: *mut c_void,
    dst: *mut c_void,
) -> ndpi_protocol;
type FnGetProtoName =
    unsafe extern "C" fn(module: *mut ndpi_detection_module_struct, proto: u16) -> *const c_char;
type FnExit = unsafe extern "C" fn(module: *mut ndpi_detection_module_struct);
type FnMallocFlow = unsafe extern "C" fn() -> *mut ndpi_flow_struct;
type FnFreeFlow = unsafe extern "C" fn(flow: *mut ndpi_flow_struct);

// ── NdpiLib — dynamically loaded library handle ───────────────────────────────

/// All nDPI function pointers, loaded at runtime.
pub struct NdpiLib {
    _handle: *mut c_void,
    pub init: FnInit,
    pub set_bitmask: FnSetBitmask,
    pub process_packet: FnProcessPacket,
    pub get_proto_name: FnGetProtoName,
    pub exit: FnExit,
    pub malloc_flow: FnMallocFlow,
    pub free_flow: FnFreeFlow,
}

// SAFETY: nDPI module handle is thread-safe per nDPI documentation.
unsafe impl Send for NdpiLib {}
unsafe impl Sync for NdpiLib {}

static NDPI_LOADED: AtomicBool = AtomicBool::new(false);

impl NdpiLib {
    /// Attempt to load the nDPI shared library.
    ///
    /// Library search order:
    /// 1. Name candidates specific to the OS (see below)
    /// 2. Path from `KUBRIC_NDPI_LIB` environment variable
    ///
    /// Returns `None` if loading fails on all candidates.
    pub fn load() -> Option<Self> {
        let candidates: &[&str] = if cfg!(target_os = "windows") {
            &["ndpi.dll", "libndpi.dll", "libndpi-4.dll"]
        } else if cfg!(target_os = "macos") {
            &["libndpi.dylib", "libndpi.4.dylib"]
        } else {
            // Linux / BSDs
            &["libndpi.so", "libndpi.so.4", "libndpi.so.4.0"]
        };

        for name in candidates {
            if let Some(lib) = Self::try_load_name(name) {
                NDPI_LOADED.store(true, Ordering::SeqCst);
                info!(library = name, "nDPI loaded successfully via dlopen");
                return Some(lib);
            }
        }

        // Fall back to user-supplied path
        if let Ok(path) = std::env::var("KUBRIC_NDPI_LIB") {
            if let Some(lib) = Self::try_load_name(&path) {
                NDPI_LOADED.store(true, Ordering::SeqCst);
                info!(library = path, "nDPI loaded from KUBRIC_NDPI_LIB");
                return Some(lib);
            }
        }

        warn!("nDPI shared library not found — L7 DPI disabled; using heuristics only");
        None
    }

    #[cfg(unix)]
    fn try_load_name(name: &str) -> Option<Self> {
        unsafe {
            let c_name = CString::new(name).ok()?;
            let handle = libc::dlopen(c_name.as_ptr(), libc::RTLD_LAZY | libc::RTLD_LOCAL);
            if handle.is_null() {
                return None;
            }

            macro_rules! sym {
                ($name:expr, $ty:ty) => {{
                    let s = CString::new($name).ok()?;
                    let ptr = libc::dlsym(handle, s.as_ptr());
                    if ptr.is_null() {
                        libc::dlclose(handle);
                        return None;
                    }
                    std::mem::transmute::<*mut c_void, $ty>(ptr)
                }};
            }

            Some(Self {
                _handle: handle,
                init: sym!("ndpi_init_detection_module", FnInit),
                set_bitmask: sym!("ndpi_set_protocol_detection_bitmask2", FnSetBitmask),
                process_packet: sym!("ndpi_detection_process_packet", FnProcessPacket),
                get_proto_name: sym!("ndpi_get_proto_name", FnGetProtoName),
                exit: sym!("ndpi_exit_detection_module", FnExit),
                malloc_flow: sym!("ndpi_flow_malloc", FnMallocFlow),
                free_flow: sym!("ndpi_flow_free", FnFreeFlow),
            })
        }
    }

    #[cfg(windows)]
    fn try_load_name(name: &str) -> Option<Self> {
        use std::os::raw::c_char;
        extern "system" {
            fn LoadLibraryA(lp_lib_file_name: *const c_char) -> *mut c_void;
            fn GetProcAddress(h_module: *mut c_void, lp_proc_name: *const c_char) -> *mut c_void;
        }

        unsafe {
            let c_name = CString::new(name).ok()?;
            let handle = LoadLibraryA(c_name.as_ptr());
            if handle.is_null() {
                return None;
            }

            macro_rules! sym {
                ($symbol:expr, $ty:ty) => {{
                    let s = CString::new($symbol).ok()?;
                    let ptr = GetProcAddress(handle, s.as_ptr());
                    if ptr.is_null() {
                        return None;
                    }
                    std::mem::transmute::<*mut c_void, $ty>(ptr)
                }};
            }

            Some(Self {
                _handle: handle,
                init: sym!("ndpi_init_detection_module", FnInit),
                set_bitmask: sym!("ndpi_set_protocol_detection_bitmask2", FnSetBitmask),
                process_packet: sym!("ndpi_detection_process_packet", FnProcessPacket),
                get_proto_name: sym!("ndpi_get_proto_name", FnGetProtoName),
                exit: sym!("ndpi_exit_detection_module", FnExit),
                malloc_flow: sym!("ndpi_flow_malloc", FnMallocFlow),
                free_flow: sym!("ndpi_flow_free", FnFreeFlow),
            })
        }
    }

    #[cfg(not(any(unix, windows)))]
    fn try_load_name(_name: &str) -> Option<Self> {
        None
    }

    /// `true` if `NdpiLib::load()` has succeeded at least once in this process.
    pub fn is_available() -> bool {
        NDPI_LOADED.load(Ordering::SeqCst)
    }
}

// ── DpiResult ────────────────────────────────────────────────────────────────

/// Classification result returned by `NdpiWrapper::classify()`.
#[derive(Debug, Clone)]
pub struct DpiResult {
    /// Application protocol ID (e.g. 7 = HTTP, 91 = TLS).
    pub protocol: u16,
    /// Human-readable protocol name from nDPI.
    pub protocol_name: String,
    /// Outer/encapsulating protocol ID (0 if none).
    pub master_protocol: u16,
    /// 0–100 confidence score (nDPI does not expose this directly; we use 100
    /// when nDPI returns a non-zero protocol, 0 otherwise).
    pub confidence: u8,
    /// Raw `ndpi_protocol` result for further inspection.
    pub raw: ndpi_protocol,
}

// ── NdpiWrapper ───────────────────────────────────────────────────────────────

/// Safe Rust wrapper around a live nDPI detection module.
pub struct NdpiWrapper {
    lib: NdpiLib,
    module: *mut ndpi_detection_module_struct,
}

// SAFETY: module pointer owned exclusively by this struct after init.
unsafe impl Send for NdpiWrapper {}

impl NdpiWrapper {
    /// Initialise an nDPI detection module with all protocols enabled.
    ///
    /// Returns `None` if `ndpi_init_detection_module` returns null.
    pub fn new(lib: NdpiLib) -> Option<Self> {
        unsafe {
            let module = (lib.init)(std::ptr::null());
            if module.is_null() {
                warn!("ndpi_init_detection_module returned NULL");
                return None;
            }
            let bitmask = ndpi_protocol_bitmask::all_enabled();
            (lib.set_bitmask)(module, &bitmask);
            Some(Self { lib, module })
        }
    }

    /// Allocate a new per-flow struct.
    pub fn alloc_flow(&self) -> *mut ndpi_flow_struct {
        unsafe { (self.lib.malloc_flow)() }
    }

    /// Free a flow struct allocated by `alloc_flow()`.
    ///
    /// # Safety
    /// `flow` must have been returned by `alloc_flow()` on this wrapper.
    pub unsafe fn free_flow(&self, flow: *mut ndpi_flow_struct) {
        (self.lib.free_flow)(flow);
    }

    /// Classify `packet` (raw Ethernet frame) using the provided `flow`.
    ///
    /// `tick` is a monotonic counter (e.g. packet sequence number or
    /// milliseconds since epoch) used by nDPI for timeout tracking.
    pub fn classify(
        &self,
        flow: *mut ndpi_flow_struct,
        packet: &[u8],
        tick: u64,
    ) -> DpiResult {
        unsafe {
            let proto = (self.lib.process_packet)(
                self.module,
                flow,
                packet.as_ptr(),
                packet.len().min(u16::MAX as usize) as u16,
                tick,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
            );

            let name_ptr = (self.lib.get_proto_name)(self.module, proto.app_protocol);
            let protocol_name = if name_ptr.is_null() {
                "Unknown".to_string()
            } else {
                CStr::from_ptr(name_ptr).to_string_lossy().into_owned()
            };

            let confidence: u8 = if proto.app_protocol != 0 { 100 } else { 0 };

            DpiResult {
                protocol: proto.app_protocol,
                protocol_name,
                master_protocol: proto.master_protocol,
                confidence,
                raw: proto,
            }
        }
    }

    /// Look up a protocol name by ID.
    pub fn proto_name(&self, proto_id: u16) -> String {
        unsafe {
            let ptr = (self.lib.get_proto_name)(self.module, proto_id);
            if ptr.is_null() {
                return format!("proto_{proto_id}");
            }
            CStr::from_ptr(ptr).to_string_lossy().into_owned()
        }
    }
}

impl Drop for NdpiWrapper {
    fn drop(&mut self) {
        unsafe {
            (self.lib.exit)(self.module);
        }
    }
}

// ── NdpiFlow — RAII per-flow wrapper ─────────────────────────────────────────

/// RAII wrapper for a single nDPI flow allocation.
pub struct NdpiFlow<'a> {
    wrapper: &'a NdpiWrapper,
    ptr: *mut ndpi_flow_struct,
}

impl<'a> NdpiFlow<'a> {
    pub fn new(wrapper: &'a NdpiWrapper) -> Option<Self> {
        let ptr = wrapper.alloc_flow();
        if ptr.is_null() {
            None
        } else {
            Some(Self { wrapper, ptr })
        }
    }

    /// Classify a packet using this flow's state.
    pub fn classify(&self, packet: &[u8], tick: u64) -> DpiResult {
        self.wrapper.classify(self.ptr, packet, tick)
    }
}

impl<'a> Drop for NdpiFlow<'a> {
    fn drop(&mut self) {
        if !self.ptr.is_null() {
            unsafe { self.wrapper.free_flow(self.ptr) };
        }
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ndpi_protocol_default_is_zero() {
        let p = ndpi_protocol::default();
        assert_eq!(p.master_protocol, 0);
        assert_eq!(p.app_protocol, 0);
        assert_eq!(p.category, 0);
    }

    #[test]
    fn bitmask_all_enabled() {
        let bm = ndpi_protocol_bitmask::all_enabled();
        for word in &bm.fds_bits {
            assert_eq!(*word, u32::MAX);
        }
    }

    #[test]
    fn load_graceful_when_lib_absent() {
        // nDPI may not be installed; verify no panic
        let lib = NdpiLib::load();
        // just ensure it doesn't crash
        let _ = lib;
    }

    #[test]
    fn ndpi_available_reflects_load() {
        // After the test above, the flag may or may not be set
        // depending on the environment.  Just make sure it's readable.
        let _ = NdpiLib::is_available();
    }

    #[test]
    fn dpi_result_fields() {
        let r = DpiResult {
            protocol: 7,
            protocol_name: "HTTP".into(),
            master_protocol: 0,
            confidence: 100,
            raw: ndpi_protocol { master_protocol: 0, app_protocol: 7, category: 0 },
        };
        assert_eq!(r.protocol, 7);
        assert_eq!(r.protocol_name, "HTTP");
        assert_eq!(r.confidence, 100);
    }
}
