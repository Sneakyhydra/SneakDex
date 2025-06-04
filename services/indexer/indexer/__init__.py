"""
SneakDex Indexer Service

This service consumes parsed web pages from Kafka, builds a TF-IDF index,
and persists it to disk for use by the query API.
"""

__version__ = "0.1.0"

