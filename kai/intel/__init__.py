"""
kai.intel — Threat Intelligence Feed Ingestion Pipeline (L3-4)

Async feed pullers for:
  1. NVD CVE Feed        (hourly)
  2. CISA KEV            (daily)
  3. FIRST EPSS          (daily)
  4. AlienVault OTX      (15 min)
  5. AbuseIPDB           (30 min)
  6. IPSum               (1 hr)
  7. PyMISP              (1 hr)
"""

from kai.intel.ti_feeds import TIFeedManager

__all__ = ["TIFeedManager"]
