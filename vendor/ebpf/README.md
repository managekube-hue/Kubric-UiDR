# vendor/ebpf — Kubric XRO eBPF Programs

Pre-compiled eBPF kernel programs for the Kubric eXtended Response Observer.
These programs attach to kernel tracepoints and stream events to the Rust `aya`
loader in `agents/coresec/src/hooks/ebpf.rs`.

## Programs

| Source | Object | Tracepoint | Perf Map | Purpose |
|--------|--------|------------|----------|---------|
| `execve_hook.c` | `execve_hook.o` | `syscalls/sys_enter_execve` | `EXEC_EVENTS` | Process execution monitoring |
| `openat2_hook.c` | `openat2_hook.o` | `syscalls/sys_enter_openat2` | `FILE_EVENTS` | File integrity monitoring (FIM) |

## Build Requirements

- Linux host (cross-compilation from macOS/Windows not supported for eBPF)
- clang >= 12 with BPF target support
- Linux kernel headers for the **target** kernel version
- libbpf-dev (for BPF helper definitions)

```bash
# Debian/Ubuntu
sudo apt-get install clang llvm libelf-dev libbpf-dev \
     linux-headers-$(uname -r) linux-tools-$(uname -r)

# RHEL/Fedora
sudo dnf install clang llvm elfutils-libelf-devel libbpf-devel kernel-devel
```

## Building

```bash
# From repo root
make -C vendor/ebpf

# Or directly
cd vendor/ebpf
make

# Release build (strip debug symbols, keeps BTF)
make strip

# Verify with bpftool (requires root)
sudo make verify
```

The compiled `.o` files are checked in to git so the Rust build does not require
a Linux build environment. Re-compile only when updating the C source.

## Runtime Requirements

The aya loader in the Rust agent requires:

- Linux kernel >= 5.8 (for `CAP_BPF` + `CAP_PERFMON` without root)
- Or: Linux kernel >= 4.4 with root (`CAP_SYS_ADMIN`)
- eBPF JIT enabled: `sysctl net.core.bpf_jit_enable=1`
- Perf events: `sysctl kernel.perf_event_paranoid=-1` (or run as root)

## Struct Compatibility

The C structs **must** byte-exactly match the Rust `#[repr(C)]` structs in
`agents/coresec/src/hooks/ebpf.rs`:

```c
// C (execve_hook.c)       ↔  Rust (ebpf.rs)
struct exec_event {           struct BpfExecEvent {
    __u32 pid;                    pid: u32,
    __u32 ppid;                   ppid: u32,
    __u32 uid;                    uid: u32,
    __u64 timestamp_ns;           timestamp_ns: u64,
    char  filename[256];          filename: [u8; 256],
    char  cmdline[512];           cmdline: [u8; 512],
};                            }
```

Any change to either struct requires rebuilding both the C and Rust code.

## OCSF Mapping

Events are forwarded by the Rust agent to the OCSF normalizer:
- `sys_enter_execve` → OCSF class 4007 (Process Activity)
- `sys_enter_openat2` → OCSF class 1001 (File System Activity)
