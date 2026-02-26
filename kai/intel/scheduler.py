"""
kai/intel/scheduler.py — APScheduler-backed TI feed scheduler (L3-4)

Schedules each TI feed at its target cadence using APScheduler
BlockingScheduler (synchronous, suitable for a dedicated subprocess).

Cadences
--------
  NVD CVE          — hourly   (cron: every :00)
  CISA KEV         — daily    (cron: 06:00 UTC)
  FIRST EPSS       — daily    (cron: 06:30 UTC)
  AlienVault OTX   — 15 min
  AbuseIPDB        — 30 min
  IPSum            — hourly
  MISP             — hourly   (staggered at :05)

Usage
-----
  # As a standalone process:
  python -m kai.intel.scheduler

  # Or import and start programmatically:
  from kai.intel.scheduler import TIScheduler
  TIScheduler().start()
"""

from __future__ import annotations

import logging
import signal
import sys

from kai.intel.ti_feeds import TIFeedManager

logger = logging.getLogger(__name__)


class TIScheduler:
    """
    Wraps APScheduler to run TI pulls on their defined cadences.

    The scheduler runs in the calling thread (BlockingScheduler).
    Call start() to enter the event loop; it blocks until SIGINT/SIGTERM.
    """

    def __init__(self, manager: TIFeedManager | None = None) -> None:
        self._mgr = manager or TIFeedManager()

    def _build_scheduler(self) -> "BlockingScheduler":  # type: ignore[name-defined]  # noqa: F821
        try:
            from apscheduler.schedulers.blocking import BlockingScheduler  # type: ignore[import]
            from apscheduler.triggers.cron import CronTrigger  # type: ignore[import]
            from apscheduler.triggers.interval import IntervalTrigger  # type: ignore[import]
        except ImportError as exc:
            raise RuntimeError(
                "APScheduler not installed.  "
                "Run: pip install 'apscheduler>=3.10,<4'"
            ) from exc

        scheduler = BlockingScheduler(timezone="UTC")

        # Feed 1 — NVD CVE (hourly at :00)
        scheduler.add_job(
            self._mgr.pull_nvd,
            trigger=CronTrigger(minute=0),
            id="nvd",
            name="NVD CVE feed",
            replace_existing=True,
            misfire_grace_time=300,
        )

        # Feed 2 — CISA KEV (daily at 06:00 UTC)
        scheduler.add_job(
            self._mgr.pull_cisa_kev,
            trigger=CronTrigger(hour=6, minute=0),
            id="cisa_kev",
            name="CISA KEV feed",
            replace_existing=True,
            misfire_grace_time=600,
        )

        # Feed 3 — FIRST EPSS (daily at 06:30 UTC — after KEV)
        scheduler.add_job(
            self._mgr.pull_epss,
            trigger=CronTrigger(hour=6, minute=30),
            id="epss",
            name="FIRST EPSS feed",
            replace_existing=True,
            misfire_grace_time=600,
        )

        # Feed 4 — AlienVault OTX (every 15 minutes)
        scheduler.add_job(
            self._mgr.pull_otx,
            trigger=IntervalTrigger(minutes=15),
            id="otx",
            name="AlienVault OTX feed",
            replace_existing=True,
            misfire_grace_time=120,
        )

        # Feed 5 — AbuseIPDB (every 30 minutes)
        scheduler.add_job(
            self._mgr.pull_abuseipdb,
            trigger=IntervalTrigger(minutes=30),
            id="abuseipdb",
            name="AbuseIPDB feed",
            replace_existing=True,
            misfire_grace_time=180,
        )

        # Feed 6 — IPSum (hourly at :10 — staggered)
        scheduler.add_job(
            self._mgr.pull_ipsum,
            trigger=CronTrigger(minute=10),
            id="ipsum",
            name="IPSum feed",
            replace_existing=True,
            misfire_grace_time=300,
        )

        # Feed 7 — MISP (hourly at :05 — staggered)
        scheduler.add_job(
            self._mgr.pull_misp,
            trigger=CronTrigger(minute=5),
            id="misp",
            name="MISP feed",
            replace_existing=True,
            misfire_grace_time=300,
        )

        return scheduler

    def start(self) -> None:
        """Start the blocking scheduler.  Does not return until shutdown."""
        scheduler = self._build_scheduler()

        # Graceful shutdown on SIGTERM (e.g. from Kubernetes)
        def _shutdown(signum: int, _frame: object) -> None:
            logger.info("TIScheduler: received signal %d — shutting down", signum)
            scheduler.shutdown(wait=False)

        signal.signal(signal.SIGTERM, _shutdown)
        if sys.platform != "win32":
            signal.signal(signal.SIGHUP, _shutdown)

        logger.info(
            "TIScheduler: starting with %d jobs (manager=%r)",
            len(scheduler.get_jobs()),
            self._mgr,
        )

        # Run all feeds immediately on startup so first-boot populates the DB
        logger.info("TIScheduler: running initial pull of all feeds...")
        results = self._mgr.pull_all()
        for feed_name, count in results.items():
            logger.info("  %-15s %d records", feed_name, count)

        try:
            scheduler.start()
        except (KeyboardInterrupt, SystemExit):
            logger.info("TIScheduler: stopped")


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    TIScheduler().start()


if __name__ == "__main__":
    main()
