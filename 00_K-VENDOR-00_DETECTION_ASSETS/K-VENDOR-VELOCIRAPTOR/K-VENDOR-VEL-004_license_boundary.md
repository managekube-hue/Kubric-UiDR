# K-VENDOR-VEL-004 -- Velociraptor License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | AGPL-3.0                                  |
| Boundary    | Process / network isolation required       |

## AGPL-3.0 Obligations

Velociraptor is licensed under the GNU Affero General Public License v3.
Any software that links to or incorporates AGPL code and is made
available over a network must also be released under AGPL.

## Kubric Compliance Strategy

- **No source-level integration.** Kubric does not import, link, or
  embed any Velociraptor library, module, or source file.
- **Process isolation.** Velociraptor runs as an independent container
  managed by Watchdog. Communication uses HTTP REST/gRPC only.
- **API boundary.** CoreSec, KAI-HUNTER, and KAI-ANALYST interact
  exclusively through Velociraptor's documented REST API.
- **Separate distribution.** Velociraptor binaries are pulled from
  upstream releases; Kubric does not redistribute modified copies.

## Permitted Operations

| Operation                        | Method           | Compliant |
|----------------------------------|------------------|-----------|
| Launch hunts via REST API        | HTTP POST        | Yes       |
| Retrieve results via REST API    | HTTP GET         | Yes       |
| Import Velociraptor Go packages  | Go import        | No        |
| Embed VQL engine in CoreSec      | Library linking  | No        |

## Audit Notes

- Review integration touchpoints quarterly.
- Maintain API-only evidence in architecture decision records.
