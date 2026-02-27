"""
K-KAI Risk: SSVC Decision Tree
Implements the CISA SSVC v2 decision tree for vulnerability prioritization.
Inputs: exploitation status, technical impact, automatable, mission prevalence, public safety impact.
Outputs: DEFER / SCHEDULED / OUT-OF-CYCLE / IMMEDIATE
"""
from __future__ import annotations
import json, logging
from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Tuple

logger = logging.getLogger(__name__)

# ── SSVC decision factors ─────────────────────────────────────────
class ExploitationStatus(str, Enum):
    NONE       = "none"         # no public exploit
    POC        = "poc"          # proof-of-concept
    ACTIVE     = "active"       # actively exploited in the wild

class Automatable(str, Enum):
    NO  = "no"
    YES = "yes"

class TechnicalImpact(str, Enum):
    PARTIAL = "partial"
    TOTAL   = "total"

class MissionAndWellbeing(str, Enum):
    LOW      = "low"
    MEDIUM   = "medium"
    HIGH     = "high"

class SsvcOutcome(str, Enum):
    DEFER         = "DEFER"
    SCHEDULED     = "SCHEDULED"
    OUT_OF_CYCLE  = "OUT_OF_CYCLE"
    IMMEDIATE     = "IMMEDIATE"

@dataclass
class SsvcInput:
    cve:                   str
    exploitation:          ExploitationStatus
    automatable:           Automatable
    technical_impact:      TechnicalImpact
    mission_and_wellbeing: MissionAndWellbeing

@dataclass
class SsvcDecision:
    cve:          str
    outcome:      SsvcOutcome
    rationale:    str
    factors:      Dict[str, str]

# ── Decision table (CISA SSVC v2.1) ──────────────────────────────
# Key: (exploitation, automatable, technical_impact, mission_wellbeing)
# Reference: https://certcc.github.io/SSVC/

_DECISION_TABLE: Dict[Tuple[str, str, str, str], SsvcOutcome] = {
    # exploitation=none
    ("none", "no",  "partial", "low"):    SsvcOutcome.DEFER,
    ("none", "no",  "partial", "medium"): SsvcOutcome.DEFER,
    ("none", "no",  "partial", "high"):   SsvcOutcome.SCHEDULED,
    ("none", "no",  "total",   "low"):    SsvcOutcome.DEFER,
    ("none", "no",  "total",   "medium"): SsvcOutcome.SCHEDULED,
    ("none", "no",  "total",   "high"):   SsvcOutcome.SCHEDULED,
    ("none", "yes", "partial", "low"):    SsvcOutcome.DEFER,
    ("none", "yes", "partial", "medium"): SsvcOutcome.SCHEDULED,
    ("none", "yes", "partial", "high"):   SsvcOutcome.SCHEDULED,
    ("none", "yes", "total",   "low"):    SsvcOutcome.SCHEDULED,
    ("none", "yes", "total",   "medium"): SsvcOutcome.SCHEDULED,
    ("none", "yes", "total",   "high"):   SsvcOutcome.OUT_OF_CYCLE,

    # exploitation=poc
    ("poc",  "no",  "partial", "low"):    SsvcOutcome.SCHEDULED,
    ("poc",  "no",  "partial", "medium"): SsvcOutcome.SCHEDULED,
    ("poc",  "no",  "partial", "high"):   SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "no",  "total",   "low"):    SsvcOutcome.SCHEDULED,
    ("poc",  "no",  "total",   "medium"): SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "no",  "total",   "high"):   SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "yes", "partial", "low"):    SsvcOutcome.SCHEDULED,
    ("poc",  "yes", "partial", "medium"): SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "yes", "partial", "high"):   SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "yes", "total",   "low"):    SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "yes", "total",   "medium"): SsvcOutcome.OUT_OF_CYCLE,
    ("poc",  "yes", "total",   "high"):   SsvcOutcome.IMMEDIATE,

    # exploitation=active
    ("active", "no",  "partial", "low"):    SsvcOutcome.OUT_OF_CYCLE,
    ("active", "no",  "partial", "medium"): SsvcOutcome.OUT_OF_CYCLE,
    ("active", "no",  "partial", "high"):   SsvcOutcome.IMMEDIATE,
    ("active", "no",  "total",   "low"):    SsvcOutcome.OUT_OF_CYCLE,
    ("active", "no",  "total",   "medium"): SsvcOutcome.IMMEDIATE,
    ("active", "no",  "total",   "high"):   SsvcOutcome.IMMEDIATE,
    ("active", "yes", "partial", "low"):    SsvcOutcome.OUT_OF_CYCLE,
    ("active", "yes", "partial", "medium"): SsvcOutcome.IMMEDIATE,
    ("active", "yes", "partial", "high"):   SsvcOutcome.IMMEDIATE,
    ("active", "yes", "total",   "low"):    SsvcOutcome.IMMEDIATE,
    ("active", "yes", "total",   "medium"): SsvcOutcome.IMMEDIATE,
    ("active", "yes", "total",   "high"):   SsvcOutcome.IMMEDIATE,
}

def decide(inp: SsvcInput) -> SsvcDecision:
    key = (
        inp.exploitation.value,
        inp.automatable.value,
        inp.technical_impact.value,
        inp.mission_and_wellbeing.value,
    )
    outcome = _DECISION_TABLE.get(key, SsvcOutcome.SCHEDULED)

    rationale = (
        f"CVE {inp.cve}: exploitation={inp.exploitation.value}, "
        f"automatable={inp.automatable.value}, impact={inp.technical_impact.value}, "
        f"mission={inp.mission_and_wellbeing.value} -> {outcome.value}"
    )
    return SsvcDecision(
        cve=inp.cve,
        outcome=outcome,
        rationale=rationale,
        factors={k: v for k, v in [
            ("exploitation",          inp.exploitation.value),
            ("automatable",           inp.automatable.value),
            ("technical_impact",      inp.technical_impact.value),
            ("mission_and_wellbeing", inp.mission_and_wellbeing.value),
        ]},
    )

def batch_decide(inputs: List[SsvcInput]) -> List[SsvcDecision]:
    return [decide(i) for i in inputs]

# ── entrypoint ────────────────────────────────────────────────────
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    demo = [
        SsvcInput("CVE-2021-44228", ExploitationStatus.ACTIVE, Automatable.YES,
                  TechnicalImpact.TOTAL,   MissionAndWellbeing.HIGH),
        SsvcInput("CVE-2023-12345", ExploitationStatus.POC,    Automatable.NO,
                  TechnicalImpact.PARTIAL, MissionAndWellbeing.MEDIUM),
        SsvcInput("CVE-2020-99999", ExploitationStatus.NONE,   Automatable.NO,
                  TechnicalImpact.PARTIAL, MissionAndWellbeing.LOW),
    ]
    for d in batch_decide(demo):
        print(json.dumps(vars(d), indent=2))
