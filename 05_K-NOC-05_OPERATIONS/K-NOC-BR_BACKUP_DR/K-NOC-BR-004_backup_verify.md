# K-NOC-BR-004 -- Backup Verification Runbook

**Purpose:** Ensure all Kubric backups (Restic, Kopia, Velero) are valid and restorable.
**SLA Targets:** RPO = 4 hours | RTO = 2 hours

---

## 1. Daily Automated Verification Process

The automated daily check runs at 06:00 UTC via a Temporal workflow (`kubric.backup.daily-verify`).
It performs the following steps for each configured backup repository:

1. **Restic:** `restic check --read-data-subset=5%` (random 5% blob verification)
2. **Kopia:** `kopia snapshot verify --verify-files-percent=10`
3. **Velero:** Compare latest backup status to `Completed` via k8s API
4. **S3 cold storage:** Check lifecycle rule execution logs in MinIO

Results are:
- Posted to Kubric NOC API: `POST /api/v1/incidents` (severity=info on success, critical on failure)
- Written to `/var/log/kubric/backup-verify-$(date +%F).log`
- Archived to `s3://kubric-backups/audit/backup-checks/`

### Trigger manually:
```bash
# Via Temporal CLI
temporal workflow start   --task-queue kubric-backup   --type BackupVerifyWorkflow   --input '{"run_full_check": false, "notify_on_success": true}'

# Or directly via the NOC API
curl -s -X POST http://noc:8083/api/v1/backups/verify   -H "Authorization: Bearer $KUBRIC_API_TOKEN"   -H "Content-Type: application/json"   -d '{"scope": "all", "data_subset_pct": 5}'
```

---

## 2. Manual Verification Commands

### 2.1 Restic
```bash
# Set repo credentials
export RESTIC_REPOSITORY=s3:http://minio:9000/kubric-backups
export RESTIC_PASSWORD_FILE=/etc/restic/password

# List snapshots (most recent 10)
restic snapshots --last 10

# Verify repository integrity (metadata check, fast)
restic check

# Verify repository with random data sampling (reads from S3)
restic check --read-data-subset=10%

# Verify repository with full data read (slow -- use only for quarterly audit)
restic check --read-data

# Restore a specific snapshot to a temp directory (functional test)
SNAP_ID=$(restic snapshots --last 1 --json | jq -r '.[0].short_id')
restic restore "$SNAP_ID" --target /tmp/restic-restore-test --verify
ls -la /tmp/restic-restore-test/var/lib/kubric/
rm -rf /tmp/restic-restore-test
```

### 2.2 Kopia
```bash
# Connect to Kopia repository
kopia repository connect s3   --bucket kubric-kopia   --endpoint minio:9000   --access-key "$MINIO_ACCESS_KEY"   --secret-access-key "$MINIO_SECRET_KEY"   --disable-tls

# List snapshots
kopia snapshot list

# Verify a specific snapshot's file integrity
SNAP_ID=$(kopia snapshot list --json | jq -r '.[0].id')
kopia snapshot verify "$SNAP_ID" --verify-files-percent=20

# Test full restoration to temp dir
kopia restore "$SNAP_ID" /tmp/kopia-restore-test
ls -la /tmp/kopia-restore-test/
rm -rf /tmp/kopia-restore-test
```

### 2.3 Velero (Kubernetes namespace backups)
```bash
# List Velero backups
velero backup get

# Describe latest backup
velero backup describe $(velero backup get -o json | jq -r '.items | sort_by(.metadata.creationTimestamp) | last | .metadata.name')

# Verify backup is complete (status.phase must be Completed)
velero backup get --output table | grep -v Completed || echo "WARNING: backups not all Complete"

# Dry-run restore to separate namespace
velero restore create test-restore-$(date +%s)   --from-backup kubric-daily-$(date +%F)   --namespace-mappings kubric:kubric-verify   --include-namespaces kubric

# Check restore status
velero restore describe test-restore-*
kubectl get all -n kubric-verify
kubectl delete namespace kubric-verify  # cleanup after verify
```

### 2.4 Verify PostgreSQL Backup (pg_dump)
```bash
# Restore pg_dump to a test database
pg_restore -l /var/backups/kubric/postgres-kubric-$(date +%F).dump | head -20

# Full restore to temp DB
createdb kubric_verify
pg_restore -d kubric_verify /var/backups/kubric/postgres-kubric-$(date +%F).dump
psql -d kubric_verify -c "SELECT COUNT(*) FROM tenants;"
psql -d kubric_verify -c "SELECT COUNT(*) FROM findings;"
dropdb kubric_verify
```

---

## 3. SLA Targets

| Metric | Target | Alerting Condition                     |
|--------|--------|----------------------------------------|
| RPO    | 4 hours | Latest snapshot older than 4h         |
| RTO    | 2 hours | Restore drill must complete in < 2h   |
| Backup success rate | 99.9%/month | > 1 failure/month triggers review |
| Data integrity | 100% | Any Restic/Kopia check failure       |

**RPO Alarm Query (Prometheus):**
```promql
# Alert if latest snapshot is older than 4 hours
(time() - kubric_backup_last_success_timestamp_seconds) > 14400
```

---

## 4. Alert Routing: Backup Failure Path

```
Backup job failure
    |
    v
NATS: kubric.incidents stream (severity=critical, source=backup-verify)
    |
    v
Kubric NOC API: POST /api/v1/incidents
    |
    v
NOC Dashboard alert (Grafana panel: "Backup Health")
    |
    v
PagerDuty-equivalent: Alertmanager webhook -> kubric-oncall-webhook:9095
    |-> Routing: receiver=noc-critical, group_wait=30s, repeat_interval=4h
    |
    v
On-call NOC engineer: investigate + run manual verify commands above
    |
    v
Escalation (if RTO breach imminent): team lead + Kubric architect
```

### Alertmanager route snippet for backup failures:
```yaml
route:
  routes:
    - receiver: "noc-critical"
      match:
        severity: critical
        source: backup-verify
      group_wait: 30s
      group_interval: 5m
      repeat_interval: 4h

receivers:
  - name: "noc-critical"
    webhook_configs:
      - url: "http://kubric-oncall-webhook:9095/alert"
        send_resolved: true
        http_config:
          authorization:
            credentials_file: /etc/alertmanager/kubric-token
```

---

## 5. Monthly Backup Audit Checklist

- [ ] Run full Restic `check --read-data` on each repository
- [ ] Complete a timed restore drill (target < 2h RTO)
- [ ] Verify MinIO bucket replication is healthy
- [ ] Confirm Velero schedule is active: `velero schedule get`
- [ ] Check Restic forget/prune ran: `restic snapshots | wc -l` within expected range
- [ ] Review backup-verify logs for any partial failures during the month
- [ ] Update retention policy if storage costs exceeded budget
