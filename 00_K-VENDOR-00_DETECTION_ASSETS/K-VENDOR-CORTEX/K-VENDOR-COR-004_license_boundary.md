# K-VENDOR-COR-004 -- Cortex License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | AGPL-3.0                                  |
| Boundary    | Process / network isolation required       |

## AGPL-3.0 Obligations

Cortex is licensed under AGPL-3.0. Network interaction with AGPL
software triggers source-disclosure obligations only if the interacting
software incorporates AGPL-licensed code.

## Kubric Compliance Strategy

- **No source-level integration.** Kubric does not import, link, or
  compile any Cortex source code into its binaries.
- **Process isolation.** Cortex runs as a standalone Docker container.
  KAI agents communicate exclusively via the Cortex REST API.
- **Separate distribution.** Cortex and its analyzer/responder images
  are pulled from upstream registries unmodified.
- **Custom analyzers.** Any Kubric-authored analyzers are deployed as
  independent containers with their own licenses and do not derive
  from Cortex source.

## Permitted Operations

| Operation                       | Method          | Compliant |
|---------------------------------|-----------------|-----------|
| Submit analysis jobs via API    | HTTP POST       | Yes       |
| Trigger responders via API      | HTTP POST       | Yes       |
| Import Cortex Python modules    | Python import   | No        |
| Fork and modify Cortex core     | Source mod      | No        |

## Audit Notes

- Cortex API interaction is logged and auditable.
- Review upstream license changes on each version upgrade.
