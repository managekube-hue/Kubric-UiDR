// Atlas Declarative Schema – Kubric UiDR PostgreSQL Database
// Managed by: atlas schema apply --to file://K-DL-PG-005_Atlas_Schema_HCL.hcl
//
// Multi-tenant design: every row-owning table carries a customer_id column.
// Row-Level Security (RLS) policies are enabled per table so that each
// tenant's data is logically isolated even on shared infra.

schema "kubric" {
  comment = "Kubric UiDR operational database – multi-tenant with RLS"
}

// ──────────────────────────────────────────────────────────────────────
// Extension dependencies
// ──────────────────────────────────────────────────────────────────────

// Ensure required PostgreSQL extensions are available.
// Atlas will issue CREATE EXTENSION IF NOT EXISTS for each.

// UUID generation (v7-style or fallback to gen_random_uuid).
// pgcrypto provides gen_random_uuid() on PG < 17.

// ──────────────────────────────────────────────────────────────────────
// Table: customers
// ──────────────────────────────────────────────────────────────────────
table "customers" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "name" {
    type = text
    null = false
  }
  column "slug" {
    type = text
    null = false
  }
  column "domain" {
    type = text
  }
  column "tier" {
    type    = text
    null    = false
    default = "standard"
  }
  column "max_endpoints" {
    type    = integer
    null    = false
    default = 100
  }
  column "contact_email" {
    type = text
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "active" {
    type    = boolean
    null    = false
    default = true
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_customers_slug" {
    columns = [column.slug]
    unique  = true
  }

  index "idx_customers_domain" {
    columns = [column.domain]
    unique  = true
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: contracts
// ──────────────────────────────────────────────────────────────────────
table "contracts" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "contract_type" {
    type    = text
    null    = false
    default = "mssp"
  }
  column "start_date" {
    type = date
    null = false
  }
  column "end_date" {
    type = date
  }
  column "monthly_fee" {
    type = numeric
  }
  column "currency" {
    type    = text
    null    = false
    default = "USD"
  }
  column "sla_response_minutes" {
    type    = integer
    default = 60
  }
  column "sla_resolution_hours" {
    type    = integer
    default = 24
  }
  column "included_hours" {
    type    = numeric
    default = 0
  }
  column "hourly_rate" {
    type    = numeric
    default = 0
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "active" {
    type    = boolean
    null    = false
    default = true
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_contracts_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  index "idx_contracts_customer" {
    columns = [column.customer_id]
  }

  check "chk_contracts_type" {
    expr = "contract_type IN ('mssp', 'project', 'retainer', 'break_fix')"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: invoices
// ──────────────────────────────────────────────────────────────────────
table "invoices" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "contract_id" {
    type = uuid
    null = false
  }
  column "invoice_number" {
    type = text
    null = false
  }
  column "status" {
    type    = text
    null    = false
    default = "draft"
  }
  column "period_start" {
    type = date
    null = false
  }
  column "period_end" {
    type = date
    null = false
  }
  column "subtotal" {
    type    = numeric
    null    = false
    default = 0
  }
  column "tax" {
    type    = numeric
    null    = false
    default = 0
  }
  column "total" {
    type    = numeric
    null    = false
    default = 0
  }
  column "currency" {
    type    = text
    null    = false
    default = "USD"
  }
  column "due_date" {
    type = date
  }
  column "paid_at" {
    type = timestamptz
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_invoices_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  foreign_key "fk_invoices_contract" {
    columns     = [column.contract_id]
    ref_columns = [table.contracts.column.id]
    on_delete   = RESTRICT
  }

  index "idx_invoices_customer" {
    columns = [column.customer_id]
  }

  index "idx_invoices_number" {
    columns = [column.invoice_number]
    unique  = true
  }

  index "idx_invoices_status" {
    columns = [column.status]
  }

  check "chk_invoices_status" {
    expr = "status IN ('draft', 'sent', 'paid', 'overdue', 'void')"
  }

  check "chk_invoices_total" {
    expr = "total >= 0"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: invoice_lines
// ──────────────────────────────────────────────────────────────────────
table "invoice_lines" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "invoice_id" {
    type = uuid
    null = false
  }
  column "description" {
    type = text
    null = false
  }
  column "quantity" {
    type    = numeric
    null    = false
    default = 1
  }
  column "unit_price" {
    type    = numeric
    null    = false
    default = 0
  }
  column "amount" {
    type    = numeric
    null    = false
    default = 0
  }
  column "line_type" {
    type    = text
    null    = false
    default = "service"
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_invoice_lines_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  foreign_key "fk_invoice_lines_invoice" {
    columns     = [column.invoice_id]
    ref_columns = [table.invoices.column.id]
    on_delete   = CASCADE
  }

  index "idx_invoice_lines_invoice" {
    columns = [column.invoice_id]
  }

  check "chk_invoice_lines_type" {
    expr = "line_type IN ('service', 'hours', 'license', 'hardware', 'adjustment')"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: time_entries
// ──────────────────────────────────────────────────────────────────────
table "time_entries" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "contract_id" {
    type = uuid
    null = false
  }
  column "analyst_email" {
    type = text
    null = false
  }
  column "description" {
    type = text
    null = false
  }
  column "date" {
    type = date
    null = false
  }
  column "hours" {
    type = numeric
    null = false
  }
  column "billable" {
    type    = boolean
    null    = false
    default = true
  }
  column "invoice_line_id" {
    type = uuid
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_time_entries_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  foreign_key "fk_time_entries_contract" {
    columns     = [column.contract_id]
    ref_columns = [table.contracts.column.id]
    on_delete   = RESTRICT
  }

  index "idx_time_entries_customer_date" {
    columns = [column.customer_id, column.date]
  }

  index "idx_time_entries_analyst" {
    columns = [column.analyst_email]
  }

  check "chk_time_entries_hours" {
    expr = "hours > 0 AND hours <= 24"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: alerts
// ──────────────────────────────────────────────────────────────────────
table "alerts" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "event_class" {
    type = integer
    null = false
  }
  column "severity" {
    type    = text
    null    = false
    default = "medium"
  }
  column "title" {
    type = text
    null = false
  }
  column "description" {
    type = text
  }
  column "status" {
    type    = text
    null    = false
    default = "open"
  }
  column "assignee" {
    type = text
  }
  column "source" {
    type = text
    null = false
  }
  column "rule_uid" {
    type = text
  }
  column "device_hostname" {
    type = text
  }
  column "device_ip" {
    type = text
  }
  column "mitre_tactics" {
    type = jsonb
  }
  column "mitre_techniques" {
    type = jsonb
  }
  column "observables" {
    type = jsonb
  }
  column "correlation_uid" {
    type = text
  }
  column "acknowledged_at" {
    type = timestamptz
  }
  column "resolved_at" {
    type = timestamptz
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_alerts_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  index "idx_alerts_customer_status" {
    columns = [column.customer_id, column.status]
  }

  index "idx_alerts_severity" {
    columns = [column.severity]
  }

  index "idx_alerts_created" {
    columns = [column.created_at]
  }

  index "idx_alerts_correlation" {
    columns = [column.correlation_uid]
  }

  check "chk_alerts_severity" {
    expr = "severity IN ('info', 'low', 'medium', 'high', 'critical')"
  }

  check "chk_alerts_status" {
    expr = "status IN ('open', 'acknowledged', 'investigating', 'resolved', 'false_positive', 'suppressed')"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: compliance_findings
// ──────────────────────────────────────────────────────────────────────
table "compliance_findings" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "framework" {
    type = text
    null = false
  }
  column "control_uid" {
    type = text
    null = false
  }
  column "control_name" {
    type = text
  }
  column "status" {
    type    = text
    null    = false
    default = "not_checked"
  }
  column "severity" {
    type    = text
    null    = false
    default = "medium"
  }
  column "resource_uid" {
    type = text
  }
  column "resource_type" {
    type = text
  }
  column "device_hostname" {
    type = text
  }
  column "description" {
    type = text
  }
  column "remediation" {
    type = text
  }
  column "evidence" {
    type = jsonb
  }
  column "scan_id" {
    type = uuid
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "checked_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_compliance_findings_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  index "idx_compliance_customer_framework" {
    columns = [column.customer_id, column.framework]
  }

  index "idx_compliance_status" {
    columns = [column.status]
  }

  index "idx_compliance_control" {
    columns = [column.framework, column.control_uid]
  }

  check "chk_compliance_status" {
    expr = "status IN ('pass', 'fail', 'error', 'not_checked', 'exempt')"
  }

  check "chk_compliance_framework" {
    expr = "framework IN ('CIS', 'NIST-800-53', 'PCI-DSS', 'HIPAA', 'SOC2', 'ISO-27001', 'GDPR', 'CMMC')"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: agent_registrations
// ──────────────────────────────────────────────────────────────────────
table "agent_registrations" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "agent_type" {
    type = text
    null = false
  }
  column "hostname" {
    type = text
    null = false
  }
  column "ip" {
    type = text
  }
  column "mac" {
    type = text
  }
  column "platform" {
    type = text
    null = false
  }
  column "os_version" {
    type = text
  }
  column "arch" {
    type    = text
    null    = false
    default = "amd64"
  }
  column "version" {
    type = text
    null = false
  }
  column "status" {
    type    = text
    null    = false
    default = "pending"
  }
  column "labels" {
    type    = jsonb
    default = "'{}'"
  }
  column "config" {
    type    = jsonb
    default = "'{}'"
  }
  column "last_heartbeat" {
    type = timestamptz
  }
  column "cpu_percent" {
    type = numeric
  }
  column "memory_bytes" {
    type = bigint
  }
  column "events_processed" {
    type    = bigint
    default = 0
  }
  column "uptime_seconds" {
    type    = bigint
    default = 0
  }
  column "last_error" {
    type = text
  }
  column "vault_role" {
    type = text
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_agent_registrations_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  index "idx_agents_customer_type" {
    columns = [column.customer_id, column.agent_type]
  }

  index "idx_agents_status" {
    columns = [column.status]
  }

  index "idx_agents_hostname" {
    columns = [column.hostname]
  }

  index "idx_agents_heartbeat" {
    columns = [column.last_heartbeat]
  }

  check "chk_agent_type" {
    expr = "agent_type IN ('coresec', 'netguard', 'perftrace', 'watchdog', 'provisioning')"
  }

  check "chk_agent_status" {
    expr = "status IN ('pending', 'deploying', 'running', 'failed', 'updating', 'stopped', 'degraded')"
  }

  check "chk_agent_platform" {
    expr = "platform IN ('linux', 'windows', 'macos', 'container', 'kubernetes')"
  }
}

// ──────────────────────────────────────────────────────────────────────
// Table: audit_log  (immutable append-only ledger)
// ──────────────────────────────────────────────────────────────────────
table "audit_log" {
  schema = schema.kubric

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "customer_id" {
    type = uuid
    null = false
  }
  column "actor" {
    type = text
    null = false
  }
  column "actor_type" {
    type    = text
    null    = false
    default = "user"
  }
  column "action" {
    type = text
    null = false
  }
  column "resource_type" {
    type = text
    null = false
  }
  column "resource_id" {
    type = text
    null = false
  }
  column "description" {
    type = text
  }
  column "diff" {
    type = jsonb
  }
  column "source_ip" {
    type = text
  }
  column "user_agent" {
    type = text
  }
  column "session_id" {
    type = text
  }
  column "correlation_uid" {
    type = text
  }
  column "blake3_hash" {
    type = bytea
    null = false
  }
  column "prev_hash" {
    type = bytea
  }
  column "metadata" {
    type    = jsonb
    default = "'{}'"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_audit_log_customer" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_delete   = CASCADE
  }

  index "idx_audit_customer_created" {
    columns = [column.customer_id, column.created_at]
  }

  index "idx_audit_actor" {
    columns = [column.actor]
  }

  index "idx_audit_resource" {
    columns = [column.resource_type, column.resource_id]
  }

  index "idx_audit_action" {
    columns = [column.action]
  }

  index "idx_audit_correlation" {
    columns = [column.correlation_uid]
  }

  check "chk_audit_actor_type" {
    expr = "actor_type IN ('user', 'service', 'agent', 'system', 'api_key')"
  }
}
