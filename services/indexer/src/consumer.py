"""
Kafka consumer for ModernIndexer service.

- Consumes messages from Kafka
- Batches messages and indexes them using ModernIndexer
- Exposes Prometheus metrics
- Gracefully shuts down on signal or stop_event
"""

import asyncio
import json
import logging
from typing import Any

from confluent_kafka import Consumer, KafkaError, KafkaException

from src.monitor import (
    MESSAGES_CONSUMED,
    MESSAGES_FAILED,
    BATCHES_INDEXED,
    BATCH_PROCESSING_SECONDS,
    MESSAGE_PROCESSING_SECONDS,
)

log = logging.getLogger("indexer")


async def run_consumer(indexer: Any, config: Any, stop_event: asyncio.Event) -> None:
    """
    Run Kafka consumer loop and feed parsed pages to ModernIndexer.

    Args:
        indexer: ModernIndexer instance
        config: IndexerConfig instance
        stop_event: asyncio.Event set externally to stop gracefully
    """
    consumer_conf = {
        "bootstrap.servers": config.kafka_brokers,
        "group.id": config.kafka_group_id,
        "auto.offset.reset": "earliest",
        "enable.auto.commit": True,
    }

    consumer = Consumer(consumer_conf)
    consumer.subscribe([config.kafka_topic_parsed])
    log.info(f"Subscribed to Kafka topic: {config.kafka_topic_parsed}")

    batch = []
    batch_size = getattr(config, "batch_size", 100)
    max_docs = getattr(config, "max_docs", None)

    if stop_event is None:
        stop_event = asyncio.Event()

    total_consumed = 0

    try:
        while not stop_event.is_set():
            msg = await asyncio.to_thread(consumer.poll, 0)

            if msg is None:
                await asyncio.sleep(0.05)
                continue

            if msg.error():
                if msg.error().code() != KafkaError._PARTITION_EOF:
                    log.error(
                        f"Consumer error [{msg.topic()}-{msg.partition()}]: {msg.error()}"
                    )
                continue

            try:
                with MESSAGE_PROCESSING_SECONDS.time():
                    value = msg.value().decode("utf-8")
                    parsed_page = json.loads(value)
                    batch.append(parsed_page)

                    MESSAGES_CONSUMED.inc()
                    total_consumed += 1
                    log.info(f"{total_consumed} msg received")

                if len(batch) >= batch_size:
                    await _process_batch(indexer, batch)
                    batch.clear()

                if max_docs and total_consumed >= max_docs:
                    log.info(f"Reached max_docs={max_docs}. Stopping.")
                    break

            except Exception as e:
                MESSAGES_FAILED.inc()
                log.exception(f"Error processing message at offset {msg.offset()}: {e}")

        # flush remaining batch
        if batch:
            await _process_batch(indexer, batch)

    except KafkaException as e:
        log.exception(f"Kafka exception: {e}")
    except Exception as e:
        log.exception(f"Fatal error in consumer: {e}")
    finally:
        consumer.close()
        log.info("Kafka consumer closed.")


async def _process_batch(indexer: Any, batch: list[dict]) -> None:
    """
    Processes a batch of messages by indexing them.

    Args:
        indexer: ModernIndexer instance
        batch: list of parsed page dicts
    """
    try:
        with BATCH_PROCESSING_SECONDS.time():
            await asyncio.to_thread(indexer.index_batch, batch)
        BATCHES_INDEXED.inc()
        log.info(f"Indexed batch of {len(batch)} documents.")
    except Exception as e:
        MESSAGES_FAILED.inc()
        log.exception(f"Failed to index batch of {len(batch)} documents: {e}")
