"""
Colorama cross-platform terminal color support for Kubric CLI output.
Module: K-DEV-TOOLS-005

Install: pip install colorama
"""
from __future__ import annotations

import os
import platform
import sys
from typing import NamedTuple

from colorama import Back, Fore, Style, init

# Initialize colorama on import. autoreset=True automatically resets after each print.
# On Windows, strip=False passes ANSI through (colorama translates them for win32 console).
_is_windows = platform.system() == "Windows"
_has_color = (
    os.environ.get("NO_COLOR") is None
    and (
        sys.stdout.isatty()
        or os.environ.get("FORCE_COLOR") == "1"
        or os.environ.get("TERM") not in (None, "dumb")
    )
)
init(autoreset=True, strip=not _has_color)


# ---------------------------------------------------------------------------
# Basic color helpers
# ---------------------------------------------------------------------------
def success(msg: str) -> str:
    """Return `msg` styled as a success (green)."""
    return f"{Fore.GREEN}{Style.BRIGHT}{msg}{Style.RESET_ALL}"


def error(msg: str) -> str:
    """Return `msg` styled as an error (bright red)."""
    return f"{Fore.RED}{Style.BRIGHT}{msg}{Style.RESET_ALL}"


def warning(msg: str) -> str:
    """Return `msg` styled as a warning (yellow)."""
    return f"{Fore.YELLOW}{Style.BRIGHT}{msg}{Style.RESET_ALL}"


def info(msg: str) -> str:
    """Return `msg` styled as informational (cyan)."""
    return f"{Fore.CYAN}{msg}{Style.RESET_ALL}"


def highlight(msg: str) -> str:
    """Return `msg` with bright white highlight."""
    return f"{Style.BRIGHT}{msg}{Style.RESET_ALL}"


def dim(msg: str) -> str:
    """Return `msg` dimmed."""
    return f"{Style.DIM}{msg}{Style.RESET_ALL}"


# ---------------------------------------------------------------------------
# Severity-coded color
# ---------------------------------------------------------------------------
SEVERITY_COLORS: dict[str, str] = {
    "critical": f"{Back.RED}{Fore.WHITE}{Style.BRIGHT}",
    "high":     f"{Fore.RED}{Style.BRIGHT}",
    "medium":   f"{Fore.YELLOW}{Style.BRIGHT}",
    "low":      f"{Fore.BLUE}",
    "info":     f"{Style.DIM}",
}


def color_severity(sev: str) -> str:
    """Return severity string with appropriate color codes."""
    style = SEVERITY_COLORS.get(sev.lower(), "")
    return f"{style}{sev.upper()}{Style.RESET_ALL}"


# ---------------------------------------------------------------------------
# Kubric ASCII banner
# ---------------------------------------------------------------------------
_BANNER = r"""
 _  __     _          _
| |/ /   _| |__  _ __(_) ___
| ' / | | | '_ \| '__| |/ __|
| . \ |_| | |_) | |  | | (__
|_|\_\__,_|_.__/|_|  |_|\___|
"""


def print_banner() -> None:
    """Print the Kubric ASCII art banner with gradient colors."""
    lines = _BANNER.strip("\n").split("\n")
    colors = [Fore.BLUE, Fore.CYAN, Fore.WHITE, Fore.CYAN, Fore.BLUE]
    for i, line in enumerate(lines):
        color = colors[i % len(colors)]
        print(f"{color}{Style.BRIGHT}{line}{Style.RESET_ALL}")
    print(f"{Fore.CYAN}  Kubric Security Platform — CLI{Style.RESET_ALL}\n")


# ---------------------------------------------------------------------------
# Status line (for service health)
# ---------------------------------------------------------------------------
def print_status_line(service: str, status: str, latency_ms: float | None = None) -> None:
    """
    Print a single colored service status line.
    status: 'ok' | 'down' | 'degraded' | string
    """
    service_col = f"{Fore.WHITE}{Style.BRIGHT}{service:<20}{Style.RESET_ALL}"
    if status.lower() in ("ok", "healthy", "up"):
        status_col = f"{Fore.GREEN}{Style.BRIGHT}{'OK':<12}{Style.RESET_ALL}"
    elif status.lower() in ("down", "error", "unreachable"):
        status_col = f"{Fore.RED}{Style.BRIGHT}{'DOWN':<12}{Style.RESET_ALL}"
    elif status.lower() in ("degraded", "slow"):
        status_col = f"{Fore.YELLOW}{Style.BRIGHT}{status.upper():<12}{Style.RESET_ALL}"
    else:
        status_col = f"{Fore.CYAN}{status:<12}{Style.RESET_ALL}"

    latency_col = ""
    if latency_ms is not None:
        latency_str = f"{latency_ms:.0f}ms"
        if latency_ms < 100:
            latency_col = f"{Fore.GREEN}{latency_str}{Style.RESET_ALL}"
        elif latency_ms < 500:
            latency_col = f"{Fore.YELLOW}{latency_str}{Style.RESET_ALL}"
        else:
            latency_col = f"{Fore.RED}{latency_str}{Style.RESET_ALL}"

    print(f"  {service_col} {status_col} {latency_col}")


# ---------------------------------------------------------------------------
# Table border
# ---------------------------------------------------------------------------
def print_section(title: str) -> None:
    """Print a colored section separator."""
    width = 60
    bar = "─" * width
    print(f"\n{Fore.BLUE}{Style.BRIGHT}{bar}{Style.RESET_ALL}")
    print(f"{Fore.BLUE}{Style.BRIGHT}  {title}{Style.RESET_ALL}")
    print(f"{Fore.BLUE}{Style.BRIGHT}{bar}{Style.RESET_ALL}\n")


# ---------------------------------------------------------------------------
# Demo
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    print_banner()
    print(success("Database migration applied"))
    print(error("CRITICAL: agent connection lost on prod-01"))
    print(warning("WARN: EPSS score 0.87 on CVE-2024-12345"))
    print(info("INFO: 847 agents reporting"))
    print(highlight("Highlight text"))

    print_section("Service Health")
    print_status_line("api-gateway",   "ok",       42.3)
    print_status_line("kai-core",      "ok",       88.1)
    print_status_line("noc-api",       "degraded", 612.0)
    print_status_line("clickhouse",    "down")

    print_section("Alert Severities")
    for sev in ("critical", "high", "medium", "low", "info"):
        print(f"  {color_severity(sev)}")
