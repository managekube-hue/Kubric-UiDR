# K-XRO-CS-002 — eBPF Compatibility Reference

**Document:** K-XRO-CS-002
**Component:** K-XRO CoreSec — eBPF Hook Subsystem
**Last updated:** 2025-02
**Status:** Approved

---

## 1. Kernel Version Requirements

| Feature | Minimum Kernel | Notes |
|---|---|---|
| BPF tracepoints | 4.7 | `BPF_PROG_TYPE_TRACEPOINT` introduced |
| `CAP_BPF` capability | 5.8 | Separates eBPF from `CAP_SYS_ADMIN` |
| Async perf ring buffer | 5.8 | Required by `aya` `AsyncPerfEventArray` |
| BTF (BPF Type Format) | 5.2 | Required for CO-RE (Compile Once, Run Everywhere) |
| `openat2(2)` syscall | 5.6 | Needed for `sys_enter_openat2` tracepoint |
| Ring buffer map (`BPF_MAP_TYPE_RINGBUF`) | 5.8 | Higher-throughput alternative to perf |
| `CAP_PERFMON` capability | 5.8 | Alternative to `CAP_SYS_ADMIN` for perf events |

**Recommended minimum: Linux 5.8** for full CoreSec eBPF functionality.

Distributions that ship 5.8+ by default:
- Ubuntu 20.04 LTS HWE kernel (5.15+)
- Ubuntu 22.04 LTS (5.15 GA)
- Debian 11 "Bullseye" (5.10)
- RHEL/CentOS 9 (5.14)
- Amazon Linux 2023 (6.1)
- Bottlerocket OS (typically 5.15+)

---

## 2. Required Linux Capabilities

The CoreSec agent must run with the following capabilities to load and attach eBPF programs. These should be granted via the Kubernetes `securityContext`, not by running as root.

```yaml
# Kubernetes pod securityContext
securityContext:
  capabilities:
    add:
      - CAP_BPF        # load/attach eBPF programs (5.8+)
      - CAP_PERFMON    # open perf event file descriptors (5.8+)
      - CAP_SYS_PTRACE # required for /proc/{pid}/mem access (forensics)
    drop:
      - ALL
  runAsNonRoot: true
  readOnlyRootFilesystem: true
```

On pre-5.8 kernels, `CAP_SYS_ADMIN` is required instead of `CAP_BPF` + `CAP_PERFMON`. This is not recommended for production.

---

## 3. BTF (BPF Type Format) Requirements

BTF enables CO-RE: eBPF programs compiled once can run on any kernel without recompilation. CoreSec uses the `aya` crate which supports BTF-based relocation.

**Checking BTF availability:**
```bash
ls /sys/kernel/btf/vmlinux   # must exist
bpftool btf dump file /sys/kernel/btf/vmlinux | head -5
```

If `/sys/kernel/btf/vmlinux` is absent, the kernel was compiled without `CONFIG_DEBUG_INFO_BTF=y`. In this case, a pre-built BTF file can be provided via the `KUBRIC_BTF_PATH` environment variable using the BTF Hub project.

**Distro kernels with BTF enabled by default:**
- Ubuntu 20.04+ (5.4+)
- Fedora 31+
- RHEL 8.2+
- Debian 11+

---

## 4. eBPF Map Types Used

| Map Name | Type | Max Entries | Purpose |
|---|---|---|---|
| `EXEC_EVENTS` | `BPF_MAP_TYPE_PERF_EVENT_ARRAY` | 65,536 (per-CPU) | execve event ring-buffer |
| `FILE_EVENTS` | `BPF_MAP_TYPE_PERF_EVENT_ARRAY` | 65,536 (per-CPU) | openat2 event ring-buffer |

Both maps use `AsyncPerfEventArray` via the `aya` crate. Userspace spawns one reader task per online CPU.

Map pressure is monitored by `K-XRO-CS-EBPF-003_map_pressure.rs`; an alert fires when utilisation exceeds 80 % to prevent event loss.

---

## 5. Fallback: ETW on Windows

On Windows hosts, the eBPF subsystem is not available. CoreSec falls back to **Event Tracing for Windows (ETW)** via the `K-XRO-CS-ETW` hook provider:

- Process creation events via `Microsoft-Windows-Security-Auditing` (Event ID 4688)
- File access via `Microsoft-Windows-Kernel-File` (ETW provider)
- Implemented in `agents/coresec/src/hooks/etw.rs`

The fallback is transparent to consumers — both eBPF and ETW providers emit the same `HookEvent::ProcessExec` variant on the same MPSC channel.

Feature flag: the ETW path is compiled in on `windows` targets only; the eBPF path requires `--features ebpf` and `target_os = "linux"`.

---

## 6. Tetragon as an Alternative

For Kubernetes environments where running privileged init containers to load eBPF programs is undesirable, **[Cilium Tetragon](https://tetragon.io/)** provides equivalent process and file telemetry from the kernel:

- Tetragon runs as a DaemonSet with its own eBPF programs already loaded
- Events are exported as JSON via a gRPC stream (`/var/run/cilium/tetragon/tetragon.sock`)
- CoreSec can consume the Tetragon gRPC stream as a third hook provider (`K-XRO-CS-TETRAGON`)
- This avoids the need for `CAP_BPF` in the CoreSec pod itself

**Trade-off:** Tetragon adds a cluster-level dependency and the gRPC event schema differs slightly from the raw eBPF layout. A translation layer is required.

---

## 7. Building the eBPF Programs

The pre-compiled ELF objects shipped in `vendor/ebpf/` are built from C sources in `bpf/src/` using `clang` targeting the `bpf` architecture:

```bash
# Build eBPF C programs (requires clang 12+, linux-headers)
cargo xtask build-ebpf

# Outputs:
#   vendor/ebpf/execve_hook.o
#   vendor/ebpf/openat2_hook.o
```

Environment variables controlling the paths at runtime:

| Variable | Default |
|---|---|
| `KUBRIC_EBPF_EXECVE` | `vendor/ebpf/execve_hook.o` |
| `KUBRIC_EBPF_OPENAT` | `vendor/ebpf/openat2_hook.o` |
| `KUBRIC_BTF_PATH` | (empty — use vmlinux from /sys) |

---

## 8. Known Limitations

- **Container environments:** eBPF maps are namespaced by the BPF filesystem mount. In heavily restricted container runtimes (e.g. gVisor/runsc), eBPF is unavailable and the agent falls back to sysinfo polling.
- **ARM64:** The eBPF ELF objects are architecture-specific. Separate builds are required for `amd64` and `arm64`.
- **Kernel lockdown mode:** Systems with `CONFIG_SECURITY_LOCKDOWN_LSM` in integrity mode will block eBPF program loading even with `CAP_BPF`.
- **Older RHEL/CentOS 7 (3.10 kernel):** eBPF is unavailable. CoreSec will operate in sysinfo-only mode.
