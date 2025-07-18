"""
Monitoring server for the ModernIndexer service.

- Exposes Prometheus metrics at `/metrics`
- Provides health & readiness checks
- Tracks useful metrics: throughput, errors, latency, uptime
- Gracefully shuts down when stop_event is set
"""

import asyncio
import logging
import time

from aiohttp import web
from prometheus_client import (
    generate_latest,
    CONTENT_TYPE_LATEST,
    Counter,
    Gauge,
    Histogram,
    Summary,
)
from prometheus_client import REGISTRY

log = logging.getLogger("indexer")

# ───────────────────────────────
# Prometheus Metrics Definitions
# ───────────────────────────────

START_TIME = time.time()

# Counts number of Kafka messages successfully consumed
MESSAGES_CONSUMED = Counter(
    "indexer_messages_consumed_total", "Total number of messages consumed from Kafka"
)

# Counts number of Kafka messages that failed to process
MESSAGES_FAILED = Counter(
    "indexer_messages_failed_total", "Total number of messages that failed processing"
)

# Counts number of batches successfully indexed
BATCHES_INDEXED = Counter(
    "indexer_batches_indexed_total", "Total number of document batches indexed"
)

# Tracks current number of vectors stored in Qdrant
CURRENT_VECTORS = Gauge(
    "indexer_current_vectors", "Current number of vectors in Qdrant collection"
)

# Tracks how long each batch indexing took
BATCH_PROCESSING_SECONDS = Histogram(
    "indexer_batch_processing_seconds",
    "Time taken to index a single batch",
    buckets=[0.1, 0.5, 1, 2, 5, 10, 30],
)

# Tracks how long each message processing took (if done per-message)
MESSAGE_PROCESSING_SECONDS = Histogram(
    "indexer_message_processing_seconds",
    "Time taken to process a single message",
    buckets=[0.01, 0.05, 0.1, 0.5, 1, 2],
)

# Tracks the event loop latency (rough proxy of system load)
EVENT_LOOP_LATENCY_SECONDS = Gauge(
    "indexer_event_loop_latency_seconds", "Estimated asyncio event loop latency"
)

# Service uptime in seconds
UPTIME_SECONDS = Gauge(
    "indexer_uptime_seconds", "Service uptime in seconds since start"
)

# Optional: Version or build info as a label-only metric
BUILD_INFO = Gauge(
    "indexer_build_info", "Build and version information", ["version", "build_date"]
)
BUILD_INFO.labels(version="1.0.0", build_date="2025-07-17").set(1)


async def monitor_loop() -> None:
    """
    Background task to update metrics like uptime & event loop latency.
    """
    loop = asyncio.get_event_loop()
    while True:
        start = loop.time()
        await asyncio.sleep(1)
        latency = loop.time() - start - 1
        EVENT_LOOP_LATENCY_SECONDS.set(max(latency, 0))
        UPTIME_SECONDS.set(time.time() - START_TIME)


def setup_routes(app: web.Application, indexer) -> None:
    """
    Registers HTTP routes on the aiohttp app.
    """

    async def home(request):
        """
        Basic homepage with links.
        """
        html = """
        <html><body>
            <h1>ModernIndexer Monitoring</h1>
            <ul>
                <li><a href="/healthz">Health Check</a></li>
                <li><a href="/readyz">Readiness Check</a></li>
                <li><a href="/metrics">Prometheus Metrics</a></li>
            </ul>
        </body></html>
        """
        return web.Response(text=html, content_type="text/html")

    async def health(request):
        """
        Deep health check: Qdrant + Supabase + Kafka.
        """
        qdrant_status = "unknown"
        supabase_status = "unknown"
        vectors_count = None

        try:
            vectors_count = await asyncio.wait_for(
                asyncio.to_thread(indexer.count), timeout=5
            )
            qdrant_status = "ok"
            CURRENT_VECTORS.set(vectors_count)
        except Exception as e:
            log.exception("Qdrant health check failed")
            qdrant_status = f"error: {e}"

        try:
            await asyncio.wait_for(
                asyncio.to_thread(
                    lambda: indexer.supabase.table("documents")
                    .select("id")
                    .limit(1)
                    .execute()
                ),
                timeout=5,
            )
            supabase_status = "ok"
        except Exception as e:
            log.exception("Supabase health check failed")
            supabase_status = f"error: {e}"

        kafka_status = (
            "configured" if indexer.config.kafka_brokers else "not configured"
        )

        status = (
            "ok" if qdrant_status == "ok" and supabase_status == "ok" else "degraded"
        )

        return web.json_response(
            {
                "status": status,
                "components": {
                    "qdrant": qdrant_status,
                    "supabase": supabase_status,
                    "kafka": kafka_status,
                },
                "current_vectors": vectors_count,
            }
        )

    async def readiness(request):
        """
        Shallow readiness probe: just returns 200.
        """
        return web.json_response({"status": "ready"})

    async def metrics(request):
        """
        Exposes Prometheus metrics.
        """
        data = generate_latest()
        return web.Response(body=data, headers={"Content-Type": CONTENT_TYPE_LATEST})

    app.router.add_get("/", home)
    app.router.add_get("/health", health)
    app.router.add_get("/ready", readiness)
    app.router.add_get("/metrics", metrics)


async def start_monitor_server(indexer, port: int, stop_event: asyncio.Event) -> None:
    """
    Starts the aiohttp monitoring server and runs a background loop
    that updates uptime & event loop metrics.
    """
    app = web.Application()
    setup_routes(app, indexer)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", port)
    await site.start()
    log.info(f"Monitoring server started on port {port}")

    # start the metrics updater
    metrics_task = asyncio.create_task(monitor_loop())

    try:
        await stop_event.wait()
        log.info("Stopping monitoring server…")
    finally:
        metrics_task.cancel()
        await runner.cleanup()
        log.info("Monitoring server stopped.")
