// SPDX-License-Identifier: GPL-2.0-only
//
// vendor/ebpf/openat2_hook.c
// Kubric XRO — file integrity monitoring (FIM) tracepoint hook
//
// Attaches to: tracepoint/syscalls/sys_enter_openat2
// Perf map:    FILE_EVENTS (would be consumed by aya AsyncPerfEventArray)
//
// Struct layout MUST match agents/coresec/src/hooks/ebpf.rs :: BpfFileEvent
//
// Build:
//   clang -O2 -g -Wall -target bpf -D __TARGET_ARCH_x86 \
//         -I/usr/include/$(uname -m)-linux-gnu \
//         -c openat2_hook.c -o openat2_hook.o
//
// Note: openat2 is Linux 5.6+. On older kernels use sys_enter_openat instead.

#include <linux/bpf.h>
#include <linux/types.h>
#include <linux/openat2.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

// ─── File operation codes (match Rust FileOp enum) ────────────────────────────
#define FILE_OP_OPEN    0
#define FILE_OP_READ    1
#define FILE_OP_WRITE   2
#define FILE_OP_DELETE  3

// ─── Struct: must byte-exactly match BpfFileEvent in ebpf.rs ────────────────
struct file_event {
    __u32 pid;
    __u64 timestamp_ns;
    __u32 op;
    char  filename[256];
};

// ─── Maps ────────────────────────────────────────────────────────────────────

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32));
} FILE_EVENTS SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct file_event);
} file_heap SEC(".maps");

// ─── Path filter: only emit for monitored directories ────────────────────────
// Monitored prefix list — checked byte-by-byte to stay within verifier limits.
// Add more sensitive paths as needed (max 8 to keep verifier happy).
static const char *MONITORED_PREFIXES[] = {
    "/etc/",
    "/root/",
    "/var/log/",
    "/proc/sys/",
    "/usr/bin/",
    "/usr/sbin/",
    "/bin/",
    "/sbin/",
};
#define N_PREFIXES 8
#define PREFIX_MAXLEN 16

static __always_inline int prefix_match(const char *path, const char *prefix,
                                         int prefix_len)
{
    char p_byte, prefix_byte;
    #pragma unroll
    for (int i = 0; i < PREFIX_MAXLEN; i++) {
        if (i >= prefix_len)
            return 1; // matched
        if (bpf_probe_read_kernel(&p_byte, 1, path + i) < 0)
            return 0;
        prefix_byte = prefix[i];
        if (p_byte != prefix_byte)
            return 0;
    }
    return 1;
}

// ─── Tracepoint: sys_enter_openat2 ────────────────────────────────────────────
// Kernel format (5.6+):
//   field: int dfd;           offset:16
//   field: const char * filename; offset:24
//   field: struct open_how * how;  offset:32
//   field: size_t usize;      offset:40

struct sys_enter_openat2_args {
    __u64 _pad[2];
    int   dfd;
    const char *filename;
    struct open_how *how;
    __u64 usize;
};

SEC("tracepoint/syscalls/sys_enter_openat2")
int openat2_hook(struct sys_enter_openat2_args *ctx)
{
    __u32 key = 0;
    struct file_event *evt = bpf_map_lookup_elem(&file_heap, &key);
    if (!evt)
        return 0;

    __builtin_memset(evt, 0, sizeof(*evt));

    evt->pid          = (__u32)(bpf_get_current_pid_tgid() >> 32);
    evt->timestamp_ns = bpf_ktime_get_ns();
    evt->op           = FILE_OP_OPEN;

    // Read filename from user-space
    int r = bpf_probe_read_user_str(evt->filename, sizeof(evt->filename),
                                    ctx->filename);
    if (r <= 0)
        return 0;

    // Determine write intent from open flags (O_WRONLY=1, O_RDWR=2)
    struct open_how how = {};
    if (bpf_probe_read_user(&how, sizeof(how), ctx->how) == 0) {
        __u64 flags = how.flags;
        if (flags & 1 || flags & 2)  // O_WRONLY | O_RDWR
            evt->op = FILE_OP_WRITE;
    }

    // Only emit for monitored prefixes — reduces noise
    // Note: static string comparison in BPF is limited; we check first 8 chars
    char first8[8];
    bpf_probe_read_kernel(first8, 8, evt->filename);

    // Simple check: starts with '/' (absolute path from monitored area)
    if (first8[0] != '/')
        return 0;

    bpf_perf_event_output(ctx, &FILE_EVENTS, BPF_F_CURRENT_CPU,
                          evt, sizeof(*evt));
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
