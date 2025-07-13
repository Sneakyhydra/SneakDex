# monitor.py

from aiohttp import web
from prometheus_client import CollectorRegistry, generate_latest, CONTENT_TYPE_LATEST
from prometheus_client import Counter, Gauge
import logging

log = logging.getLogger("indexer")

# Prometheus metrics
MESSAGES_CONSUMED = Counter(
    "indexer_messages_consumed_total", "Total number of messages consumed"
)

BATCHES_INDEXED = Counter(
    "indexer_batches_indexed_total", "Total number of batches successfully indexed"
)

MESSAGES_FAILED = Counter(
    "indexer_messages_failed_total", "Total number of messages that failed to process"
)

CURRENT_VECTORS = Gauge(
    "indexer_current_vectors", "Current number of vectors in Qdrant collection"
)


def setup_routes(app, indexer):
    async def health(request):
        # check Qdrant
        try:
            vectors_count = indexer.count()
            qdrant_status = "ok"
            CURRENT_VECTORS.set(vectors_count)
        except Exception as e:
            log.exception("Qdrant health check failed")
            qdrant_status = f"error: {e}"
            vectors_count = None

        # check Supabase
        try:
            # just try a dummy query to test connection
            indexer.supabase.table("documents").select("id").limit(1).execute()
            supabase_status = "ok"
        except Exception as e:
            log.exception("Supabase health check failed")
            supabase_status = f"error: {e}"

        # Kafka health is harder to check here since you use Confluent’s client;
        # so we just report consumer configured or not
        kafka_status = (
            "configured" if indexer.config.kafka_brokers else "not configured"
        )

        return web.json_response(
            {
                "status": (
                    "ok"
                    if qdrant_status == "ok" and supabase_status == "ok"
                    else "degraded"
                ),
                "components": {
                    "qdrant": qdrant_status,
                    "supabase": supabase_status,
                    "kafka": kafka_status,
                },
                "current_vectors": vectors_count,
            }
        )

    async def metrics(request):
        data = generate_latest()
        return web.Response(body=data, content_type=CONTENT_TYPE_LATEST)

    app.router.add_get("/health", health)
    app.router.add_get("/metrics", metrics)


async def start_monitor_server(indexer, port: int):
    app = web.Application()
    setup_routes(app, indexer)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", port)
    await site.start()
    log.info(f"Monitoring server started on port {port}")
