# Audit Remediation Checkpoint (2026-02-27)

This checkpoint aligns implementation to the architecture foundations in `docs/KUBRIC Orchestration.docx.md`:

- CoreSec eBPF programs must be available at runtime.
- Windows process hooks must emit events (not no-op polling).
- PerfTrace metrics must include real disk I/O counters.
- Vendor detection rule sets (YARA/Sigma) must be populated and verifiable.

## Implemented fixes

1. **eBPF object availability**
   - Added auto-build fallback in `agents/coresec/src/hooks/ebpf.rs` (`ensure_ebpf_object`).
   - Updated `Dockerfile.agents` to build eBPF objects during image build and copy them into CoreSec runtime image.
   - Added runtime env vars in CoreSec image:
     - `KUBRIC_EBPF_EXECVE=/opt/kubric/vendor/ebpf/execve_hook.o`
     - `KUBRIC_EBPF_OPENAT=/opt/kubric/vendor/ebpf/openat2_hook.o`

2. **Windows hook processing**
   - Replaced ETW no-op placeholder loop in `agents/coresec/src/hooks/etw.rs` with a functional process-event loop using `sysinfo` process-table deltas.
   - Provider now emits `HookEvent::ProcessExec` events for new processes.

3. **PerfTrace disk I/O metrics**
   - Removed hardcoded zero values in `agents/perftrace/src/agent.rs`.
   - Added Linux collector `collect_disk_io_bytes()` using `/proc/diskstats` cumulative sector counters.

4. **One-command audit verification**
   - Added `scripts/bootstrap/ops-batch-06-audit-verify.ps1`.
   - Added `make ops-batch-06` target in `Makefile`.

## Verification command

```powershell
make ops-batch-06
```

Expected result:

- `[batch-06] Audit remediation verification PASSED`

## Notes

- Rust compile validation (`cargo check`) could not be executed in the current shell because `cargo` is not installed in this environment.
