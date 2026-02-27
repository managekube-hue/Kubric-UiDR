"""
K-KAI-API-003: NATS JetStream consumer runner — __main__ entry point.

Connects to NATS JetStream, ensures the KUBRIC_AGENTS stream exists,
creates a durable pull consumer for each of the 13 KAI persona agents,
then runs all 13 consume loops concurrently via asyncio.gather().

Graceful shutdown is handled on SIGINT / SIGTERM:
  - Sets a shared shutdown asyncio.Event
  - Waits for all consumer tasks to exit their poll loops
  - Drains the NATS connection

Environment variables:
  NATS_URL            NATS server URL         (default: nats://localhost:4222)
  NATS_CONNECT_TIMEOUT  Seconds to wait for initial connect (default: 10)
  CONSUMER_FETCH_TIMEOUT  Seconds per fetch() poll cycle (default: 5)
  CONSUMER_BATCH_SIZE     Messages per fetch() call       (default: 1)
  LOG_LEVEL           Python log level string  (default: INFO)

Usage:
  python K-KAI-API-003_consumer_runner.py
  NATS_URL=nats://nats.internal:4222 python K-KAI-API-003_consumer_runner.py
"""

from __future__ import annotations

import asyncio
import logging
import os
import signal
import sys
from typing import Any

import nats
import nats.errors
from nats.aio.client import Client as NATSConnection
from nats.js import JetStreamContext
from nats.js.api import (
    ConsumerConfig,
    AckPolicy,
    DeliverPolicy,
    RetentionPolicy,
    StorageType,
    StreamConfig,
)

# Ensure the K-KAI-API package directory is on the path so we can import
# the consumer worker module regardless of working directory.
_THIS_DIR = os.path.dirname(os.path.abspath(__file__))
if _THIS_DIR not in sys.path:
    sys.path.insert(0, _THIS_DIR)

from K_KAI_API_002_nats_consumers import (  # noqa: E402
    ALL_PERSONAS,
    STREAM_NAME,
    NatsConsumerWorker,
)

