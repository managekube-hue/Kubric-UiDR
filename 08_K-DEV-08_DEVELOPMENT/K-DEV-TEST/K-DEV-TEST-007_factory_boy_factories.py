"""
factory_boy factories for Kubric data models.
Module: K-DEV-TEST-007

Install: pip install factory_boy faker
"""
from __future__ import annotations

import random
from datetime import datetime, timezone
from uuid import uuid4

import factory
from factory import Factory, LazyFunction, LazyAttribute, SubFactory, Iterator, List, Faker


# ---------------------------------------------------------------------------
# TenantFactory
# ---------------------------------------------------------------------------
class TenantFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    name = Faker("company")
    email = Faker("company_email")
    plan = Iterator(["starter", "growth", "professional", "enterprise", "unlimited"])
    status = "active"
    stripe_customer_id = LazyFunction(lambda: f"cus_{uuid4().hex[:14]}")
    hle_units = LazyFunction(lambda: round(random.uniform(1, 50), 2))
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    updated_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())


# ---------------------------------------------------------------------------
# UserFactory
# ---------------------------------------------------------------------------
class UserFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    email = Faker("email")
    full_name = Faker("name")
    role = Iterator([
        "kubric:admin",
        "kubric:analyst",
        "kubric:readonly",
        "kubric:responder",
        "kubric:hunter",
    ])
    is_active = True
    mfa_enabled = LazyFunction(lambda: random.choice([True, False]))
    last_login_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())


# ---------------------------------------------------------------------------
# AssetFactory
# ---------------------------------------------------------------------------
class AssetFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    hostname = Faker("hostname")
    fqdn = LazyAttribute(lambda o: f"{o.hostname}.kubric.local")
    ip_address = Faker("ipv4_private")
    os_family = Iterator(["linux", "windows", "macos", "cloud_vm", "container"])
    os_version = Iterator(["Ubuntu 22.04", "Windows Server 2022", "RHEL 9", "Debian 12"])
    asset_type = Iterator(["server", "workstation", "cloud_vm", "container"])
    criticality_tier = Iterator([1, 2, 3, 4, 5])
    cloud_provider = Iterator(["aws", "azure", "gcp", "on_prem"])
    is_active = True
    last_seen_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    updated_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    tags = LazyFunction(lambda: {"env": random.choice(["prod", "staging", "dev"])})


# ---------------------------------------------------------------------------
# AlertFactory
# ---------------------------------------------------------------------------
class AlertFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    severity = Iterator(["critical", "high", "medium", "low", "info"])
    title = Faker("sentence", nb_words=8)
    description = Faker("paragraph")
    status = Iterator(["open", "in_progress", "resolved", "closed", "false_positive"])
    source = Iterator(["edr", "siem", "vuln_scanner", "kai_ml", "nids"])
    asset_id = LazyFunction(lambda: str(uuid4()))
    cve_ids = LazyFunction(
        lambda: [f"CVE-{random.randint(2020, 2025)}-{random.randint(1000, 99999)}"]
        if random.random() > 0.5
        else []
    )
    ocsf_class_uid = Iterator([1007, 4001, 3002, 2001, 1001])
    ssvc_outcome = Iterator(["DEFER", "SCHEDULED", "OUT_OF_CYCLE", "IMMEDIATE", ""])
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    closed_at = None


# ---------------------------------------------------------------------------
# VulnFactory
# ---------------------------------------------------------------------------
class VulnFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    cve_id = LazyFunction(
        lambda: f"CVE-{random.randint(2020, 2025)}-{random.randint(1000, 99999)}"
    )
    cvss_v3 = LazyFunction(lambda: round(random.uniform(4.0, 10.0), 1))
    cvss_vector = LazyFunction(
        lambda: f"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
    )
    epss_score = LazyFunction(lambda: round(random.uniform(0.001, 0.95), 4))
    exploit_available = LazyFunction(lambda: random.choice([True, False]))
    patch_available = LazyFunction(lambda: random.choice([True, False]))
    affected_package = Faker("word")
    affected_version = LazyFunction(lambda: f"{random.randint(1,9)}.{random.randint(0,20)}.{random.randint(0,5)}")
    fixed_version = LazyFunction(lambda: f"{random.randint(1,9)}.{random.randint(0,20)}.{random.randint(0,5)}")
    description = Faker("sentence", nb_words=15)
    published_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    severity = Iterator(["CRITICAL", "HIGH", "MEDIUM", "LOW"])


