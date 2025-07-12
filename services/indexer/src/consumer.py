"""Kafka consumer for ModernIndexer service."""

import json
import logging
import signal
from typing import Any

from confluent_kafka import Consumer, KafkaError

log = logging.getLogger("indexer")


def run_consumer(indexer: Any, config: Any, stop_event=None) -> None:
    """
    Starts Kafka consumer loop to feed parsed pages into ModernIndexer.

    Args:
        indexer: ModernIndexer instance with `index_batch(documents: list[dict])`
        config: IndexerConfig instance.
        stop_event: Optional threading.Event or similar to gracefully stop loop.
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

    # Batch config
    batch, batch_size = [], getattr(config, "batch_size", 50)
    message_count = 0

    def signal_handler(sig, _):
        log.info(f"Received signal {sig}. Shutting down consumer.")
        if stop_event:
            stop_event.set()

    try:
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
    except ValueError:
        log.warning("Signal handling is only allowed in main thread.")

    while True:
        if stop_event and stop_event.is_set():
            log.info("Stop event set. Exiting consumer loop.")
            break

        msg = consumer.poll(timeout=5.0)
        if msg is None:
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
            message_count += 1

            if len(batch) >= batch_size:
                indexer.index_batch(batch)
                log.info(f"Indexed batch of {len(batch)} documents.")
                batch.clear()

            if config.max_docs and message_count >= config.max_docs:
                log.info(f"Reached max docs: {config.max_docs}. Stopping.")
                break

        except Exception as e:
            log.exception(f"Error processing message at offset {msg.offset()}: {e}")

    # flush remaining docs
    if batch:
        indexer.index_batch(batch)
        log.info(f"Indexed final batch of {len(batch)} documents before exit.")

    consumer.close()
    log.info("Kafka consumer closed.")
