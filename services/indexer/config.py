"""Configuration for the Indexer service."""

from multiprocessing import cpu_count
from pathlib import Path
from pydantic_settings import BaseSettings
from pydantic import Field, field_validator


class IndexerConfig(BaseSettings):
    """Loads Indexer configuration from environment variables or defaults."""

    kafka_brokers: str = Field(
        default="kafka:9092",
        description="Kafka brokers (comma-separated)",
        validation_alias="KAFKA_BROKERS",
    )
    kafka_topic_parsed: str = Field(
        default="parsed-pages", validation_alias="KAFKA_TOPIC_PARSED"
    )
    kafka_group_id: str = Field(
        default="indexer-group", validation_alias="KAFKA_GROUP_ID"
    )

    index_output_path: str = Field(
        default="/app/data/index", validation_alias="INDEX_OUTPUT_PATH"
    )

    tf_idf_min_df: int = Field(default=2, validation_alias="TF_IDF_MIN_DF")
    max_features: int = Field(default=100_000, validation_alias="MAX_FEATURES")
    batch_size: int = Field(default=1_000, validation_alias="BATCH_SIZE")
    max_docs: int | None = Field(default=None, validation_alias="MAX_DOCS")

    num_worker_threads: int = Field(default=4, validation_alias="NUM_WORKER_THREADS")
    preprocessing_workers: int = Field(
        default=cpu_count(), validation_alias="PREPROCESSING_WORKERS"
    )

    index_save_interval: int = Field(
        default=5_000, validation_alias="INDEX_SAVE_INTERVAL"
    )
    monitor_port: int = Field(default=8080, validation_alias="MONITOR_PORT")

    model_config = {
        "env_file": ".env",
        "env_file_encoding": "utf-8",
        "case_sensitive": False,
    }

    @field_validator("index_output_path")
    @classmethod
    def validate_index_output_path(cls, v: str) -> str:
        path = Path(v)
        path.parent.mkdir(parents=True, exist_ok=True)
        return str(path)