# ---------------------------------------------------------------------------
# Logging configuration
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=os.environ.get("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s %(levelname)-8s %(name)s %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)
logger = logging.getLogger("kai.consumer_runner")

# ---------------------------------------------------------------------------
# Runtime configuration
# ---------------------------------------------------------------------------
NATS_URL: str = os.environ.get("NATS_URL", "nats://localhost:4222")
NATS_CONNECT_TIMEOUT: float = float(os.environ.get("NATS_CONNECT_TIMEOUT", "10"))
CONSUMER_FETCH_TIMEOUT: float = float(os.environ.get("CONSUMER_FETCH_TIMEOUT", "5"))
CONSUMER_BATCH_SIZE: int = int(os.environ.get("CONSUMER_BATCH_SIZE", "1"))

# JetStream stream configuration covering all agent trigger subjects.
_STREAM_CONFIG = StreamConfig(
    name=STREAM_NAME,
    subjects=["kubric.*.agent.*.trigger"],
    retention=RetentionPolicy.WORK_QUEUE,
    storage=StorageType.FILE,
    max_msgs=5_000_000,
    max_msg_size=1 * 1024 * 1024,  # 1 MiB per message
    num_replicas=1,
    # duplicate_window is 2 minutes; prevents re-processing duplicate publishes
    duplicate_window=120_000_000_000,  # nanoseconds
)


# ---------------------------------------------------------------------------
# Stream / consumer bootstrap helpers
# ---------------------------------------------------------------------------

async def _ensure_stream(js: JetStreamContext) -> None:
    """
    Create the KUBRIC_AGENTS stream if it does not already exist.
    If the stream exists with a different config, this is a no-op (add_stream
    is idempotent when called with the same name and compatible subjects).
    """
    try:
        info = await js.find_stream(STREAM_NAME)
        logger.info(
            "Stream '%s' already exists  msgs=%d  bytes=%d",
            STREAM_NAME,
            info.state.messages,
            info.state.bytes,
        )
    except nats.errors.NotFoundError:
        await js.add_stream(_STREAM_CONFIG)
        logger.info("Stream '%s' created  subjects=%s", STREAM_NAME, _STREAM_CONFIG.subjects)
    except Exception as exc:
        # Soft-fail: the stream may have been created by another instance
        # racing us to startup.  Consumer creation will fail noisily later
        # if something is truly wrong.
        logger.warning("Could not verify/create stream '%s': %s", STREAM_NAME, exc)


async def _ensure_consumer(
    js: JetStreamContext,
    persona: str,
    *,
    ack_wait_seconds: int = 60,
    max_deliver: int = 5,
) -> Any:
    """
    Create the durable pull consumer for *persona* if it does not exist.
    Returns the PullSubscription.

    Consumer naming: kubric-agent-<persona>
    Bound to stream : KUBRIC_AGENTS
    Filter subject  : kubric.*.agent.<persona>.trigger
    """
    durable: str = NatsConsumerWorker.durable_name(persona)
    subject: str = NatsConsumerWorker.trigger_subject(persona)

    logger.debug(
        "Creating pull consumer  stream=%s  durable=%s  subject=%s",
        STREAM_NAME,
        durable,
        subject,
    )

    sub = await js.pull_subscribe(
        subject=subject,
        durable=durable,
        stream=STREAM_NAME,
        config=ConsumerConfig(
            durable_name=durable,
            filter_subject=subject,
            deliver_policy=DeliverPolicy.ALL,
            ack_policy=AckPolicy.EXPLICIT,
            ack_wait=ack_wait_seconds * 1_000_000_000,  # nanoseconds
            max_deliver=max_deliver,
        ),
    )
    logger.info(
        "Pull consumer ready  persona=%-10s  durable=%s  subject=%s",
        persona,
        durable,
        subject,
    )
    return sub


# ---------------------------------------------------------------------------
# Main orchestration
# ---------------------------------------------------------------------------

async def run() -> None:
    """
    Full lifecycle:
      1. Connect to NATS with auto-reconnect
      2. Obtain JetStream context
      3. Ensure KUBRIC_AGENTS stream exists
      4. Create durable pull subscriptions for all 13 personas
      5. Instantiate NatsConsumerWorker
      6. Launch 13 asyncio tasks (one consume_loop per persona)
      7. Block until SIGINT / SIGTERM sets the shutdown event
      8. Cancel tasks and drain NATS
    """
    shutdown_event: asyncio.Event = asyncio.Event()

    # NATS connection callbacks -------------------------------------------

    async def _on_error(exc: Exception) -> None:
        logger.error("NATS client error: %s", exc)

    async def _on_reconnect() -> None:
        if nc.connected_url:
            logger.info("NATS reconnected to %s", nc.connected_url.netloc)

    async def _on_disconnect() -> None:
        logger.warning("NATS disconnected")

    async def _on_close() -> None:
        logger.info("NATS connection closed")

    logger.info("Connecting to NATS at %s …", NATS_URL)
    nc: NATSConnection = await nats.connect(
        NATS_URL,
        error_cb=_on_error,
        reconnected_cb=_on_reconnect,
        disconnected_cb=_on_disconnect,
        closed_cb=_on_close,
        connect_timeout=NATS_CONNECT_TIMEOUT,
        reconnect_time_wait=2,
        max_reconnect_attempts=-1,   # infinite
        ping_interval=20,
        max_outstanding_pings=3,
    )
    logger.info("NATS connected  server=%s", nc.connected_url.netloc if nc.connected_url else NATS_URL)

    js: JetStreamContext = nc.jetstream()

    # Stream bootstrap -------------------------------------------------------
    await _ensure_stream(js)

    # Create worker ----------------------------------------------------------
    worker = NatsConsumerWorker(js=js, nc=nc)

    # Create pull subscriptions for all 13 personas --------------------------
    subscriptions: dict[str, Any] = {}
    for persona in ALL_PERSONAS:
        try:
            sub = await _ensure_consumer(js, persona)
            subscriptions[persona] = sub
        except Exception as exc:
            logger.error(
                "Failed to create consumer for persona=%s: %s — skipping",
                persona,
                exc,
            )

    if not subscriptions:
        logger.critical("No consumers could be created; aborting.")
        await nc.drain()
        return

    logger.info(
        "Starting %d consumer loops: %s",
        len(subscriptions),
        ", ".join(subscriptions),
    )

    # Launch consumer tasks --------------------------------------------------
    tasks: list[asyncio.Task] = [
        asyncio.create_task(
            worker.consume_loop(
                persona,
                sub,
                shutdown_event,
                fetch_timeout=CONSUMER_FETCH_TIMEOUT,
                batch_size=CONSUMER_BATCH_SIZE,
            ),
            name=f"consumer-{persona}",
        )
        for persona, sub in subscriptions.items()
    ]

    # Signal handling --------------------------------------------------------
    loop = asyncio.get_running_loop()

    def _handle_signal(sig: signal.Signals) -> None:
        logger.info("Received signal %s — initiating graceful shutdown …", sig.name)
        shutdown_event.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, _handle_signal, sig)
        except NotImplementedError:
            # Windows does not support add_signal_handler for all signals;
            # KeyboardInterrupt will still propagate normally on SIGINT.
            logger.debug("Signal handler for %s not supported on this platform", sig.name)

    # Block until shutdown ---------------------------------------------------
    logger.info("All %d consumer loops running.  Press Ctrl+C or send SIGTERM to stop.", len(tasks))

    try:
        await asyncio.gather(*tasks, return_exceptions=True)
    except (asyncio.CancelledError, KeyboardInterrupt):
        shutdown_event.set()
        # Give running tasks a moment to finish their current message
        await asyncio.gather(*tasks, return_exceptions=True)

    # Teardown ---------------------------------------------------------------
    logger.info("Consumer tasks finished.  Draining NATS connection …")
    try:
        await nc.drain()
    except Exception as exc:
        logger.warning("NATS drain error (safe to ignore on shutdown): %s", exc)

    logger.info("KAI consumer runner shut down cleanly.")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    """
    Synchronous wrapper so this module is usable as:
      - python K-KAI-API-003_consumer_runner.py
      - Temporal worker or systemd ExecStart= command
    """
    try:
        asyncio.run(run())
    except KeyboardInterrupt:
        # asyncio.run() cancels pending tasks on KeyboardInterrupt;
        # the graceful-shutdown path inside run() handles the rest.
        pass


if __name__ == "__main__":
    main()
