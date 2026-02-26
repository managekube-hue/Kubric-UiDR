"""
Kubric KAI Temporal worker.

Runs two workers:
  - kubric-remediation   (RemediationWorkflow activities)
  - kubric-billing       (BillingWorkflow activities)

Start with:
  python -m kai.workers.temporal_worker

Or via the Makefile:
  make kai-worker

Requires temporalio to be installed:
  pip install "kubric-kai[workflows]"
"""

from __future__ import annotations

import asyncio
import os

import structlog

log = structlog.get_logger(__name__)

_TEMPORAL_URL = os.getenv("KUBRIC_TEMPORAL_URL", "temporal:7233")


async def main() -> None:
    try:
        from temporalio.client import Client  # noqa: PLC0415
        from temporalio.worker import Worker  # noqa: PLC0415
    except ImportError:
        log.error("temporal_worker.temporalio_not_installed", hint="pip install 'kubric-kai[workflows]'")
        return

    from kai.workflows.remediation import (  # noqa: PLC0415
        close_finding,
        run_ansible,
        validate_finding,
        verify_remediation,
    )
    from kai.workflows.billing import (  # noqa: PLC0415
        aggregate_usage,
        create_stripe_invoice,
        record_invoice,
    )

    # Decorate activities dynamically (avoids having temporalio as a hard dep)
    from temporalio import activity as _activity  # noqa: PLC0415

    _decorate = _activity.defn

    remediation_activities = [
        _decorate(validate_finding),
        _decorate(run_ansible),
        _decorate(verify_remediation),
        _decorate(close_finding),
    ]
    billing_activities = [
        _decorate(aggregate_usage),
        _decorate(create_stripe_invoice),
        _decorate(record_invoice),
    ]

    client = await Client.connect(_TEMPORAL_URL)
    log.info("temporal_worker.connected", url=_TEMPORAL_URL)

    remediation_worker = Worker(
        client,
        task_queue="kubric-remediation",
        activities=remediation_activities,
    )
    billing_worker = Worker(
        client,
        task_queue="kubric-billing",
        activities=billing_activities,
    )

    log.info("temporal_worker.starting", queues=["kubric-remediation", "kubric-billing"])
    await asyncio.gather(
        remediation_worker.run(),
        billing_worker.run(),
    )


if __name__ == "__main__":
    asyncio.run(main())
