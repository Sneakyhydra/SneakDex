"""
Configuration for the Modern Indexer service.
Loads settings from environment variables or `.env` file.
"""

from pydantic_settings import BaseSettings
from pydantic import Field, model_validator


class IndexerConfig(BaseSettings):
    """
    Configuration for ModernIndexer.

    Reads from environment variables or `.env`.
    Raises validation errors if critical fields are missing or invalid.
    """

    # Kafka
    kafka_brokers: str = Field(
        default="kafka:9092",
        description="Kafka bootstrap brokers (comma-separated list).",
        validation_alias="KAFKA_BROKERS",
    )
    kafka_topic_parsed: str = Field(
        default="parsed-pages",
        description="Kafka topic for parsed pages.",
        validation_alias="KAFKA_TOPIC_PARSED",
    )
    kafka_group_id: str = Field(
        default="indexer-group",
        description="Kafka consumer group ID.",
        validation_alias="KAFKA_GROUP_ID",
    )

    # Indexing
    batch_size: int = Field(
        default=100,
        description="Number of documents to process per batch (must be > 0).",
        validation_alias="BATCH_SIZE",
    )
    max_docs: int | None = Field(
        default=None,
        description="Maximum number of documents to process (None = unlimited).",
        validation_alias="MAX_DOCS",
    )
    max_docs_supabase: int | None = Field(
        default=None,
        description="Maximum number of documents to add to supabase (None = unlimited).",
        validation_alias="MAX_DOCS_SUPABASE",
    )

    # Qdrant
    qdrant_url: str = Field(
        default="http://qdrant:6333",
        description="URL of Qdrant instance (required).",
        validation_alias="QDRANT_URL",
    )
    qdrant_api_key: str = Field(
        default="",
        description="API key for Qdrant (optional).",
        validation_alias="QDRANT_API_KEY",
    )
    collection_name: str = Field(
        default="sneakdex",
        description="Qdrant collection name for documents.",
        validation_alias="COLLECTION_NAME",
    )
    collection_name_images: str = Field(
        default="sneakdex-images",
        description="Qdrant collection name for images.",
        validation_alias="COLLECTION_NAME_IMAGES",
    )

    # Supabase
    supabase_url: str = Field(
        default="",
        description="Supabase project URL (required).",
        validation_alias="SUPABASE_URL",
    )
    supabase_api_key: str = Field(
        default="",
        description="Supabase service role API key (required).",
        validation_alias="SUPABASE_API_KEY",
    )

    # Monitoring
    monitor_port: int = Field(
        default=8080,
        description="Port on which the monitoring server runs (1-65535).",
        validation_alias="MONITOR_PORT",
    )

    model_config = {
        "env_file": ".env",
        "env_file_encoding": "utf-8",
        "case_sensitive": False,
    }

    @model_validator(mode="after")
    def validate_config(self) -> "IndexerConfig":
        """
        Ensures all required configurations are valid.
        """
        errors = []

        if not self.qdrant_url.strip():
            errors.append("QDRANT_URL is required and cannot be empty.")
        if not self.supabase_url.strip():
            errors.append("SUPABASE_URL is required and cannot be empty.")
        if not self.supabase_api_key.strip():
            errors.append("SUPABASE_API_KEY is required and cannot be empty.")

        if self.batch_size <= 0:
            errors.append("BATCH_SIZE must be > 0.")
        if not (1 <= self.monitor_port <= 65535):
            errors.append("MONITOR_PORT must be between 1 and 65535.")

        if errors:
            raise ValueError("Invalid configuration:\n" + "\n".join(errors))

        return self
