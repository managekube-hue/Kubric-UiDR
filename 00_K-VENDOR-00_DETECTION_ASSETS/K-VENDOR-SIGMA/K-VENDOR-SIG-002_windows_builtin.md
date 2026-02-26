# K-VENDOR-SIG-002 -- Windows Built-in Sigma Rules

| Field | Value |
|-------|-------|
| **Rule Path** | `vendor/sigma/rules/windows/` |
| **Log Sources** | Sysmon, Security, PowerShell, WMI, Windows Defender |
| **Engine** | CoreSec `SigmaEngine` -- field extraction via `extract_field()` |

## Scope

This category covers Sigma rules targeting native Windows event log sources.
CoreSec maps `ProcessEvent` fields (`executable`, `cmdline`, `user`, `pid`,
`ppid`) to Sigma field names (`Image`, `CommandLine`, `User`, `ParentImage`,
`ProcessId`) through a case-insensitive lookup in `extract_field()`.

## Key Rule Families

### Process Creation (Sysmon EventID 1 / Security 4688)

Rules that match suspicious process spawns by image path, command-line
arguments, or parent-child relationships.

- Credential-dumping tools: mimikatz, procdump LSASS access, comsvcs MiniDump
- LOLBins: certutil download, mshta inline script, rundll32 unusual DLL
- Reconnaissance: whoami, nltest, net group
- Lateral movement: PsExec service install, WMIC remote process create

**MITRE ATT&CK**: T1003 (Credential Dumping), T1218 (System Binary Proxy Execution),
T1059 (Command and Scripting Interpreter), T1047 (WMI)

### PowerShell Script Block Logging (EventID 4104)

Rules inspecting decoded script blocks for obfuscation indicators, encoded
commands, AMSI bypass, and known offensive tooling strings.

**MITRE ATT&CK**: T1059.001 (PowerShell), T1027 (Obfuscated Files)

### Service Installation (System EventID 7045)

Rules detecting new service creation, especially those using `cmd.exe`,
`powershell.exe`, or temporary paths as the service binary.

**MITRE ATT&CK**: T1543.003 (Windows Service), T1569.002 (Service Execution)

### Scheduled Task / Registry (Security 4698 / Sysmon 13)

Rules covering persistence via scheduled tasks, Run/RunOnce registry keys,
and autostart locations.

**MITRE ATT&CK**: T1053.005 (Scheduled Task), T1547.001 (Registry Run Keys)

## CoreSec Evaluation Flow

```
ProcessEvent arrives via NATS (kubric.{tenant}.endpoint.process.v1)
  --> SigmaEngine::evaluate(&event)
        --> for each loaded windows rule:
              extract_field(event, "Image")       -> event.executable
              extract_field(event, "CommandLine")  -> event.cmdline
              apply field modifiers (contains, startswith, endswith, re)
              eval_condition("selection and not filter", ...)
        --> matching rules -> Vec<SigmaMatch>
  --> SigmaMatch.tags carry ATT&CK technique IDs (e.g. "attack.t1003")
```

## Filter / Tuning

Many Windows rules include a `filter` group to exclude known-good binaries.
Kubric-specific tuning rules are placed in `vendor/sigma/rules/windows/overrides/`
and referenced via `condition: selection and not filter and not kubric_tuning`.
