"""
Main entry point for the Modern Indexer service.

- Initializes configuration, indexer, monitor and consumer.
- Handles graceful shutdown on SIGINT / SIGTERM.
- Orchestrates monitor and consumer tasks.
"""

import sys
import logging
import asyncio
import signal

from rich.console import Console
from rich.logging import RichHandler

from src.orchestrator import ModernIndexer
from src.config import IndexerConfig
from src.consumer import run_consumer
from src.monitor import start_monitor_server

# Configure rich logging
console = Console()
FORMAT = "%(message)s"
logging.basicConfig(
    level=logging.INFO,
    format=FORMAT,
    datefmt="[%X]",
    handlers=[RichHandler(console=console)],
)

log = logging.getLogger("indexer")


async def main_async() -> int:
    """
    Run the Modern Indexer service.
    """
    config = IndexerConfig()
    log.info("🚀 Starting Modern Indexer Service.")
    log.debug(f"Configuration: {config.model_dump()}")

    indexer = ModernIndexer(config=config)

    # Event to signal shutdown
    stop_event = asyncio.Event()

    def _signal_handler():
        log.warning("📣 Shutdown signal received. Stopping service…")
        stop_event.set()

    # Register signal handlers
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _signal_handler)

    # Start monitor and consumer
    monitor_task = asyncio.create_task(
        start_monitor_server(indexer, config.monitor_port, stop_event)
    )
    consumer_task = asyncio.create_task(
        run_consumer(indexer, config, stop_event=stop_event)
    )

    # Wait for tasks
    done, pending = await asyncio.wait(
        [consumer_task, monitor_task],
        return_when=asyncio.FIRST_EXCEPTION,
    )

    # handle outcomes
    for task in done:
        if task.exception():
            log.error(f"Task raised exception: {task.exception()}")
            stop_event.set()
            for p in pending:
                p.cancel()
            return 1

    log.info("✅ Indexer shut down cleanly.")
    return 0


def main() -> int:
    """
    Entry point.
    """
    try:
        return asyncio.run(main_async())
    except KeyboardInterrupt:
        log.warning("⚠️ Interrupted by user before startup completed.")
        return 1
    except Exception as e:
        log.exception(f"Unhandled exception in main: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
