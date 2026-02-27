"""
K-KAI-API-008: asgiref SyncToAsync/AsyncToSync bridge utilities.
Provides decorators and a SyncWorkerPool for bridging Django ORM or
other sync code into async contexts used by KAI.
"""

import asyncio
import logging
from concurrent.futures import Future, ThreadPoolExecutor, wait, FIRST_EXCEPTION
from functools import wraps
from typing import Any, Callable, Coroutine

from asgiref.sync import SyncToAsync, AsyncToSync, sync_to_async, async_to_sync

logger = logging.getLogger("kai.asgiref")


# ---------------------------------------------------------------------------
# Decorator: sync_to_async_handler
# ---------------------------------------------------------------------------
def sync_to_async_handler(sync_fn: Callable) -> Callable:
    """
    Decorator that wraps a synchronous function so it can be awaited in
    an async context without blocking the event loop.

    Thread-safety: uses a new thread per call (thread_sensitive=False) which
    is correct for non-Django sync code.

    Usage::

        @sync_to_async_handler
        def load_csv(path: str) -> list[dict]:
            ...

        result = await load_csv("/data/events.csv")
    """
    wrapper = sync_to_async(sync_fn, thread_sensitive=False)

    @wraps(sync_fn)
    async def _async_wrapper(*args: Any, **kwargs: Any) -> Any:
        return await wrapper(*args, **kwargs)

    return _async_wrapper


# ---------------------------------------------------------------------------
# Decorator: async_to_sync_handler
# ---------------------------------------------------------------------------
def async_to_sync_handler(async_fn: Callable[..., Coroutine]) -> Callable:
    """
    Decorator that wraps an async function so it can be called from
    synchronous code (e.g., Django management commands, Celery tasks).

    Usage::

        @async_to_sync_handler
        async def fetch_score(tenant_id: str) -> int:
            ...

        score = fetch_score("acme")
    """
    wrapper = async_to_sync(async_fn)

    @wraps(async_fn)
    def _sync_wrapper(*args: Any, **kwargs: Any) -> Any:
        return wrapper(*args, **kwargs)

    return _sync_wrapper


# ---------------------------------------------------------------------------
# run_in_django_loop
# ---------------------------------------------------------------------------
def run_in_django_loop(coroutine: Coroutine) -> Any:
    """
    Run a coroutine from a purely synchronous context — e.g., a Django ORM
    signal handler or a Celery task.

    Creates a new event loop if none is running (Celery / management commands).
    If an event loop is already running (ASGI context), wraps via async_to_sync.
    """
    try:
        loop = asyncio.get_running_loop()
    except RuntimeError:
        loop = None

    if loop is None or not loop.is_running():
        return asyncio.run(coroutine)
    else:
        # We're inside an event loop; bridge via asgiref
        bridge = async_to_sync(lambda: coroutine)
        return bridge()


# ---------------------------------------------------------------------------
# SyncWorkerPool
# ---------------------------------------------------------------------------
class SyncWorkerPool:
    """
    Thread pool that submits synchronous callables from async or sync code.
    Uses asgiref SyncToAsync semantics: each callable runs in a dedicated
    thread and the result is returned as a concurrent.futures.Future so
    callers can await or block as needed.

    Usage (async)::

        pool = SyncWorkerPool()
        future = pool.submit(blocking_io_func, arg1, arg2)
        result = await asyncio.wrap_future(future)

    Usage (sync)::

        future = pool.submit(blocking_io_func, arg1, arg2)
        result = future.result(timeout=30)
    """

    def __init__(self, max_workers: int = 8) -> None:
        self._executor = ThreadPoolExecutor(
            max_workers=max_workers,
            thread_name_prefix="kai-sync-worker",
        )
        logger.info("SyncWorkerPool created with max_workers=%d", max_workers)

    def submit(self, fn: Callable, *args: Any, **kwargs: Any) -> Future:
        """
        Submit *fn* to the thread pool.
        Returns a concurrent.futures.Future immediately.
        """
        future = self._executor.submit(fn, *args, **kwargs)
        logger.debug("SyncWorkerPool: submitted %s", getattr(fn, "__name__", repr(fn)))
        return future

    async def submit_async(self, fn: Callable, *args: Any, **kwargs: Any) -> Any:
        """
        Submit *fn* to the thread pool and await the result.
        Wraps the concurrent.futures.Future in asyncio.
        """
        loop = asyncio.get_event_loop()
        future = self._executor.submit(fn, *args, **kwargs)
        return await loop.run_in_executor(None, future.result)

    def shutdown(self, wait: bool = True) -> None:
        """Shut down the underlying ThreadPoolExecutor."""
        self._executor.shutdown(wait=wait)
        logger.info("SyncWorkerPool shut down")


# ---------------------------------------------------------------------------
# Module-level pool singleton
# ---------------------------------------------------------------------------
_default_pool = SyncWorkerPool(max_workers=8)


def get_worker_pool() -> SyncWorkerPool:
    return _default_pool
