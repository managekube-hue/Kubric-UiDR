# K-VENDOR-MISP-002 -- MISP Taxonomies

## Overview

MISP taxonomies are machine-readable classification vocabularies expressed as JSON. They provide standardized tagging for threat intelligence events, enabling consistent labeling across tools and organizations.

## Data Structure

```json
{
  "namespace": "tlp",
  "description": "Traffic Light Protocol",
  "version": 5,
  "predicates": [
    { "value": "white", "expanded": "TLP:WHITE - Unlimited disclosure" },
    { "value": "green", "expanded": "TLP:GREEN - Community-wide" },
    { "value": "amber", "expanded": "TLP:AMBER - Limited disclosure" },
    { "value": "red",   "expanded": "TLP:RED - Named recipients only" }
  ]
}
```

## Key Taxonomies Used by Kubric

| Taxonomy | Purpose | Kubric Consumer |
|---|---|---|
| **tlp** | Traffic Light Protocol marking on shared IOCs | VDR, KAI-Analyst |
| **admiralty-scale** | Source reliability + information credibility | KAI-Analyst confidence scoring |
| **kill-chain** | Lockheed Martin Cyber Kill Chain phase | KAI-Hunter hypothesis mapping |
| **veris** | VERIS incident classification | KAI-Risk FAIR modeling inputs |
| **ms-caro-malware** | Microsoft malware naming convention | VDR malware family normalization |
| **estimated-severity** | Severity rating (low/medium/high/critical) | Alert prioritization across all agents |

## Kubric Integration

The VDR Go service loads taxonomy JSON at startup from `vendor/misp/taxonomies/`. When ingesting IOCs from external MISP instances via REST API, taxonomy tags are parsed and normalized into Kubric's internal tagging schema. KAI agents reference taxonomy predicates to standardize severity and TLP markings across multi-tenant alert streams.

## File Layout

```
vendor/misp/taxonomies/
  tlp/machinetag.json
  admiralty-scale/machinetag.json
  kill-chain/machinetag.json
  ...
```

Each taxonomy directory contains a single `machinetag.json` file with the full vocabulary definition.
