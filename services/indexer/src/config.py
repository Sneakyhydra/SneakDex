"""Configuration for the Indexer service."""

from pydantic_settings import BaseSettings
from pydantic import Field


class IndexerConfig(BaseSettings):
    """Loads Indexer configuration from environment variables or defaults."""

    # Kafka
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

    batch_size: int = Field(default=1000, validation_alias="BATCH_SIZE")
    max_docs: int | None = Field(default=None, validation_alias="MAX_DOCS")

    # Qdrant
    qdrant_url: str = Field(default="", validation_alias="QDRANT_URL")
    qdrant_api_key: str = Field(default="", validation_alias="QDRANT_API_KEY")
    collection_name: str = Field(default="sneakdex", validation_alias="COLLECTION_NAME")
    collection_name_images: str = Field(
        default="sneakdex-images", validation_alias="COLLECTION_NAME_IMAGES"
    )

    # Supabase/Postgres
    supabase_url: str = Field(default="", validation_alias="SUPABASE_URL")
    supabase_api_key: str = Field(default="", validation_alias="SUPABASE_API_KEY")

    monitor_port: int = Field(default=8080, validation_alias="MONITOR_PORT")

    model_config = {
        "env_file": ".env",
        "env_file_encoding": "utf-8",
        "case_sensitive": False,
    }
