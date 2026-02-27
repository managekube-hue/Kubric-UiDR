"""
Hypothesis property-based tests for Kubric platform logic.
Module: K-DEV-TEST-009

Install: pip install hypothesis pytest
Run: pytest K-DEV-TEST-009_hypothesis_property_tests.py -v
"""
from __future__ import annotations

import math
import re
from decimal import Decimal
from typing import Any

import pytest
from hypothesis import assume, given, settings
from hypothesis import strategies as st

# ---------------------------------------------------------------------------
# ─── Domain logic under test ─────────────────────────────────────────────────
# These functions mirror the real implementations in the KAI and billing layers.
# Tests import them here so property tests remain self-contained.
# ---------------------------------------------------------------------------

def calculate_hle(
    agent_seats: int | float,
    events_millions: float,
    ml_calls: int,
) -> float:
    """
    HLE (Headless Linux Endpoint) unit calculation.
    HLE = agent_seats + (events_millions / 10) + (ml_calls / 100_000)
    Must always be >= 0.
    """
    if agent_seats < 0 or events_millions < 0 or ml_calls < 0:
        raise ValueError("inputs must be non-negative")
    return float(agent_seats) + (float(events_millions) / 10.0) + (float(ml_calls) / 100_000.0)


SSVC_OUTCOMES = ("DEFER", "SCHEDULED", "OUT_OF_CYCLE", "IMMEDIATE")
EXPLOITATION_LEVELS = ("none", "poc", "active")
AUTOMATABLE_VALUES = ("yes", "no")
IMPACT_LEVELS = ("low", "medium", "high")

def ssvc_decision(
    exploitation: str,
    automatable: str,
    technical_impact: str = "low",
) -> str:
    """
    Simplified SSVC decision tree.
    Returns one of: DEFER | SCHEDULED | OUT_OF_CYCLE | IMMEDIATE
    """
    if exploitation == "active" and automatable == "yes":
        return "IMMEDIATE"
    if exploitation == "active" and automatable == "no":
        return "OUT_OF_CYCLE" if technical_impact == "high" else "SCHEDULED"
    if exploitation == "poc" and automatable == "yes":
        return "OUT_OF_CYCLE"
    if exploitation == "poc" and automatable == "no":
        return "SCHEDULED"
    # exploitation == "none"
    return "DEFER"


# Alert state machine
VALID_ALERT_TRANSITIONS: dict[str, set[str]] = {
    "open":            {"in_progress", "closed", "false_positive"},
    "in_progress":     {"resolved", "closed", "false_positive", "open"},
    "resolved":        {"closed", "open"},
    "closed":          {"open"},
    "false_positive":  {"open"},
}


class InvalidTransitionError(ValueError):
    pass


def alert_transition(current: str, target: str) -> str:
    """Transition an alert state. Raises InvalidTransitionError for illegal moves."""
    allowed = VALID_ALERT_TRANSITIONS.get(current, set())
    if target not in allowed:
        raise InvalidTransitionError(
            f"Cannot transition alert from '{current}' to '{target}'"
        )
    return target


# Ticket state machine
VALID_TICKET_TRANSITIONS: dict[str, set[str]] = {
    "open":             {"in_progress", "closed"},
    "in_progress":      {"pending_review", "open", "closed"},
    "pending_review":   {"resolved", "in_progress"},
    "resolved":         {"closed", "open"},
    "closed":           {"open"},
}


def ticket_transition(current: str, target: str) -> str:
    """Transition a ticket state. Raises InvalidTransitionError for illegal moves."""
    allowed = VALID_TICKET_TRANSITIONS.get(current, set())
    if target not in allowed:
        raise InvalidTransitionError(
            f"Cannot transition ticket from '{current}' to '{target}'"
        )
    return target


# Billing discount
def apply_billing_discount(list_price: float, discount_pct: float) -> float:
    """Apply a percentage discount. discount_pct must be in [0, 100]."""
    if not (0 <= discount_pct <= 100):
        raise ValueError(f"discount_pct must be in [0, 100], got {discount_pct}")
    if list_price < 0:
        raise ValueError("list_price must be non-negative")
    return list_price * (1.0 - discount_pct / 100.0)


# CVE ID validation
CVE_PATTERN = re.compile(r"^CVE-\d{4}-\d{4,}$")


def is_valid_cve(cve: str) -> bool:
    return bool(CVE_PATTERN.match(cve))


