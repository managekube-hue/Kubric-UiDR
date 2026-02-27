"""
K-KAI-TR-004: KISS (Keep It Simple Scoring) Risk Calculator
Combines CVSS base score, EPSS probability, asset criticality, and active
exploitation flag into a composite 0-100 risk score.
"""

import logging
import math

logger = logging.getLogger("K-KAI-TR-004")

# ---------------------------------------------------------------------------
# Grade thresholds  (score → letter grade)
# ---------------------------------------------------------------------------
_GRADE_THRESHOLDS: list[tuple[int, str]] = [
    (90, "F"),  # ≥ 90  → F  (Fail / Act immediately)
    (70, "D"),  # ≥ 70  → D
    (50, "C"),  # ≥ 50  → C
    (30, "B"),  # ≥ 30  → B
    (0,  "A"),  # ≥  0  → A  (Pass / Monitor)
]

# ---------------------------------------------------------------------------
# Recommended action lookup (grade-based)
# ---------------------------------------------------------------------------
_ACTIONS: dict[str, str] = {
    "A": "Monitor – log and review during next business cycle.",
    "B": "Investigate – assign to on-call analyst within 24 hours.",
    "C": "Remediate – patch or mitigate within 72 hours.",
    "D": "Urgent – escalate to incident response within 4 hours.",
    "F": "Critical – initiate emergency response immediately.",
}


class KISSCalculator:
    """
    KISS composite risk scorer.

    Formula
    -------
    weighted_cvss  = (cvss / 10.0) * 40          ; weight 40 %
    weighted_epss  = epss_normalised * 30         ; weight 30 %
    weighted_crit  = ((criticality - 1) / 4) * 20; weight 20 %
    exploitation   = 10 if exploited else 0       ; weight 10 %
    raw_score      = sum of above (0–100)
    final_score    = min(100, round(raw_score))
    """

    def score(
        self,
        cvss: float,
        epss: float,
        criticality: int,
        exploited: bool,
    ) -> dict:
        """
        Compute composite risk score.

        Args:
            cvss:         CVSS v3 base score (0.0–10.0).
            epss:         EPSS probability (0.0–1.0).
            criticality:  Asset criticality tier (1 = low … 5 = critical).
            exploited:    True if the vulnerability has known active exploitation.

        Returns:
            dict with keys: score (int), grade (str), recommended_action (str),
                            breakdown (dict with individual component scores).
        """
        # Clamp inputs to valid ranges
        cvss = max(0.0, min(10.0, float(cvss)))
        epss = max(0.0, min(1.0, float(epss)))
        criticality = max(1, min(5, int(criticality)))

        # Component scores
        cvss_component: float = (cvss / 10.0) * 40.0

        # EPSS uses a log-scaled normalisation so rare but high-probability
        # vulns don't over-dominate; log(1 + epss*10)/log(11) ≈ 0…1
        epss_normalised: float = math.log1p(epss * 10.0) / math.log(11.0)
        epss_component: float = epss_normalised * 30.0

        crit_component: float = ((criticality - 1) / 4.0) * 20.0
        exploitation_component: float = 10.0 if exploited else 0.0

        raw: float = (
            cvss_component + epss_component + crit_component + exploitation_component
        )
        final_score: int = min(100, max(0, round(raw)))

        grade: str = self._score_to_grade(final_score)
        action: str = _ACTIONS[grade]

        logger.debug(
            "KISS score=%d grade=%s [cvss=%.1f epss=%.1f crit=%.1f exploit=%.1f]",
            final_score,
            grade,
            cvss_component,
            epss_component,
            crit_component,
            exploitation_component,
        )

        return {
            "score": final_score,
            "grade": grade,
            "recommended_action": action,
            "breakdown": {
                "cvss_component": round(cvss_component, 2),
                "epss_component": round(epss_component, 2),
                "criticality_component": round(crit_component, 2),
                "exploitation_component": exploitation_component,
                "inputs": {
                    "cvss": cvss,
                    "epss": epss,
                    "criticality": criticality,
                    "exploited": exploited,
                },
            },
        }

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _score_to_grade(score: int) -> str:
        for threshold, grade in _GRADE_THRESHOLDS:
            if score >= threshold:
                return grade
        return "A"