# ---------------------------------------------------------------------------
# TicketFactory
# ---------------------------------------------------------------------------
class TicketFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    title = Faker("sentence", nb_words=6)
    description = Faker("paragraph")
    priority = Iterator(["P1", "P2", "P3", "P4"])
    state = Iterator(["open", "in_progress", "pending_review", "resolved", "closed"])
    assignee_id = LazyFunction(lambda: str(uuid4()))
    alert_id = LazyFunction(lambda: str(uuid4()))
    sla_breach_at = None
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    updated_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    resolved_at = None


# ---------------------------------------------------------------------------
# PatchJobFactory
# ---------------------------------------------------------------------------
class PatchJobFactory(Factory):
    class Meta:
        model = dict

    id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    asset_id = LazyFunction(lambda: str(uuid4()))
    status = Iterator(["pending", "scheduled", "running", "completed", "failed", "cancelled"])
    cve_ids = LazyFunction(
        lambda: [
            f"CVE-{random.randint(2020, 2025)}-{random.randint(1000, 99999)}"
            for _ in range(random.randint(1, 5))
        ]
    )
    patch_packages = LazyFunction(
        lambda: [f"pkg-{i}" for i in range(random.randint(1, 8))]
    )
    scheduled_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())
    started_at = None
    completed_at = None
    failure_reason = None
    initiated_by = Iterator(["kai_agent", "manual", "policy"])
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())


# ---------------------------------------------------------------------------
# AgentDecisionFactory
# ---------------------------------------------------------------------------
class AgentDecisionFactory(Factory):
    class Meta:
        model = dict

    decision_id = LazyFunction(lambda: str(uuid4()))
    tenant_id = LazyAttribute(lambda _: str(uuid4()))
    correlation_id = LazyFunction(lambda: str(uuid4()))
    agent_persona = Iterator(["triage", "analyst", "hunter", "responder", "patch"])
    decision_type = Iterator(["triage", "escalate", "close", "remediate", "defer"])
    decision_rationale = Faker("paragraph")
    confidence_score = LazyFunction(lambda: round(random.uniform(0.5, 1.0), 3))
    model_name = Iterator(["claude-sonnet-4-5", "claude-opus-4-5", "gpt-4o"])
    model_version = "20251101"
    execution_ms = LazyFunction(lambda: random.randint(200, 8000))
    token_count_in = LazyFunction(lambda: random.randint(500, 10000))
    token_count_out = LazyFunction(lambda: random.randint(100, 3000))
    ssvc_outcome = Iterator(["DEFER", "SCHEDULED", "OUT_OF_CYCLE", "IMMEDIATE"])
    epss_score = LazyFunction(lambda: round(random.uniform(0.001, 0.95), 4))
    cvss_score = LazyFunction(lambda: round(random.uniform(4.0, 10.0), 1))
    created_at = LazyFunction(lambda: datetime.now(timezone.utc).isoformat())


# ---------------------------------------------------------------------------
# Convenience: batch
# ---------------------------------------------------------------------------
def build_batch(factory_cls: type, count: int, **kwargs) -> list[dict]:
    """Return a list of `count` factory-built dicts."""
    return [factory_cls.build(**kwargs) for _ in range(count)]


if __name__ == "__main__":
    import json

    print("=== TenantFactory (3) ===")
    print(json.dumps(build_batch(TenantFactory, 3), indent=2))

    print("\n=== AlertFactory (5) ===")
    tenant_id = str(uuid4())
    alerts = [AlertFactory.build(tenant_id=tenant_id) for _ in range(5)]
    print(json.dumps(alerts, indent=2))

    print("\n=== VulnFactory (3) ===")
    print(json.dumps(build_batch(VulnFactory, 3), indent=2))
