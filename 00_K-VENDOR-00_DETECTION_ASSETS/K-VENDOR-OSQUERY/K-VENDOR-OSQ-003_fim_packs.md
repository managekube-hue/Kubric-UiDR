# K-VENDOR-OSQ-003 -- osquery FIM Packs

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | File integrity monitoring via osquery          |
| Format      | osquery FIM configuration (JSON packs)          |
| Consumer    | CoreSec agent, KAI-TRIAGE                      |

## Purpose

osquery file integrity monitoring packs that detect unauthorized
changes to critical system files, configuration, and binaries.
CoreSec reads FIM events from osquery's `file_events` table.

## Monitored Paths

| Platform | Monitored Directories                          |
|----------|-------------------------------------------------|
| Linux    | `/etc/`, `/usr/bin/`, `/usr/sbin/`, `/boot/`    |
| Linux    | `/var/spool/cron/`, `/root/.ssh/`               |
| Windows  | `C:\Windows\System32\`, `C:\Windows\SysWOW64\`  |
| Windows  | `C:\Users\*\AppData\Roaming\Microsoft\Windows\Start Menu\` |
| macOS    | `/Library/LaunchDaemons/`, `/usr/local/bin/`     |

## FIM Event Fields

| Field          | Description                          |
|----------------|--------------------------------------|
| target_path    | Full path of modified file           |
| category       | created / modified / deleted         |
| action         | ATTRIBUTES_MODIFIED, UPDATED, etc.   |
| md5 / sha256   | Hash of file after change            |
| uid / gid      | Ownership at time of event           |
| time           | Epoch timestamp of change            |

## Integration Flow

1. osqueryd monitors configured paths using inotify/FSEvents/ETW.
2. `file_events` are written to the osquery results log.
3. CoreSec tails the results log and publishes FIM events to NATS.
4. KAI-TRIAGE correlates FIM events with process creation telemetry
   to distinguish legitimate changes from suspicious modifications.

## Notes

- FIM pack schedules are configured independently from IR packs.
- Hash computation is enabled only for files under 50MB.
