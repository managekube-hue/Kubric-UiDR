"""
K-KAI-API-007: anyio backend configuration and task group utilities.
Provides run_sync_in_worker, create_task_group, run_all_concurrent,
and BackgroundTaskManager for KAI mixed async workloads.
"""

import asyncio
import logging
from collections.abc import Coroutine
from typing import Any, Callable

import anyio
import anyio.to_thread
import anyio.from_thread
from anyio import create_task_group as _anyio_tg
from anyio.abc import TaskGroup

logger = logging.getLogger("kai.anyio")


# ---------------------------------------------------------------------------
# run_sync_in_worker
# ---------------------------------------------------------------------------
async def run_sync_in_worker(func: Callable, *args: Any) -> Any:
    """
    Run a synchronous callable in a worker thread via anyio's thread pool.
    Prevents blocking the event loop for CPU-bound or I/O-bound sync code.

    Usage::

        result = await run_sync_in_worker(some_blocking_function, arg1, arg2)
    """
    return await anyio.to_thread.run_sync(lambda: func(*args))


# ---------------------------------------------------------------------------
# create_task_group
# ---------------------------------------------------------------------------
def create_task_group() -> TaskGroup:
    """
    Return an anyio TaskGroup context manager.
    Wraps anyio.create_task_group so callers don't import anyio directly.

    Usage::

        async with create_task_group() as tg:
            tg.start_soon(coro_a)
            tg.start_soon(coro_b)
    """
    return _anyio_tg()


# ---------------------------------------------------------------------------
# run_all_concurrent
# ---------------------------------------------------------------------------
async def run_all_concurrent(coroutines: list[Coroutine]) -> list[Any]:
    """
    Run all coroutines concurrently inside a single anyio task group.
    Returns results in the same order as the input list.

    If any coroutine raises an exception the entire group is cancelled
    (anyio nursery semantics).
    """
    results: list[Any] = [None] * len(coroutines)
    exceptions: list[BaseException | None] = [None] * len(coroutines)

    async def _run(index: int, coro: Coroutine) -> None:
        try:
            results[index] = await coro
        except Exception as exc:
            logger.warning("run_all_concurrent task %d raised: %s", index, exc)
            exceptions[index] = exc
            raise  # propagate so task group can cancel siblings

    async with _anyio_tg() as tg:
        for idx, coro in enumerate(coroutines):
            tg.start_soon(_run, idx, coro)

    return results


# ---------------------------------------------------------------------------
# BackgroundTaskManager
# ---------------------------------------------------------------------------
class BackgroundTaskManager:
    """
    Tracks long-running background tasks and supports graceful shutdown.

    Usage::

        mgr = BackgroundTaskManager()
        await mgr.start()

        async def worker():
            while True:
                await anyio.sleep(1)

        mgr.spawn(worker())

        await mgr.cancel_all()
    """

    def __init__(self) -> None:
        self._tasks: list[asyncio.Task] = []
        self._running = False

    def spawn(self, coro: Coroutine, name: str | None = None) -> asyncio.Task:
        """Schedule *coro* as a background asyncio task and track it."""
        task = asyncio.get_event_loop().create_task(coro, name=name)
        self._tasks.append(task)
        task.add_done_callback(self._tasks.remove)
        logger.debug("BackgroundTaskManager: spawned task %s", name or repr(task))
        return task

    async def cancel_all(self, timeout: float = 5.0) -> None:
        """Cancel all tracked tasks and wait for them to finish."""
        if not self._tasks:
            return
        logger.info("BackgroundTaskManager: cancelling %d tasks", len(self._tasks))
        for task in list(self._tasks):
            task.cancel()
        try:
            await asyncio.wait_for(
                asyncio.gather(*self._tasks, return_exceptions=True),
                timeout=timeout,
            )
        except asyncio.TimeoutError:
            logger.warning("BackgroundTaskManager: cancel_all timed out after %.1fs", timeout)
        logger.info("BackgroundTaskManager: all tasks cancelled")

    @property
    def active_count(self) -> int:
        return len(self._tasks)


# ---------------------------------------------------------------------------
# Module-level manager singleton
# ---------------------------------------------------------------------------
background_tasks = BackgroundTaskManager()
