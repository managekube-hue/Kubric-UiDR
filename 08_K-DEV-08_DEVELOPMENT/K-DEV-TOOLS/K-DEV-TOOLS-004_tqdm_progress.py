"""
tqdm progress bar utilities for Kubric batch operations.
Module: K-DEV-TOOLS-004

Install: pip install tqdm httpx
"""
from __future__ import annotations

import asyncio
import os
from pathlib import Path
from typing import Any, Callable, Coroutine, Iterable, TypeVar

import httpx
from tqdm import tqdm, trange
from tqdm.asyncio import tqdm as atqdm

T = TypeVar("T")
R = TypeVar("R")


# ---------------------------------------------------------------------------
# Sync batch processing
# ---------------------------------------------------------------------------
def batch_process_with_progress(
    items: list[T],
    fn: Callable[[T], R],
    desc: str = "Processing",
    unit: str = "item",
) -> list[R]:
    """
    Apply `fn` to each item in `items` with a tqdm progress bar.
    Returns the list of results in order.
    """
    results: list[R] = []
    with tqdm(items, desc=desc, unit=unit, dynamic_ncols=True) as bar:
        for item in bar:
            result = fn(item)
            results.append(result)
    return results


# ---------------------------------------------------------------------------
# Async batch processing
# ---------------------------------------------------------------------------
async def async_batch_with_progress(
    items: list[T],
    coro_fn: Callable[[T], Coroutine[Any, Any, R]],
    batch_size: int = 50,
    desc: str = "Processing",
    unit: str = "item",
) -> list[R]:
    """
    Run `coro_fn(item)` for every item in `items`, concurrently up to
    `batch_size` at a time, with a tqdm progress bar.
    """
    results: list[R] = []
    with tqdm(total=len(items), desc=desc, unit=unit, dynamic_ncols=True) as bar:
        for start in range(0, len(items), batch_size):
            chunk = items[start : start + batch_size]
            chunk_results = await asyncio.gather(*[coro_fn(i) for i in chunk])
            results.extend(chunk_results)
            bar.update(len(chunk))
    return results


# ---------------------------------------------------------------------------
# Download with progress
# ---------------------------------------------------------------------------
def download_with_progress(url: str, dest_path: str | Path) -> Path:
    """
    Stream-download `url` to `dest_path` with a tqdm progress bar.
    Returns the destination Path.
    """
    dest = Path(dest_path)
    dest.parent.mkdir(parents=True, exist_ok=True)

    with httpx.stream("GET", url, follow_redirects=True, timeout=300.0) as resp:
        resp.raise_for_status()
        total = int(resp.headers.get("content-length", 0)) or None
        with (
            open(dest, "wb") as f,
            tqdm(
                total=total,
                unit="B",
                unit_scale=True,
                unit_divisor=1024,
                desc=dest.name,
                dynamic_ncols=True,
            ) as bar,
        ):
            for chunk in resp.iter_bytes(chunk_size=65536):
                f.write(chunk)
                bar.update(len(chunk))
    return dest


# ---------------------------------------------------------------------------
# Database migration progress
# ---------------------------------------------------------------------------
def database_migration_progress(
    tasks: list[str],
    fn: Callable[[str], None] | None = None,
) -> None:
    """
    Show nested progress for a multi-step database migration.
    `fn` is called with each task name; if None, a no-op is used.
    """
    if fn is None:
        def fn(t: str) -> None:
            import time; time.sleep(0.05)

    outer = tqdm(tasks, desc="DB Migrations", unit="migration", position=0, leave=True, dynamic_ncols=True)
    for task in outer:
        outer.set_description(f"Applying: {task}")
        fn(task)
        outer.set_postfix_str("done")
    outer.close()
    tqdm.write("All migrations applied.")


# ---------------------------------------------------------------------------
# Bulk insert progress helper
# ---------------------------------------------------------------------------
def bulk_insert_progress(
    rows: list[dict[str, Any]],
    insert_fn: Callable[[list[dict[str, Any]]], int],
    batch_size: int = 1000,
    desc: str = "Inserting rows",
) -> int:
    """
    Insert `rows` in chunks of `batch_size` with a tqdm bar.
    `insert_fn` receives a chunk and returns the number inserted.
    Returns total rows inserted.
    """
    total_inserted = 0
    with tqdm(total=len(rows), desc=desc, unit="row", dynamic_ncols=True) as bar:
        for start in range(0, len(rows), batch_size):
            chunk = rows[start : start + batch_size]
            n = insert_fn(chunk)
            total_inserted += n
            bar.update(len(chunk))
            bar.set_postfix(inserted=total_inserted)
    return total_inserted


# ---------------------------------------------------------------------------
# CLI demo
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import time

    print("=== Sync batch demo ===")
    items = list(range(30))
    results = batch_process_with_progress(
        items,
        fn=lambda x: x * 2,
        desc="Doubling ints",
    )
    print(f"First 5 results: {results[:5]}")

    print("\n=== DB migration demo ===")
    migrations = [
        "001_create_tenants",
        "002_create_users",
        "003_create_assets",
        "004_create_alerts",
        "005_rls_policies",
    ]
    database_migration_progress(migrations, fn=lambda t: time.sleep(0.1))

    print("\n=== Bulk insert demo ===")
    fake_rows = [{"id": i, "val": i * 10} for i in range(5000)]
    total = bulk_insert_progress(
        fake_rows,
        insert_fn=lambda chunk: len(chunk),
        batch_size=500,
        desc="Inserting alerts",
    )
    print(f"Total inserted: {total}")
