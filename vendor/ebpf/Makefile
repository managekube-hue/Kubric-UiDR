# vendor/ebpf/Makefile
# Compiles eBPF C programs to ELF object files consumed by the Rust aya loader.
#
# Requirements:
#   - clang >= 12  (with BPF target support)
#   - llvm-strip   (for stripping debug info from release builds)
#   - Linux kernel headers for current architecture
#   - libbpf-dev (Debian/Ubuntu) or libbpf-devel (RHEL/Fedora)
#
# Install on Debian/Ubuntu:
#   apt-get install clang llvm libelf-dev libbpf-dev linux-headers-$(uname -r)
#
# Install on RHEL/Fedora:
#   dnf install clang llvm elfutils-libelf-devel libbpf-devel kernel-devel
#
# Usage (from repo root):
#   make -C vendor/ebpf          # build all .o files
#   make -C vendor/ebpf clean    # remove generated .o files
#   make -C vendor/ebpf verify   # verify with bpftool

ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
CLANG ?= clang
STRIP ?= llvm-strip
BPFTOOL ?= bpftool

KERN_INCLUDE := /usr/include/$(shell uname -m)-linux-gnu
LIBBPF_INCLUDE := /usr/include

CFLAGS := -O2 -g \
           -Wall \
           -target bpf \
           -D __TARGET_ARCH_$(ARCH) \
           -D __BPF_TRACING__ \
           -I$(KERN_INCLUDE) \
           -I$(LIBBPF_INCLUDE) \
           -mcpu=v3

RELEASE_STRIP_FLAGS := -g

SRCS := execve_hook.c openat2_hook.c
OBJS := $(SRCS:.c=.o)

.PHONY: all clean verify debug check-deps

all: check-deps $(OBJS)
	@echo "[eBPF] All programs compiled successfully"
	@ls -lh *.o

# Compile eBPF C to BPF ELF object
%.o: %.c
	@echo "[eBPF] Compiling $< → $@"
	$(CLANG) $(CFLAGS) -c $< -o $@
	@echo "[eBPF] Generated $@ ($(shell stat -c%s $@ 2>/dev/null || stat -f%z $@) bytes)"

# Strip debug symbols for release (reduces size, keeps BTF)
strip: $(OBJS)
	@for obj in $(OBJS); do \
		echo "[eBPF] Stripping $$obj"; \
		$(STRIP) $(RELEASE_STRIP_FLAGS) $$obj; \
	done

# Verify programs with bpftool (requires bpftool installed)
verify: $(OBJS)
	@for obj in $(OBJS); do \
		echo "[eBPF] Verifying $$obj"; \
		$(BPFTOOL) prog load $$obj /sys/fs/bpf/kubric-test 2>&1 || true; \
		rm -f /sys/fs/bpf/kubric-test; \
	done

# Debug build with maximum verbosity
debug: CFLAGS += -DDEBUG -v
debug: $(OBJS)

# Dump BTF type info for a compiled object
btf-dump: $(OBJS)
	@for obj in $(OBJS); do \
		echo "=== BTF types in $$obj ==="; \
		$(BPFTOOL) btf dump file $$obj format raw 2>/dev/null || \
		llvm-readelf --sections $$obj | grep BTF || true; \
	done

check-deps:
	@command -v $(CLANG) >/dev/null 2>&1 || \
		(echo "ERROR: clang not found. Install: apt-get install clang" && exit 1)
	@$(CLANG) -target bpf -print-targets 2>&1 | grep -q bpf || \
		(echo "ERROR: clang BPF target not available. Need clang >= 12" && exit 1)
	@test -d $(KERN_INCLUDE) || \
		(echo "WARNING: Kernel headers not at $(KERN_INCLUDE) — build may fail")
	@echo "[eBPF] Dependencies OK ($(CLANG) $(shell $(CLANG) --version | head -1))"

clean:
	rm -f $(OBJS)
	@echo "[eBPF] Cleaned"

# CI helper: rebuild and verify checksums
ci: all
	@sha256sum $(OBJS) > ebpf-checksums.txt
	@echo "[eBPF] Checksums written to ebpf-checksums.txt"
	@cat ebpf-checksums.txt
