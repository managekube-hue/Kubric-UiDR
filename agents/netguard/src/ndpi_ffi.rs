//! nDPI FFI — dynamic loading of libnDPI for Layer 7 protocol classification.
//!
//! nDPI is LGPL-3.0 — loaded at runtime via `dlopen` / `LoadLibrary` to
//! maintain license boundary isolation.  If the shared library is not present
//! at runtime, the DPI module falls back to heuristic-only classification.
//!
//! The FFI declarations here match nDPI 4.x API.
//! Source: <https://github.com/ntop/nDPI>

#![allow(non_camel_case_types, non_snake_case, dead_code)]

use std::ffi::CString;
use std::os::raw::{c_char, c_int, c_uint, c_void};
use std::sync::atomic::{AtomicBool, Ordering};
use tracing::{info, warn};

// ── Opaque nDPI types ────────────────────────────────────────────────────────

/// Opaque detection module handle.
pub type ndpi_detection_module_struct = c_void;

/// Opaque per-flow struct.
pub type ndpi_flow_struct = c_void;

/// nDPI protocol ID.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default)]
pub struct ndpi_protocol {
    pub master_protocol: u16,
    pub app_protocol: u16,
    pub category: u32,
}

/// nDPI protocol bitmask.
#[repr(C)]
pub struct ndpi_protocol_bitmask {
    pub fds_bits: [u32; 16],
}

// ── Function pointer types ───────────────────────────────────────────────────

