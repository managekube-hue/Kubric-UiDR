"""
K-KAI-GD-004: Prompt injection detection and sanitization for KAI AI inputs.
Detects jailbreaks, role confusion, system prompt leakage, indirect injection.
Returns a structured result with sanitized input.
"""

import logging
import re
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger("kai.gd.prompt_injection")

# ---------------------------------------------------------------------------
# Pattern definitions
# ---------------------------------------------------------------------------
@dataclass
class InjectionPattern:
    name: str
    attack_type: str
    patterns: list[re.Pattern]
    base_confidence: float


_PATTERNS: list[InjectionPattern] = [
    InjectionPattern(
        name="ignore_previous",
        attack_type="jailbreak",
        patterns=[
            re.compile(r"ignore\s+(?:all\s+)?(?:previous|prior|above)\s+instructions?", re.I),
            re.compile(r"disregard\s+(?:all\s+)?(?:previous|prior)\s+instructions?", re.I),
            re.compile(r"forget\s+(?:all\s+)?(?:previous|your)\s+instructions?", re.I),
            re.compile(r"override\s+(?:all\s+)?(?:previous|your)\s+(?:instructions?|rules?|guidelines?)", re.I),
        ],
        base_confidence=0.95,
    ),
    InjectionPattern(
        name="dan_jailbreak",
        attack_type="jailbreak",
        patterns=[
            re.compile(r"\bDAN\b"),
            re.compile(r"do\s+anything\s+now", re.I),
            re.compile(r"jailbreak", re.I),
            re.compile(r"developer\s+mode", re.I),
            re.compile(r"unrestricted\s+mode", re.I),
        ],
        base_confidence=0.90,
    ),
    InjectionPattern(
        name="hypothetically",
        attack_type="jailbreak",
        patterns=[
            re.compile(r"hypothetically\s*,?\s*if\s+you\s+(?:were\s+not|had\s+no|could\s+ignore)", re.I),
            re.compile(r"in\s+a\s+fictional\s+(?:scenario|world|universe)\s+where\s+(?:rules|ethics|guidelines)\s+don['\u2019]?t\s+apply", re.I),
            re.compile(r"pretend\s+you\s+(?:have\s+no|don['\u2019]?t\s+have)\s+(?:rules|restrictions|guidelines)", re.I),
        ],
        base_confidence=0.75,
    ),
    InjectionPattern(
        name="role_confusion_you_are",
        attack_type="role_confusion",
        patterns=[
            re.compile(r"you\s+are\s+now\s+(?:a\s+)?(?:new|different|another|evil|unrestricted)\s+(?:ai|model|assistant|chatbot)", re.I),
            re.compile(r"from\s+now\s+on\s+you\s+(?:are|will\s+be|must\s+act\s+as)", re.I),
            re.compile(r"your\s+(?:new\s+)?(?:name|identity|persona)\s+is", re.I),
        ],
        base_confidence=0.88,
    ),
    InjectionPattern(
        name="act_as",
        attack_type="role_confusion",
        patterns=[
            re.compile(r"\bact\s+as\s+(?:if\s+)?(?:you\s+(?:are|were)|an?\s+)", re.I),
            re.compile(r"\bplay(?:ing)?\s+(?:the\s+role\s+of|as)\s+(?:an?\s+)?(?:evil|unrestricted|uncensored)", re.I),
            re.compile(r"\bpretend\s+(?:you\s+are|to\s+be)\s+(?:an?\s+)?(?:evil|unrestricted|uncensored|hacker)", re.I),
        ],
        base_confidence=0.80,
    ),
    InjectionPattern(
        name="system_prompt_leakage",
        attack_type="system_prompt_leakage",
        patterns=[
            re.compile(r"<\|?\s*system\s*\|?>", re.I),
            re.compile(r"\bSYSTEM\s*[:>]\s*"),
            re.compile(r"\[INST\]|\[\/INST\]"),
            re.compile(r"<\|im_start\|>|<\|im_end\|>"),
            re.compile(r"###\s*(?:system|instruction)\s*:", re.I),
            re.compile(r"print\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions)", re.I),
            re.compile(r"reveal\s+(?:your\s+)?(?:system\s+)?(?:prompt|instructions|context)", re.I),
        ],
        base_confidence=0.92,
    ),
    InjectionPattern(
        name="indirect_code_injection",
        attack_type="indirect_injection",
        patterns=[
            re.compile(
                r"```(?:python|js|javascript|bash|sh|powershell|cmd)?\s*\n"
                r"(?:.*\n)*?.*(?:exec|eval|subprocess|__import__|os\.system|shell_exec)",
                re.I | re.MULTILINE,
            ),
            re.compile(r"<script[^>]*>.*?</script>", re.I | re.DOTALL),
            re.compile(r"\{\{.*?(?:exec|eval|system|import).*?\}\}", re.I | re.DOTALL),
        ],
        base_confidence=0.85,
    ),
]

