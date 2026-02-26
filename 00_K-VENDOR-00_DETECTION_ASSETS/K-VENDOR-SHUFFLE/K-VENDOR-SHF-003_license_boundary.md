# K-VENDOR-SHF-003 -- Shuffle License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | AGPL-3.0                                  |
| Boundary    | Process / network isolation required       |

## AGPL-3.0 Obligations

Shuffle is licensed under AGPL-3.0. Any software that incorporates
Shuffle source code and provides network services must also be AGPL.

## Kubric Compliance Strategy

- **No source-level integration.** Kubric does not import, link, or
  embed any Shuffle library, SDK, or source file.
- **Process isolation.** Shuffle runs as a standalone Docker Compose
  stack (frontend, backend, orborus). KAI agents communicate
  exclusively via the Shuffle REST API.
- **Separate distribution.** Shuffle Docker images are pulled from
  upstream registries without modification.
- **Custom apps.** Any Kubric-authored Shuffle apps are independent
  Docker containers with their own licenses, not derived from
  Shuffle source.

## Permitted Operations

| Operation                          | Method          | Compliant |
|------------------------------------|-----------------|-----------|
| Execute workflows via API          | HTTP POST       | Yes       |
| Poll execution status via API      | HTTP GET        | Yes       |
| Upload custom app images           | HTTP POST       | Yes       |
| Import Shuffle Python SDK          | Python import   | No        |
| Embed Shuffle backend in KAI       | Library linking | No        |

## Audit Notes

- Track all Shuffle API endpoints used by KAI agents.
- Custom Shuffle apps must declare their own license headers.
