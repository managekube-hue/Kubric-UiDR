# K-MB-SUB-009 — Billing Usage

> NATS subject mapping reference for metered usage telemetry within the Kubric UIDR platform.

## Subject Pattern

```
kubric.billing.usage.v1
kubric.billing.usage.>          # wildcard for all billing usage sub-subjects
```

Tokens:
| Position | Token       | Description                        |
|----------|-------------|------------------------------------|
| 1        | `kubric`    | Platform root namespace            |
| 2        | `billing`   | Domain — billing and invoicing     |
| 3        | `usage`     | Event type — metered usage record  |
| 4        | `v1`        | Schema version                     |

## Publisher

**All DR modules** — every detection and response module across the platform emits usage records:

| Module    | Usage Type                                      |
|-----------|------------------------------------------------|
| CoreSec   | EDR scan hours, FIM events processed            |
| NetGuard  | NDR traffic bytes inspected, IDS alerts fired   |
| VDR       | Vulnerability scans executed, assets scanned    |
| KIC       | Compliance checks run, policies evaluated       |
| SIDR TI   | TI feed queries, IOC lookups performed          |
| KAI crew  | AI inference tokens consumed, analyst hours     |

Each module publishes on task completion or at a configurable flush interval (default: 60 seconds).

## Consumer(s)

| Consumer                | Runtime        | Role                                              |
|-------------------------|----------------|----------------------------------------------------|
| **KAI-CLERK** (primary) | Python / FastAPI | Aggregates usage, reconciles against contract rate tables, pushes line items to Stripe |
| K-SVC Portal            | TypeScript / Next.js | Displays near-real-time usage dashboard for tenants |

## Payload

**Format:** JSON — OCSF base class extended with custom metering fields.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "service_module": "coresec",
  "usage_type": "edr_scan_hours",
  "quantity": 4.25,
  "unit": "hours",
  "timestamp": "2026-02-26T14:30:00Z",
  "metadata": {
    "region": "us-east-1",
    "contract_id": "ctr_xyz789",
    "billing_period": "2026-02"
  }
}
```

## Fields

| Field              | Type     | Required | Description                                                    |
|--------------------|----------|----------|----------------------------------------------------------------|
| `schema_version`   | string   | yes      | Payload schema version (semver)                                |
| `tenant_id`        | string   | yes      | Unique tenant identifier (`tnt_` prefix)                       |
| `service_module`   | string   | yes      | Originating module: `coresec`, `netguard`, `vdr`, `kic`, `sidr_ti`, `kai` |
| `usage_type`       | string   | yes      | Specific usage metric key (e.g., `edr_scan_hours`, `ndr_bytes_inspected`) |
| `quantity`         | float    | yes      | Numeric quantity of usage consumed                             |
| `unit`             | string   | yes      | Unit of measure: `hours`, `bytes`, `count`, `tokens`, `scans`  |
| `timestamp`        | datetime | yes      | ISO 8601 UTC timestamp of the usage record                     |
| `metadata.region`  | string   | no       | Deployment region for multi-region billing splits              |
| `metadata.contract_id` | string | no    | Associated contract for rate table lookup                      |
| `metadata.billing_period` | string | no | Billing period in `YYYY-MM` format                            |

### Billing Aggregation

KAI-CLERK aggregates usage records by `(tenant_id, service_module, usage_type, billing_period)` and applies tiered pricing from `contract_rate_tables`:

```
contract_rate_tables
├── base_rate        # per-unit cost at standard tier
├── tier_thresholds  # volume breakpoints for discounted pricing
├── overage_rate     # rate applied beyond contracted allocation
└── currency         # ISO 4217 currency code (default: USD)
```

### Stripe Sync

Aggregated line items are pushed to Stripe via the Billing Meter API at the end of each billing period. Mid-cycle usage is available through the K-SVC Portal for tenant self-service visibility.

## JetStream Configuration

```
Stream:          BILLING_USAGE
Subjects:        kubric.billing.usage.>
Storage:         File
Retention:       Limits
Max Age:         90 days
Max Bytes:       10 GB
Replicas:        3
Discard Policy:  Old
Duplicate Window: 2 minutes
```

## Consumer Groups

| Consumer Group     | Deliver Policy | Ack Policy  | Max Deliver | Filter Subject              |
|--------------------|---------------|-------------|-------------|-----------------------------|
| `clerk-aggregator` | All           | Explicit    | 5           | `kubric.billing.usage.>`    |
| `portal-usage`     | Last Per Subj | Explicit    | 3           | `kubric.billing.usage.>`    |

## Example (NATS CLI)

**Publish a usage record:**

```bash
nats pub kubric.billing.usage.v1 '{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "service_module": "coresec",
  "usage_type": "edr_scan_hours",
  "quantity": 4.25,
  "unit": "hours",
  "timestamp": "2026-02-26T14:30:00Z",
  "metadata": {
    "region": "us-east-1",
    "contract_id": "ctr_xyz789",
    "billing_period": "2026-02"
  }
}'
```

**Subscribe to all billing usage events:**

```bash
nats sub "kubric.billing.usage.>"
```

**Create the JetStream stream:**

```bash
nats stream add BILLING_USAGE \
  --subjects "kubric.billing.usage.>" \
  --storage file \
  --retention limits \
  --max-age 90d \
  --max-bytes 10GB \
  --replicas 3 \
  --discard old \
  --dupe-window 2m
```

**Create the clerk-aggregator consumer:**

```bash
nats consumer add BILLING_USAGE clerk-aggregator \
  --deliver all \
  --ack explicit \
  --max-deliver 5 \
  --filter "kubric.billing.usage.>"
```

## Notes

- **Idempotency:** Each usage record should include a deterministic deduplication key derived from `(tenant_id, service_module, usage_type, timestamp)` to prevent double-counting within the 2-minute duplicate window.
- **Rate Table Lookup:** KAI-CLERK loads `contract_rate_tables` from PostgreSQL on startup and caches with a 5-minute TTL. Rate changes mid-cycle are prorated.
- **Audit Trail:** All raw usage records are retained in JetStream for 90 days for dispute resolution and auditing. Aggregated summaries are persisted to PostgreSQL indefinitely.
- **Currency:** All monetary values use the tenant's contracted currency from `contract_rate_tables`. Default is USD.
- **Back-pressure:** If KAI-CLERK falls behind, the JetStream consumer will buffer messages. An alert fires if the pending count exceeds 10,000.
- **Related Subjects:** `kubric.billing.invoice.>` (downstream invoice generation), `kubric.billing.payment.>` (payment confirmation from Stripe webhook).
