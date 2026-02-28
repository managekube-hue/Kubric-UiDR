# K-VENDOR-THV-004 -- TheHive License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | AGPL-3.0                                  |
| Boundary    | Process / network isolation required       |

## AGPL-3.0 Obligations

TheHive is licensed under AGPL-3.0. Network interaction alone does not
trigger copyleft obligations, but any incorporation of TheHive source
code into Kubric would require Kubric to be released under AGPL.

## Kubric Compliance Strategy

- **No source-level integration.** Kubric does not import, link, or
  embed any TheHive library, module, or source file.
- **Process isolation.** TheHive runs as an independent container.
  All Kubric components interact via the documented REST API.
- **API boundary.** KAI-TRIAGE, KAI-KEEPER, and KAI-COMM use only
  standard HTTP requests against TheHive's `/api/v1/` endpoints.
- **Separate distribution.** TheHive Docker images are pulled from
  StrangeBee's official registry without modification.

## Permitted Operations

| Operation                       | Method          | Compliant |
|---------------------------------|-----------------|-----------|
| Create alerts via API           | HTTP POST       | Yes       |
| Manage cases via API            | HTTP CRUD       | Yes       |
| Read observables via API        | HTTP GET        | Yes       |
| Import TheHive4py library       | Python import   | No        |
| Embed TheHive query engine      | Library linking | No        |

## Audit Notes

- Document all API endpoints consumed by KAI agents.
- TheHive4py (the official Python client) is AGPL-licensed and must
  not be imported. Use plain HTTP requests instead.
