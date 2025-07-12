"""
Main entry point for the Indexer service.

- Loads configuration from environment (.env)
- Initializes ModernIndexer (Qdrant + SentenceTransformer)
- Starts Kafka consumer to fetch and index documents
- Gracefully shuts down on interrupt or error
"""

import sys
import logging
from rich.console import Console
from rich.logging import RichHandler

from orchestrator import ModernIndexer
from config import IndexerConfig
from consumer import run_consumer

console = Console()

FORMAT = "%(message)s"
logging.basicConfig(
    level=logging.INFO,
    format=FORMAT,
    datefmt="[%X]",
    handlers=[RichHandler(console=console)],
)

log = logging.getLogger("indexer")


def main() -> int:
    """
    Run the Modern Indexer service.

    Returns:
        Exit code (0 for success, 1 for failure)
    """
    config = IndexerConfig()
    log.info("Starting Modern Indexer Service.")
    log.debug(f"Configuration: {config.model_dump()}")

    collection_name = getattr(config, "collection_name", "sneakdex")
    indexer = ModernIndexer(config=config, collection_name=collection_name)

    try:
        run_consumer(indexer, config)
    except KeyboardInterrupt:
        log.warning("Interrupted by user.")
    except Exception as e:
        log.exception(f"Fatal error occurred: {e}")
        return 1

    log.info("Indexer shut down cleanly.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
