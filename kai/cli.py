"""
KAI CLI entry point.

Usage:
  kai serve     — Start the FastAPI server
  kai version   — Print version
"""

import typer
import uvicorn

from kai import __version__
from kai.config import settings

app = typer.Typer(name="kai", help="Kubric KAI orchestration layer CLI")


@app.command()
def serve() -> None:
    """Start the KAI API server."""
    uvicorn.run(
        "kai.api.main:app",
        host=settings.api_host,
        port=settings.api_port,
        log_level=settings.log_level.lower(),
    )


@app.command()
def version() -> None:
    """Print KAI version."""
    typer.echo(f"kubric-kai {__version__}")


if __name__ == "__main__":
    app()