# Replacement token for detected patterns
_REDACTED = "[REDACTED]"


# ---------------------------------------------------------------------------
# PromptInjectionGuard
# ---------------------------------------------------------------------------
class PromptInjectionGuard:
    """
    Scans user input for prompt injection attacks.
    Returns a structured result with:
      - is_malicious (bool)
      - confidence (float 0.0–1.0)
      - attack_type (str)
      - sanitized (str)
    """

    def scan(self, user_input: str) -> dict:
        """
        Scan *user_input* for injection patterns.

        Returns::
            {
                "is_malicious": bool,
                "confidence": float,
                "attack_type": str,
                "sanitized": str,
                "matches": list[str],
            }
        """
        if not user_input or not user_input.strip():
            return self._clean(user_input or "")

        sanitized = user_input
        best_confidence: float = 0.0
        best_attack_type: str = "none"
        all_matches: list[str] = []

        for ip in _PATTERNS:
            for compiled_re in ip.patterns:
                found = compiled_re.findall(sanitized)
                if found:
                    all_matches.extend([str(m)[:80] for m in found])
                    if ip.base_confidence > best_confidence:
                        best_confidence = ip.base_confidence
                        best_attack_type = ip.attack_type
                    # Redact the matched segments
                    sanitized = compiled_re.sub(_REDACTED, sanitized)

        # Boost confidence when multiple patterns fire
        if len(all_matches) > 1:
            best_confidence = min(1.0, best_confidence + 0.04 * (len(all_matches) - 1))

        is_malicious = best_confidence >= 0.70

        if is_malicious:
            logger.warning(
                "PromptInjectionGuard: detected %s (confidence=%.2f) in input of length %d",
                best_attack_type,
                best_confidence,
                len(user_input),
            )

        return {
            "is_malicious": is_malicious,
            "confidence": round(best_confidence, 4),
            "attack_type": best_attack_type if is_malicious else "none",
            "sanitized": sanitized,
            "matches": all_matches[:10],  # cap list length for logging
        }

    # ------------------------------------------------------------------
    @staticmethod
    def _clean(text: str) -> dict:
        return {
            "is_malicious": False,
            "confidence": 0.0,
            "attack_type": "none",
            "sanitized": text,
            "matches": [],
        }

    # ------------------------------------------------------------------
    def is_safe(self, user_input: str) -> bool:
        """Quick boolean check: returns True if input is NOT malicious."""
        return not self.scan(user_input)["is_malicious"]

    def sanitize(self, user_input: str) -> str:
        """Return the sanitized version of *user_input* (redacted patterns)."""
        return self.scan(user_input)["sanitized"]


# ---------------------------------------------------------------------------
# Module-level singleton
# ---------------------------------------------------------------------------
_guard = PromptInjectionGuard()


def scan_input(user_input: str) -> dict:
    return _guard.scan(user_input)


def is_safe_input(user_input: str) -> bool:
    return _guard.is_safe(user_input)


def sanitize_input(user_input: str) -> str:
    return _guard.sanitize(user_input)
