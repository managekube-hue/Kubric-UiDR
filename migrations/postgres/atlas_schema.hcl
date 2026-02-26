# K-DL-PG-005 — Atlas Schema HCL
# Declarative PostgreSQL schema management via Atlas (ariga.io/atlas)
# atlas schema apply --url "postgres://..." --to "file://schema.hcl"

schema "public" {}

table "tenants" {
  schema = schema.public
  column "id" {
    type = uuid
    default = sql("gen_random_uuid()")
  }
  column "name" {
    type = text
  }
  column "slug" {
    type = text
  }
  column "tier" {
    type = text
    default = "professional"
  }
  column "stripe_customer_id" {
    type = text
    null = true
  }
  column "created_at" {
    type = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_tenants_slug" {
    columns = [column.slug]
    unique  = true
  }
}

table "assets" {
  schema = schema.public
  column "id" {
    type = uuid
    default = sql("gen_random_uuid()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "hostname" {
    type = text
  }
  column "os" {
    type = text
    null = true
  }
  column "agent_version" {
    type = text
    null = true
  }
  column "ip_addresses" {
    type = sql("text[]")
    null = true
  }
  column "last_seen" {
    type = timestamptz
    null = true
  }
  column "health_score" {
    type = integer
    default = 0
  }
  column "created_at" {
    type = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
  }
  index "idx_assets_tenant" {
    columns = [column.tenant_id]
  }
}

table "contract_rate_tables" {
  schema = schema.public
  column "id" {
    type = uuid
    default = sql("gen_random_uuid()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "contract_id" {
    type = uuid
  }
  column "effective_from" {
    type = date
  }
  column "effective_to" {
    type = date
    null = true
  }
  column "rate_type" {
    type = text
  }
  column "service_module" {
    type = text
  }
  column "tier" {
    type = text
    default = "professional"
  }
  column "unit_price_cents" {
    type = integer
  }
  column "included_units" {
    type = integer
    default = 0
  }
  column "overage_price_cents" {
    type = integer
    default = 0
  }
  column "currency" {
    type = text
    default = "USD"
  }
  column "created_at" {
    type = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
  }
  index "idx_rate_active" {
    columns = [column.tenant_id, column.service_module, column.tier]
  }
}

table "oscal_controls" {
  schema = schema.public
  column "id" {
    type = uuid
    default = sql("gen_random_uuid()")
  }
  column "catalog_id" {
    type = text
  }
  column "control_id" {
    type = text
  }
  column "title" {
    type = text
  }
  column "family" {
    type = text
  }
  column "description" {
    type = text
  }
  column "baseline_impact" {
    type = sql("text[]")
    null = true
  }
  column "created_at" {
    type = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_oscal_unique" {
    columns = [column.catalog_id, column.control_id]
    unique  = true
  }
}

table "tenant_control_status" {
  schema = schema.public
  column "id" {
    type = uuid
    default = sql("gen_random_uuid()")
  }
  column "tenant_id" {
    type = uuid
  }
  column "control_id" {
    type = uuid
  }
  column "status" {
    type = text
    default = "not_implemented"
  }
  column "implementation_description" {
    type = text
    null = true
  }
  column "updated_at" {
    type = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_tenant" {
    columns     = [column.tenant_id]
    ref_columns = [table.tenants.column.id]
  }
  foreign_key "fk_control" {
    columns     = [column.control_id]
    ref_columns = [table.oscal_controls.column.id]
  }
  index "idx_tenant_control_unique" {
    columns = [column.tenant_id, column.control_id]
    unique  = true
  }
}
