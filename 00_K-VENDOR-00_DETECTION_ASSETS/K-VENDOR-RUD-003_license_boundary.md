# K-VENDOR-RUD-003 -- Rudder License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | GPL-3.0 (server) / Apache-2.0 (agent)     |
| Boundary    | Server: API-only; Agent: direct OK        |

## License Split

Rudder has a dual-license model:
- **Rudder Server (webapp, relay):** GPL-3.0 -- requires process isolation.
- **Rudder Agent (cf-agent based):** Apache-2.0 -- may be directly
  integrated or bundled.

## Kubric Compliance Strategy

- **Server interaction is API-only.** Kubric does not import, link, or
  embed any Rudder server code. KAI-DEPLOY and KAI-RISK interact
  exclusively via the Rudder REST API.
- **Agent deployment is direct.** The Apache-2.0 licensed Rudder agent
  is deployed on managed nodes by Watchdog. This does not trigger
  copyleft obligations.
- **Separate distribution.** Rudder server Docker images are pulled
  from Normation's official registry unmodified.

## Permitted Operations

| Operation                          | Component | Method      | Compliant |
|------------------------------------|-----------|-------------|-----------|
| Query compliance via REST API      | Server    | HTTP GET    | Yes       |
| Read node inventory via REST API   | Server    | HTTP GET    | Yes       |
| Deploy Rudder agent on endpoints   | Agent     | Package     | Yes       |
| Import Rudder server Java classes  | Server    | JVM import  | No        |
| Embed Rudder relay in Kubric       | Server    | Linking     | No        |

## Audit Notes

- Track Rudder server API usage in architecture decision records.
- Agent version updates are managed by Watchdog via TUF manifests.
