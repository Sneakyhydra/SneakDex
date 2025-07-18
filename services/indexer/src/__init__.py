"""
SneakDex Indexer Service
~~~~~~~~~~~~~~~~~~~~~~~~

This package implements the SneakDex indexing service, which consumes parsed pages
from Kafka, builds embeddings for both text and images, and stores:

- Text documents: in Qdrant (vector) & Supabase (sparse tsvector) for hybrid search
- Images: in Qdrant (vector) for image search

It also exposes Prometheus metrics and a healthcheck server for monitoring,
and supports graceful shutdown and batch processing.

Author: Dhruv Rishishwar
Version: 1.0.0
"""

__version__ = "1.0.0"
__author__ = "Dhruv Rishishwar"
