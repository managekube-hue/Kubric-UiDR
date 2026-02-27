// SPDX-License-Identifier: GPL-2.0-only
//
// vendor/ebpf/execve_hook.c
// Kubric XRO — process execution tracepoint hook
//
// Attaches to: tracepoint/syscalls/sys_enter_execve
// Perf map:    EXEC_EVENTS (AsyncPerfEventArray consumed by aya in Rust)
//
// Struct layout MUST match agents/coresec/src/hooks/ebpf.rs :: BpfExecEvent
//
// Build:
//   clang -O2 -g -Wall -target bpf -D __TARGET_ARCH_x86 \
//         -I/usr/include/$(uname -m)-linux-gnu \
//         -c execve_hook.c -o execve_hook.o
//
// Requires: clang >= 12, Linux kernel >= 5.8 with eBPF tracepoints enabled.

#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

// ─── Struct: must byte-exactly match BpfExecEvent in ebpf.rs ────────────────
struct exec_event {
    __u32 pid;
    __u32 ppid;
    __u32 uid;
    __u64 timestamp_ns;
    char  filename[256];
    char  cmdline[512];
};

// ─── Maps ────────────────────────────────────────────────────────────────────

// Perf ring buffer — one slot per CPU, read by aya AsyncPerfEventArray
struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32));
} EXEC_EVENTS SEC(".maps");

// Per-CPU scratch space so we don't overflow the BPF stack (512-byte limit)
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct exec_event);
} exec_heap SEC(".maps");

// ─── Helper: read up to N bytes of a user-space string ───────────────────────
static __always_inline int read_str(void *dst, int dst_size,
                                    const char __user *src)
{
    int ret = bpf_probe_read_user_str(dst, dst_size, src);
    return (ret < 0) ? ret : 0;
}

// ─── Tracepoint: sys_enter_execve ────────────────────────────────────────────
// Kernel tracepoint format (from /sys/kernel/debug/tracing/events/syscalls/
// sys_enter_execve/format):
//   field:const char * filename;  offset:16; size:8; signed:0;
//   field:const char *const * argv; offset:24; size:8; signed:0;
//   field:const char *const * envp; offset:32; size:8; signed:0;

struct sys_enter_execve_args {
    __u64 _pad[2];          // common fields: type (u16) + pad (u16) + pid (u32) + time (u64)
    const char *filename;   // offset 16
    const char *const *argv;
    const char *const *envp;
};

SEC("tracepoint/syscalls/sys_enter_execve")
int execve_hook(struct sys_enter_execve_args *ctx)
{
    __u32 key = 0;
    struct exec_event *evt = bpf_map_lookup_elem(&exec_heap, &key);
    if (!evt)
        return 0;

    // Zero the struct (per-CPU heap may contain previous data)
    __builtin_memset(evt, 0, sizeof(*evt));

    // Process and parent IDs
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    evt->pid = (__u32)(pid_tgid >> 32);   // tgid = process ID
    evt->uid = (__u32)bpf_get_current_uid_gid();

    // Parent PID via task_struct (CO-RE)
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    evt->ppid = (__u32)BPF_CORE_READ(task, real_parent, tgid);

    evt->timestamp_ns = bpf_ktime_get_ns();

    // Executable path from tracepoint arg
    bpf_probe_read_user_str(evt->filename, sizeof(evt->filename), ctx->filename);

    // Reconstruct cmdline: argv[0] + " " + argv[1] + ...
    // We read up to 4 argv entries to stay within verifier complexity limits.
    const char *arg;
    char *dst = evt->cmdline;
    int remaining = sizeof(evt->cmdline) - 1;
    int total = 0;

    #pragma unroll
    for (int i = 0; i < 4; i++) {
        if (remaining <= 0)
            break;
        if (bpf_probe_read_user(&arg, sizeof(arg), &ctx->argv[i]) < 0)
            break;
        if (!arg)
            break;
        int n = bpf_probe_read_user_str(dst, remaining, arg);
        if (n <= 0)
            break;
        // n includes the null terminator — convert to space separator
        if (n > 1 && i > 0) {
            // Already wrote null at dst[n-1]; move back and write space
            total += (n - 1);
            dst   += (n - 1);
            if (remaining > n) {
                *dst = ' ';
                dst++;
                remaining -= n;
            } else {
                break;
            }
        } else {
            total += (n - 1);
            dst   += (n - 1);
            remaining -= (n - 1);
        }
    }

    // Submit to perf ring buffer
    bpf_perf_event_output(ctx, &EXEC_EVENTS, BPF_F_CURRENT_CPU,
                          evt, sizeof(*evt));
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
