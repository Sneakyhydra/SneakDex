"""Kafka consumer for ModernIndexer service."""

import asyncio
import json
import logging
import signal
from typing import Any

from confluent_kafka import Consumer, KafkaError
from src.monitor import MESSAGES_CONSUMED, MESSAGES_FAILED, BATCHES_INDEXED

log = logging.getLogger("indexer")


async def run_consumer(indexer: Any, config: Any, stop_event=None) -> None:
    """
    Starts Kafka consumer loop to feed parsed pages into ModernIndexer.
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

    batch, batch_size = [], getattr(config, "batch_size", 100)
    message_count = 0

    if stop_event is None:
        stop_event = asyncio.Event()

    def signal_handler(sig, _):
        log.info(f"Received signal {sig}. Shutting down consumer.")
        stop_event.set()

    try:
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
    except ValueError:
        log.warning("Signal handling is only allowed in main thread.")

    while not stop_event.is_set():
        msg = consumer.poll(timeout=5.0)
        if msg is None:
            await asyncio.sleep(0.1)
            continue

        if msg.error():
            if msg.error().code() != KafkaError._PARTITION_EOF:
                log.error(
                    f"Consumer error [{msg.topic()}-{msg.partition()}]: {msg.error()}"
                )
            continue

        try:
            value = msg.value().decode("utf-8")
            parsed_page = json.loads(value)
            batch.append(parsed_page)

            MESSAGES_CONSUMED.inc()

            if len(batch) >= batch_size:
                await asyncio.to_thread(indexer.index_batch, batch)
                log.info(f"Indexed batch of {len(batch)} documents.")
                BATCHES_INDEXED.inc()
                batch.clear()

            if config.max_docs and message_count >= config.max_docs:
                log.info(f"Reached max docs: {config.max_docs}. Stopping.")
                break

        except Exception as e:
            MESSAGES_FAILED.inc()
            log.exception(f"Error processing message at offset {msg.offset()}: {e}")

    if batch:
        await asyncio.to_thread(indexer.index_batch, batch)
        BATCHES_INDEXED.inc()
        log.info(f"Indexed final batch of {len(batch)} documents before exit.")

    consumer.close()
    log.info("Kafka consumer closed.")
