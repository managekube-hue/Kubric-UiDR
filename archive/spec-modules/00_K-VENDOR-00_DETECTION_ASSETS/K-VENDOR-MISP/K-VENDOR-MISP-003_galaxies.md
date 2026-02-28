# K-VENDOR-MISP-003 -- MISP Galaxies

## Overview

MISP galaxies are knowledge bases of threat actors, malware families, attack tools, and threat clusters. Each galaxy contains a set of clusters with metadata, synonyms, and relationships -- essentially a structured threat intelligence encyclopedia.

## Data Structure

```json
{
  "name": "Threat Actor",
  "type": "threat-actor",
  "description": "Known threat actors and APT groups",
  "version": 12,
  "values": [
    {
      "value": "APT28",
      "description": "Russian GRU-linked threat actor",
      "meta": {
        "synonyms": ["Fancy Bear", "Sofacy", "Sednit"],
        "country": "RU",
        "cfr-target-category": ["Government", "Military"],
        "refs": ["https://attack.mitre.org/groups/G0007/"]
      }
    }
  ]
}
```

## Key Galaxies Used by Kubric

| Galaxy | Content | Kubric Usage |
|---|---|---|
| **threat-actor** | APT groups with aliases, TTPs, targets | KAI-Hunter attribution during threat hunts |
| **mitre-attack-pattern** | ATT&CK techniques as galaxy clusters | Cross-reference with MIT-002 STIX data |
| **ransomware** | Ransomware families with IOCs and behaviors | KAI-Analyst incident classification |
| **tool** | Attacker tools (Mimikatz, Impacket, etc.) | VDR tool-to-technique mapping |
| **sector** | Industry sectors for targeting context | KAI-Risk sector-specific loss modeling |
| **malpedia** | Comprehensive malware database | YARA rule-to-family correlation |

## Kubric Integration

KAI-Hunter loads galaxy data to build threat actor profiles during hypothesis-driven hunts. When an IOC matches a galaxy cluster, the full actor profile (aliases, known TTPs, target sectors) is included in the `findings` payload published to `kubric.kai.hunter.findings`.

VDR uses the `mitre-attack-pattern` galaxy as a secondary cross-reference to validate ATT&CK mappings from the primary STIX bundle (MIT-002).

## File Layout

```
vendor/misp/galaxy/clusters/
  threat-actor.json
  ransomware.json
  mitre-attack-pattern.json
  ...
```
