"""
K-KAI Housekeeper: Criticality Check
Evaluates whether a housekeeping action (patch, remediation, config change)
is safe to apply automatically based on asset criticality and change window rules.
"""
from __future__ import annotations
import json, logging, os
from dataclasses import dataclass
from datetime import datetime, timezone, time as dtime
from enum import Enum
from typing import Dict, List, Optional, Tuple

logger = logging.getLogger(__name__)

class AssetCriticality(int, Enum):
    TIER1 = 5   # Production critical — DBA, financial, identity systems
    TIER2 = 4   # Production supporting — load balancers, caches
    TIER3 = 3   # Staging / pre-prod
    TIER4 = 2   # Dev / test
    TIER5 = 1   # Sandbox / throwaway

class ChangeWindow(str, Enum):
    BUSINESS_HOURS   = "business_hours"    # Mon-Fri 09:00-17:00 UTC
    OFF_HOURS        = "off_hours"         # Mon-Fri 18:00-06:00 UTC + weekends
    MAINTENANCE_WINDOW = "maintenance"     # Sat-Sun 02:00-06:00 UTC
    EMERGENCY        = "emergency"         # Any time, manual override

@dataclass
class CriticalityPolicy:
    criticality: AssetCriticality
    allowed_windows: List[ChangeWindow]
    require_mfa_approval: bool
    auto_rollback_on_failure: bool
    max_concurrent_changes: int

@dataclass
class CriticalityDecision:
    asset_id:     str
    criticality:  AssetCriticality
    action:       str
    approved:     bool
    reason:       str
    required_window: ChangeWindow
    current_window:  ChangeWindow

_POLICIES: Dict[AssetCriticality, CriticalityPolicy] = {
    AssetCriticality.TIER1: CriticalityPolicy(
        criticality=AssetCriticality.TIER1,
        allowed_windows=[ChangeWindow.MAINTENANCE_WINDOW, ChangeWindow.EMERGENCY],
        require_mfa_approval=True,
        auto_rollback_on_failure=True,
        max_concurrent_changes=1,
    ),
    AssetCriticality.TIER2: CriticalityPolicy(
        criticality=AssetCriticality.TIER2,
        allowed_windows=[ChangeWindow.OFF_HOURS, ChangeWindow.MAINTENANCE_WINDOW, ChangeWindow.EMERGENCY],
        require_mfa_approval=True,
        auto_rollback_on_failure=True,
        max_concurrent_changes=3,
    ),
    AssetCriticality.TIER3: CriticalityPolicy(
        criticality=AssetCriticality.TIER3,
        allowed_windows=[ChangeWindow.BUSINESS_HOURS, ChangeWindow.OFF_HOURS, ChangeWindow.MAINTENANCE_WINDOW],
        require_mfa_approval=False,
        auto_rollback_on_failure=True,
        max_concurrent_changes=10,
    ),
    AssetCriticality.TIER4: CriticalityPolicy(
        criticality=AssetCriticality.TIER4,
        allowed_windows=[ChangeWindow.BUSINESS_HOURS, ChangeWindow.OFF_HOURS, ChangeWindow.MAINTENANCE_WINDOW],
        require_mfa_approval=False,
        auto_rollback_on_failure=False,
        max_concurrent_changes=20,
    ),
    AssetCriticality.TIER5: CriticalityPolicy(
        criticality=AssetCriticality.TIER5,
        allowed_windows=[c for c in ChangeWindow],
        require_mfa_approval=False,
        auto_rollback_on_failure=False,
        max_concurrent_changes=50,
    ),
}

def detect_current_window(now: Optional[datetime] = None) -> ChangeWindow:
    now = now or datetime.now(timezone.utc)
    weekday = now.weekday()   # 0=Mon, 6=Sun
    hour    = now.hour

    if weekday in (5, 6):     # Weekend
        if 2 <= hour < 6:
            return ChangeWindow.MAINTENANCE_WINDOW
        return ChangeWindow.OFF_HOURS

    # Weekday
    if 9 <= hour < 17:
        return ChangeWindow.BUSINESS_HOURS
    if 18 <= hour or hour < 6:
        return ChangeWindow.OFF_HOURS
    # 06:00-09:00 and 17:00-18:00 = transition -> off_hours
    return ChangeWindow.OFF_HOURS

def evaluate_change(
    asset_id:    str,
    criticality: AssetCriticality,
    action:      str,
    override:    bool = False,
    now:         Optional[datetime] = None,
) -> CriticalityDecision:
    policy  = _POLICIES[criticality]
    current = detect_current_window(now)

    if override:
        return CriticalityDecision(
            asset_id=asset_id,
            criticality=criticality,
            action=action,
            approved=True,
            reason="Emergency override applied",
            required_window=ChangeWindow.EMERGENCY,
            current_window=current,
        )

    if current in policy.allowed_windows:
        return CriticalityDecision(
            asset_id=asset_id,
            criticality=criticality,
            action=action,
            approved=True,
            reason=f"Action permitted during {current.value} window for Tier{criticality.value}",
            required_window=policy.allowed_windows[0],
            current_window=current,
        )

    best_window = policy.allowed_windows[0]
    return CriticalityDecision(
        asset_id=asset_id,
        criticality=criticality,
        action=action,
        approved=False,
        reason=(
            f"Tier{criticality.value} asset cannot be changed during {current.value}. "
            f"Next allowed window: {best_window.value}"
        ),
        required_window=best_window,
        current_window=current,
    )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    for tier in AssetCriticality:
        decision = evaluate_change("asset-001", tier, "patch_apply")
        print(json.dumps({
            "tier":     tier.name,
            "approved": decision.approved,
            "reason":   decision.reason,
        }, indent=2))
