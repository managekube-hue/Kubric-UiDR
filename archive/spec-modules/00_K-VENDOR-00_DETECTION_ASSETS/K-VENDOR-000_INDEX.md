# Kubric — Vendor Detection Assets Index

## Status: SYNC SCRIPTS READY — Run `scripts/sync-vendor-assets.sh --all` to download

Detection rules, threat intelligence, and security data files are **not bundled in git**.
They are fetched at dev-environment setup or CI time using the sync script below.

## Quick Start

```bash
# From repo root — sync all vendor assets (requires git + curl, ~3 GB total)
bash scripts/sync-vendor-assets.sh --all

# Selective sync
bash scripts/sync-vendor-assets.sh --sigma --mitre --nuclei
```

---

## Vendor Manifest

| Folder | Source | License | Approx Size | Status |
|--------|--------|---------|-------------|--------|
| `K-VENDOR-SIGMA` | SigmaHQ/sigma | Apache 2.0 | ~25 k YAML rules | Not synced |
| `K-VENDOR-YARA` | Neo23x0/signature-base + elastic/detection-rules | BSD / Apache 2.0 | ~5 k rules | Not synced |
| `K-VENDOR-MITRE` | mitre/cti + CWE XML | CC BY 4.0 | ~200 MB JSON | Not synced |
| `K-VENDOR-NUCLEI` | projectdiscovery/nuclei-templates | MIT | ~8 k templates | Not synced |
| `K-VENDOR-BLOODHOUND` | SpecterOps/BloodHound | Apache 2.0 | ~100 Cypher files | Not synced |
| `K-VENDOR-VELOCIRAPTOR` | Velocidex/velociraptor-artifact-exchange | AGPL 3.0 (data) | ~1 k YAML artifacts | Not synced |
| `K-VENDOR-MISP` | MISP-Project repos + CISA KEV | CC0 / CC BY 4.0 | ~120 k TI assets | Not synced |
| `K-VENDOR-SURICATA` | Emerging Threats Open | GPL 2.0 | ~5 k .rules files | Not synced |
| `K-VENDOR-WAZUH` | wazuh/wazuh ruleset | GPL 2.0 | ~2 k XML rules | Not synced |
| `K-VENDOR-FALCO` | falcosecurity/rules | Apache 2.0 | ~50 YAML rules | Not synced |
| `K-VENDOR-OSQUERY` | osquery/osquery packs | Apache 2.0 | ~30 JSON packs | Not synced |
| `K-VENDOR-OPENSCAP` | ComplianceAsCode/content | Apache / LGPL | ~500 MB SCAP content | Not synced |
| `K-VENDOR-CORTEX` | TheHive-Project/Cortex-Analyzers | AGPL 3.0 (sidecar) | — | Sidecar only |
| `K-VENDOR-THEHIVE` | TheHive-Project | AGPL 3.0 (sidecar) | — | Sidecar only |
| `K-VENDOR-SHUFFLE` | Shuffle-SOAR | GPL 3.0 (sidecar) | — | Sidecar only |
| `K-VENDOR-OSCAL` | usnistgov/OSCAL | public domain | ~50 MB | Not synced |

---

## Integration Strategy (per Orchestration Doc)

| Integration Type | Libraries / Tools |
|---|---|
| **Direct Import** (MIT/Apache) | Nuclei, yara-x, OPA, Aya-rs, BloodHound Cypher |
| **Vendor Data Files** (GPL data) | Sigma YAML, Suricata .rules, Wazuh XML, Falco YAML |
| **Subprocess / Sidecar** (AGPL) | Cortex, TheHive, Velociraptor, Shuffle |
| **REST API Pull** (public) | NVD, CISA KEV, OTX, EPSS, AbuseIPDB |
| **FFI / Dynamic Link** (LGPL) | nDPI (`libnDPI.so` loaded via `dlopen` at runtime) |

---

## License Compliance Notes

- **GPL 2.0 rule files** (Suricata, Wazuh): Used as data/configuration — not compiled into Kubric binaries. GPL copyleft does not propagate to data consumers.
- **AGPL 3.0 tools** (Cortex, TheHive, Velociraptor, Shuffle): Run as isolated subprocesses or Docker containers. Kubric communicates over REST/gRPC. Source code is NOT linked.
- **MITRE ATT&CK / CAPEC** (CC BY 4.0): Attribution notice required in any redistribution.
- **MISP data** (CC0): No restrictions.
- See `K-VENDOR-RUD-003_license_boundary.md` for the full legal boundary analysis.

---

## .gitignore Rules

The following vendor subdirectories are excluded from git to avoid bloating the repo.
Add to `.gitignore` if not already present:

```gitignore
# Vendor detection asset downloads (synced at setup time)
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/rules/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/*/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/stix2/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/cwe/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/nuclei-templates/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/bloodhound-ce/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/custom-queries/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/velociraptor-docs/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/artifact-exchange/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/taxonomies/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/galaxies/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/warninglists/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/objects/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/cisa-kev/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/emerging-threats/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/wazuh-rules/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/falco-rules/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/osquery-packs/
00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/scap-content/
```
