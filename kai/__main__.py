"""Entry point: python -m kai

Starts the KAI FastAPI service (kai.api.main:app) with uvicorn.

Environment variables:
  KAI_HOST        Bind address (default: 0.0.0.0)
  KAI_PORT        TCP port     (default: 8101)
  KAI_LOG_LEVEL   Uvicorn log level: debug|info|warning|error|critical (default: info)
  KAI_WORKERS     Number of uvicorn worker processes (default: 1)
                  NOTE: set to 1 when using lifespan background tasks (NATS subscriber,
                  Foresight loop) — multiple workers would each start their own copy.
"""
import os

import uvicorn


def main() -> None:
    uvicorn.run(
        "kai.api.main:app",
        host=os.getenv("KAI_HOST", "0.0.0.0"),
        port=int(os.getenv("KAI_PORT", "8101")),
        log_level=os.getenv("KAI_LOG_LEVEL", "info"),
        access_log=True,
        workers=int(os.getenv("KAI_WORKERS", "1")),
        loop="uvloop",
    )


if __name__ == "__main__":
    main()
