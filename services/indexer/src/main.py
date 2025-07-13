"""
Main entry point for the Indexer service.
"""

import sys
import logging
import asyncio
from rich.console import Console
from rich.logging import RichHandler

from src.orchestrator import ModernIndexer
from src.config import IndexerConfig
from src.consumer import run_consumer
from src.monitor import start_monitor_server, MESSAGES_CONSUMED

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
    log.info("Starting Modern Indexer Service.")
    log.debug(f"Configuration: {config.model_dump()}")

    indexer = ModernIndexer(config=config)

    # create a stop_event so both consumer & monitor can shut down gracefully
    stop_event = asyncio.Event()

    # start monitor server
    await start_monitor_server(indexer, config.monitor_port)

    # run consumer
    try:
        await run_consumer(indexer, config, stop_event=stop_event)
    except KeyboardInterrupt:
        log.warning("Interrupted by user.")
    except Exception as e:
        log.exception(f"Fatal error occurred: {e}")
        return 1

    log.info("Indexer shut down cleanly.")
    return 0


def main() -> int:
    return asyncio.run(main_async())


if __name__ == "__main__":
    sys.exit(main())