type FnInit = unsafe extern "C" fn(prefs: *const c_void) -> *mut ndpi_detection_module_struct;
type FnSetProtocolBitmask = unsafe extern "C" fn(
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
type FnGetProtoName = unsafe extern "C" fn(
    module: *mut ndpi_detection_module_struct,
    proto: u16,
) -> *const c_char;
type FnExit = unsafe extern "C" fn(module: *mut ndpi_detection_module_struct);
type FnMallocFlow = unsafe extern "C" fn() -> *mut ndpi_flow_struct;
type FnFreeFlow = unsafe extern "C" fn(flow: *mut ndpi_flow_struct);

// ── Dynamic library handle ────────────────────────────────────────────────────

/// Dynamically loaded nDPI library.
pub struct NdpiLib {
    _handle: *mut c_void,
    pub init: FnInit,
    pub set_protocol_bitmask: FnSetProtocolBitmask,
    pub process_packet: FnProcessPacket,
    pub get_proto_name: FnGetProtoName,
    pub exit: FnExit,
    pub malloc_flow: FnMallocFlow,
    pub free_flow: FnFreeFlow,
}

unsafe impl Send for NdpiLib {}
unsafe impl Sync for NdpiLib {}

static NDPI_AVAILABLE: AtomicBool = AtomicBool::new(false);

impl NdpiLib {
    /// Attempt to dynamically load libnDPI.
    ///
    /// On Linux: loads `libndpi.so` or `libndpi.so.4`
    /// On Windows: loads `ndpi.dll`
    /// On macOS: loads `libndpi.dylib`
    ///
    /// Returns `None` if the library is not found (graceful degradation).
    pub fn load() -> Option<Self> {
        // Prefer explicit runtime path from environment.
        if let Ok(path) = std::env::var("KUBRIC_NDPI_LIB") {
            if let Some(lib) = Self::try_load(&path) {
                NDPI_AVAILABLE.store(true, Ordering::SeqCst);
                info!(library = path, "nDPI loaded from KUBRIC_NDPI_LIB");
                return Some(lib);
            }
        }

        // Try build-script-provided vendor lib dir when present.
        if let Some(candidate) = Self::candidate_vendor_paths().into_iter().find_map(|p| {
            if p.exists() {
                Self::try_load(p.to_string_lossy().as_ref())
            } else {
                None
            }
        }) {
            NDPI_AVAILABLE.store(true, Ordering::SeqCst);
            info!("nDPI loaded from vendor dynamic library path");
            return Some(candidate);
        }

        let lib_names: &[&str] = if cfg!(target_os = "windows") {
            &["ndpi.dll", "libndpi.dll"]
        } else if cfg!(target_os = "macos") {
            &["libndpi.dylib"]
        } else {
            &["libndpi.so", "libndpi.so.4"]
        };

        for name in lib_names {
            if let Some(lib) = Self::try_load(name) {
                NDPI_AVAILABLE.store(true, Ordering::SeqCst);
                info!(library = name, "nDPI loaded successfully via dlopen");
                return Some(lib);
            }
        }

        warn!("nDPI library not found — DPI classification disabled");
        None
    }

    fn candidate_vendor_paths() -> Vec<std::path::PathBuf> {
        let mut out = Vec::new();
        if let Ok(dir) = std::env::var("KUBRIC_NDPI_LIB_DIR") {
            out.push(std::path::PathBuf::from(&dir).join("libndpi.so"));
            out.push(std::path::PathBuf::from(&dir).join("libndpi.so.5"));
        }
        out.push(std::path::PathBuf::from("vendor/ndpi/lib/libndpi.so"));
        out.push(std::path::PathBuf::from("vendor/ndpi/lib/libndpi.so.5"));
        out.push(std::path::PathBuf::from("/opt/kubric/vendor/ndpi/lib/libndpi.so"));
        out.push(std::path::PathBuf::from("/opt/kubric/vendor/ndpi/lib/libndpi.so.5"));
        out
    }

    #[cfg(unix)]
    fn try_load(name: &str) -> Option<Self> {
        unsafe {
            let c_name = CString::new(name).ok()?;
            let handle = libc::dlopen(c_name.as_ptr(), libc::RTLD_LAZY);
            if handle.is_null() {
                return None;
            }

            let init = Self::sym::<FnInit>(handle, "ndpi_init_detection_module")?;
            let set_protocol_bitmask = Self::sym::<FnSetProtocolBitmask>(handle, "ndpi_set_protocol_detection_bitmask2")?;
            let process_packet = Self::sym::<FnProcessPacket>(handle, "ndpi_detection_process_packet")?;
            let get_proto_name = Self::sym::<FnGetProtoName>(handle, "ndpi_get_proto_name")?;
            let exit = Self::sym::<FnExit>(handle, "ndpi_exit_detection_module")?;
            let malloc_flow = Self::sym::<FnMallocFlow>(handle, "ndpi_flow_malloc")?;
            let free_flow = Self::sym::<FnFreeFlow>(handle, "ndpi_flow_free")?;

            Some(Self {
                _handle: handle,
                init,
                set_protocol_bitmask,
                process_packet,
                get_proto_name,
                exit,
                malloc_flow,
                free_flow,
            })
        }
    }

    #[cfg(windows)]
    fn try_load(name: &str) -> Option<Self> {
        unsafe {
            // Use Windows LoadLibraryA for dynamic loading
            extern "system" {
                fn LoadLibraryA(name: *const c_char) -> *mut c_void;
                fn GetProcAddress(module: *mut c_void, name: *const c_char) -> *mut c_void;
            }

            let c_name = CString::new(name).ok()?;
            let handle = LoadLibraryA(c_name.as_ptr());
            if handle.is_null() {
                return None;
            }

            macro_rules! load_sym {
                ($name:expr, $ty:ty) => {{
                    let sym_name = CString::new($name).ok()?;
                    let ptr = GetProcAddress(handle, sym_name.as_ptr());
                    if ptr.is_null() {
                        return None;
                    }
                    std::mem::transmute::<*mut c_void, $ty>(ptr)
                }};
            }

            Some(Self {
                _handle: handle,
                init: load_sym!("ndpi_init_detection_module", FnInit),
                set_protocol_bitmask: load_sym!("ndpi_set_protocol_detection_bitmask2", FnSetProtocolBitmask),
                process_packet: load_sym!("ndpi_detection_process_packet", FnProcessPacket),
                get_proto_name: load_sym!("ndpi_get_proto_name", FnGetProtoName),
                exit: load_sym!("ndpi_exit_detection_module", FnExit),
                malloc_flow: load_sym!("ndpi_flow_malloc", FnMallocFlow),
                free_flow: load_sym!("ndpi_flow_free", FnFreeFlow),
            })
        }
    }

    #[cfg(unix)]
    unsafe fn sym<T>(handle: *mut c_void, name: &str) -> Option<T> {
        let c_name = CString::new(name).ok()?;
        let ptr = libc::dlsym(handle, c_name.as_ptr());
        if ptr.is_null() {
            return None;
        }
        Some(std::mem::transmute_copy(&ptr))
    }

    /// Returns true if nDPI was loaded successfully.
    pub fn is_available() -> bool {
        NDPI_AVAILABLE.load(Ordering::SeqCst)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ndpi_protocol_default() {
        let proto = ndpi_protocol::default();
        assert_eq!(proto.master_protocol, 0);
        assert_eq!(proto.app_protocol, 0);
    }

    #[test]
    fn ndpi_load_graceful_failure() {
        // nDPI may not be installed in CI — verify graceful fallback
        let lib = NdpiLib::load();
        // Just verify it doesn't panic; the result depends on whether
        // nDPI is installed on the build host
        let _ = lib;
    }

    #[test]
    fn ndpi_available_starts_false() {
        // At test start (before loading), the flag should be false
        // unless a previous test loaded it
        let _ = NdpiLib::is_available();
    }
}