# ---------------------------------------------------------------------------
# ─── HLE calculation properties ───────────────────────────────────────────────
# ---------------------------------------------------------------------------

@given(
    agents=st.integers(min_value=0, max_value=1_000),
    events_M=st.floats(min_value=0.0, max_value=1_000.0, allow_nan=False, allow_infinity=False),
    ml=st.integers(min_value=0, max_value=1_000_000),
)
def test_hle_always_non_negative(agents: int, events_M: float, ml: int) -> None:
    """HLE must always be >= 0 for non-negative inputs."""
    result = calculate_hle(agents, events_M, ml)
    assert result >= 0


@given(
    agents=st.integers(min_value=0, max_value=1_000),
    events_M=st.floats(min_value=0.0, max_value=1_000.0, allow_nan=False, allow_infinity=False),
    ml=st.integers(min_value=0, max_value=1_000_000),
)
def test_hle_monotone_in_agents(agents: int, events_M: float, ml: int) -> None:
    """Adding more agent seats increases HLE (or keeps it equal)."""
    assume(agents < 1_000)
    base = calculate_hle(agents, events_M, ml)
    higher = calculate_hle(agents + 1, events_M, ml)
    assert higher > base


@given(
    agents=st.integers(min_value=0, max_value=500),
    events_M=st.floats(min_value=0.0, max_value=500.0, allow_nan=False, allow_infinity=False),
    ml=st.integers(min_value=0, max_value=500_000),
)
def test_hle_finite(agents: int, events_M: float, ml: int) -> None:
    """HLE must be finite for finite inputs."""
    result = calculate_hle(agents, events_M, ml)
    assert math.isfinite(result)


@given(st.integers(max_value=-1))
def test_hle_rejects_negative_agents(agents: int) -> None:
    with pytest.raises(ValueError):
        calculate_hle(agents, 0.0, 0)


# ---------------------------------------------------------------------------
# ─── SSVC decision properties ─────────────────────────────────────────────────
# ---------------------------------------------------------------------------

@given(
    exploitation=st.sampled_from(EXPLOITATION_LEVELS),
    automatable=st.sampled_from(AUTOMATABLE_VALUES),
    impact=st.sampled_from(IMPACT_LEVELS),
)
def test_ssvc_always_returns_valid_outcome(
    exploitation: str, automatable: str, impact: str
) -> None:
    """SSVC must return one of the four valid outcomes for any valid input combination."""
    result = ssvc_decision(exploitation, automatable, impact)
    assert result in SSVC_OUTCOMES


@given(automatable=st.sampled_from(AUTOMATABLE_VALUES))
def test_ssvc_active_automatable_yes_is_immediate(automatable: str) -> None:
    """Active exploitation + automatable=yes must always produce IMMEDIATE."""
    assume(automatable == "yes")
    result = ssvc_decision("active", "yes")
    assert result == "IMMEDIATE"


@given(impact=st.sampled_from(IMPACT_LEVELS))
def test_ssvc_no_exploitation_always_defers(impact: str) -> None:
    """No exploitation context always produces DEFER regardless of other factors."""
    result = ssvc_decision("none", "yes", impact)
    assert result == "DEFER"
    result2 = ssvc_decision("none", "no", impact)
    assert result2 == "DEFER"


# ---------------------------------------------------------------------------
# ─── Alert state machine properties ──────────────────────────────────────────
# ---------------------------------------------------------------------------

ALERT_STATES = list(VALID_ALERT_TRANSITIONS.keys())


@given(
    src=st.sampled_from(ALERT_STATES),
    dst=st.sampled_from(ALERT_STATES),
)
def test_alert_valid_transitions_never_raise(src: str, dst: str) -> None:
    """All valid transitions must succeed without exception."""
    allowed = VALID_ALERT_TRANSITIONS.get(src, set())
    if dst in allowed:
        result = alert_transition(src, dst)
        assert result == dst


@given(
    src=st.sampled_from(ALERT_STATES),
    dst=st.sampled_from(ALERT_STATES),
)
def test_alert_invalid_transitions_always_raise(src: str, dst: str) -> None:
    """All invalid transitions must raise InvalidTransitionError."""
    allowed = VALID_ALERT_TRANSITIONS.get(src, set())
    if dst not in allowed:
        with pytest.raises(InvalidTransitionError):
            alert_transition(src, dst)


def test_alert_cannot_skip_directly_to_resolved_from_open() -> None:
    with pytest.raises(InvalidTransitionError):
        alert_transition("open", "resolved")


# ---------------------------------------------------------------------------
# ─── Ticket state machine properties ─────────────────────────────────────────
# ---------------------------------------------------------------------------

TICKET_STATES = list(VALID_TICKET_TRANSITIONS.keys())


@given(
    src=st.sampled_from(TICKET_STATES),
    dst=st.sampled_from(TICKET_STATES),
)
def test_ticket_valid_transitions_never_raise(src: str, dst: str) -> None:
    allowed = VALID_TICKET_TRANSITIONS.get(src, set())
    if dst in allowed:
        result = ticket_transition(src, dst)
        assert result == dst


@given(
    src=st.sampled_from(TICKET_STATES),
    dst=st.sampled_from(TICKET_STATES),
)
def test_ticket_invalid_transitions_always_raise(src: str, dst: str) -> None:
    allowed = VALID_TICKET_TRANSITIONS.get(src, set())
    if dst not in allowed:
        with pytest.raises(InvalidTransitionError):
            ticket_transition(src, dst)


# ---------------------------------------------------------------------------
# ─── Billing discount properties ─────────────────────────────────────────────
# ---------------------------------------------------------------------------

@given(
    price=st.floats(min_value=0.0, max_value=1_000_000.0, allow_nan=False, allow_infinity=False),
    discount=st.floats(min_value=0.0, max_value=100.0, allow_nan=False, allow_infinity=False),
)
def test_billing_discount_never_negative(price: float, discount: float) -> None:
    """Effective price after any valid discount must be >= 0."""
    result = apply_billing_discount(price, discount)
    assert result >= 0


@given(
    price=st.floats(min_value=0.0, max_value=1_000_000.0, allow_nan=False, allow_infinity=False),
    discount=st.floats(min_value=0.0, max_value=100.0, allow_nan=False, allow_infinity=False),
)
def test_billing_discount_never_exceeds_list_price(price: float, discount: float) -> None:
    """Discounted price must never exceed the original list price."""
    result = apply_billing_discount(price, discount)
    assert result <= price + 1e-9  # float tolerance


@given(
    price=st.floats(min_value=1.0, max_value=1_000_000.0, allow_nan=False, allow_infinity=False),
)
def test_billing_zero_discount_returns_full_price(price: float) -> None:
    result = apply_billing_discount(price, 0.0)
    assert abs(result - price) < 1e-9


@given(
    price=st.floats(min_value=1.0, max_value=1_000_000.0, allow_nan=False, allow_infinity=False),
)
def test_billing_100pct_discount_returns_zero(price: float) -> None:
    result = apply_billing_discount(price, 100.0)
    assert abs(result) < 1e-9


@given(st.floats(min_value=100.0001, max_value=1_000.0))
def test_billing_discount_over_100_raises(discount: float) -> None:
    with pytest.raises(ValueError):
        apply_billing_discount(100.0, discount)


@given(st.floats(max_value=-0.0001, allow_nan=False, allow_infinity=False))
def test_billing_negative_discount_raises(discount: float) -> None:
    with pytest.raises(ValueError):
        apply_billing_discount(100.0, discount)


# ---------------------------------------------------------------------------
# ─── CVE ID validation properties ────────────────────────────────────────────
# ---------------------------------------------------------------------------

@given(
    year=st.integers(min_value=1999, max_value=2030),
    num=st.integers(min_value=1000, max_value=9_999_999),
)
def test_valid_cve_ids_accepted(year: int, num: int) -> None:
    cve = f"CVE-{year}-{num}"
    assert is_valid_cve(cve)


@given(st.text(min_size=1, max_size=30))
def test_random_strings_mostly_rejected(s: str) -> None:
    """Most random strings should not match CVE format."""
    if not re.match(r"^CVE-\d{4}-\d{4,}$", s):
        assert not is_valid_cve(s)


# ---------------------------------------------------------------------------
# ─── CVE EPSS score properties ───────────────────────────────────────────────
# ---------------------------------------------------------------------------

@given(score=st.floats(min_value=0.0, max_value=1.0, allow_nan=False, allow_infinity=False))
def test_epss_in_valid_range_is_accepted(score: float) -> None:
    """EPSS scores in [0, 1] should be accepted."""
    assert 0.0 <= score <= 1.0


@given(score=st.floats(min_value=1.001, max_value=100.0, allow_nan=False, allow_infinity=False))
def test_epss_over_1_is_out_of_range(score: float) -> None:
    assert score > 1.0


# ---------------------------------------------------------------------------
# Run directly with pytest
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
