#!/usr/bin/env python3
"""
Main module for the SneakDex Indexer service.

This service:
1. Consumes parsed web pages from Kafka
2. Processes text using NLP techniques
3. Builds a TF-IDF index
4. Persists the index to disk
"""

import json
import logging
import os
import sys
import time
from collections import defaultdict
from pathlib import Path
from typing import Dict, List, Optional, Set, Tuple, Union

import joblib
import nltk
import numpy as np
import pandas as pd
from confluent_kafka import Consumer, KafkaError, KafkaException
from nltk.corpus import stopwords
from nltk.stem import PorterStemmer
from nltk.tokenize import word_tokenize
from pydantic import Field
from pydantic_settings import BaseSettings
from rich.console import Console
from rich.logging import RichHandler
from sklearn.feature_extraction.text import TfidfVectorizer
from tqdm import tqdm

# Set up rich console for better output
console = Console()

# Configure logging
FORMAT = "%(message)s"
logging.basicConfig(
    level=logging.INFO,
    format=FORMAT,
    datefmt="[%X]",
    handlers=[RichHandler(console=console)]
)
log = logging.getLogger("indexer")


class IndexerConfig(BaseSettings):
    """Configuration for the indexer service."""
    
    # Kafka settings
    kafka_brokers: str = Field(default="kafka:9092", env="KAFKA_BROKERS")
    kafka_topic_parsed: str = Field(default="parsed-pages", env="KAFKA_TOPIC_PARSED")
    kafka_group_id: str = Field(default="indexer-group", env="KAFKA_GROUP_ID")
    
    # Indexing settings
    index_output_path: str = Field(default="/app/data/index", env="INDEX_OUTPUT_PATH")
    tf_idf_min_df: int = Field(default=2, env="TF_IDF_MIN_DF")
    max_features: Optional[int] = Field(default=100000, env="MAX_FEATURES")
    
    # Processing settings
    batch_size: int = Field(default=1000, env="BATCH_SIZE")
    max_docs: Optional[int] = Field(default=None, env="MAX_DOCS")
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


class DocumentStore:
    """Stores documents and their metadata for indexing."""
    
    def __init__(self):
        self.documents = []
        self.urls = []
        self.titles = []
        self.document_count = 0
    
    def add_document(self, doc_text: str, url: str, title: str) -> None:
        """Add a document to the store."""
        self.documents.append(doc_text)
        self.urls.append(url)
        self.titles.append(title)
        self.document_count += 1
    
    def get_dataframe(self) -> pd.DataFrame:
        """Convert stored documents to a DataFrame."""
        return pd.DataFrame({
            "url": self.urls,
            "title": self.titles,
            "text": self.documents
        })


class Preprocessor:
    """Text preprocessing for indexing."""
    
    def __init__(self):
        # Download NLTK resources if needed
        try:
            nltk.data.find('tokenizers/punkt')
            nltk.data.find('corpora/stopwords')
        except LookupError:
            nltk.download('punkt')
            nltk.download('stopwords')
        
        self.stemmer = PorterStemmer()
        self.stop_words = set(stopwords.words('english'))
    
    def preprocess(self, text: str) -> str:
        """Preprocess text by tokenizing, removing stopwords, and stemming."""
        # Tokenize
        tokens = word_tokenize(text.lower())
        
        # Remove stopwords and stem
        processed_tokens = [
            self.stemmer.stem(token)
            for token in tokens
            if token.isalnum() and token not in self.stop_words
        ]
        
        return " ".join(processed_tokens)


class Indexer:
    """Builds and persists a TF-IDF index."""
    
    def __init__(self, config: IndexerConfig):
        self.config = config
        self.preprocessor = Preprocessor()
        self.document_store = DocumentStore()
        self.vectorizer = TfidfVectorizer(
            min_df=config.tf_idf_min_df,
            max_features=config.max_features,
            stop_words='english',
            use_idf=True,
            norm='l2'
        )
    
    def process_message(self, message: dict) -> None:
        """Process a parsed page message."""
        try:
            url = message.get('url', '')
            title = message.get('title', 'No Title')
            body_text = message.get('body_text', '')
            
            if not body_text or not url:
                log.warning(f"Skipping document with empty body or URL: {url}")
                return
            
            # Preprocess text
            processed_text = self.preprocessor.preprocess(body_text)
            
            # Add to document store
            self.document_store.add_document(processed_text, url, title)
            
            # Log progress periodically
            if self.document_store.document_count % 100 == 0:
                log.info(f"Processed {self.document_store.document_count} documents")
                
        except Exception as e:
            log.error(f"Error processing message: {e}")
    
    def build_index(self) -> Tuple[np.ndarray, List[str], pd.DataFrame]:
        """Build TF-IDF index from collected documents."""
        log.info(f"Building TF-IDF index from {self.document_store.document_count} documents")
        
        # Get document dataframe
        df = self.document_store.get_dataframe()
        
        # Create TF-IDF matrix
        tfidf_matrix = self.vectorizer.fit_transform(df['text'])
        
        # Get feature names
        feature_names = self.vectorizer.get_feature_names_out()
        
        log.info(f"Created TF-IDF matrix with shape {tfidf_matrix.shape}")
        log.info(f"Vocabulary size: {len(feature_names)}")
        
        return tfidf_matrix, feature_names, df
    
    def save_index(self, tfidf_matrix: np.ndarray, feature_names: List[str], df: pd.DataFrame) -> None:
        """Save the built index to disk."""
        output_path = Path(self.config.index_output_path)
        output_path.mkdir(parents=True, exist_ok=True)
        
        log.info(f"Saving index to {output_path}")
        
        # Save TF-IDF matrix
        joblib.dump(tfidf_matrix, output_path / 'tfidf_matrix.joblib')
        
        # Save feature names (vocabulary)
        joblib.dump(feature_names, output_path / 'feature_names.joblib')
        
        # Save document metadata (URLs and titles)
        df[['url', 'title']].to_csv(output_path / 'document_metadata.csv', index=False)
        
        # Save vectorizer for consistent transformation
        joblib.dump(self.vectorizer, output_path / 'vectorizer.joblib')
        
        log.info("Index saved successfully")


def main():
    """Main entry point for the indexer service."""
    try:
        # Load configuration
        config = IndexerConfig()
        log.info(f"Starting indexer service with config: {config.dict()}")
        
        # Create indexer
        indexer = Indexer(config)
        
        # Configure Kafka consumer
        consumer_conf = {
            'bootstrap.servers': config.kafka_brokers,
            'group.id': config.kafka_group_id,
            'auto.offset.reset': 'earliest',
            'enable.auto.commit': True,
        }
        
        consumer = Consumer(consumer_conf)
        consumer.subscribe([config.kafka_topic_parsed])
        
        log.info(f"Subscribed to Kafka topic: {config.kafka_topic_parsed}")
        log.info("Waiting for messages...")
        
        # Process messages
        message_count = 0
        max_retries = 5
        retry_count = 0
        
        try:
            while True:
                if config.max_docs and message_count >= config.max_docs:
                    log.info(f"Reached maximum document count ({config.max_docs}), stopping consumption")
                    break
                
                msg = consumer.poll(timeout=1.0)
                
                if msg is None:
                    retry_count += 1
                    if retry_count > max_retries:
                        log.info("No more messages received, proceeding to indexing")
                        break
                    continue
                
                retry_count = 0
                
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        log.info("Reached end of partition")
                        continue
                    else:
                        log.error(f"Consumer error: {msg.error()}")
                        continue
                
                try:
                    # Parse message
                    value = msg.value().decode('utf-8')
                    parsed_page = json.loads(value)
                    
                    # Process message
                    indexer.process_message(parsed_page)
                    message_count += 1
                    
                except Exception as e:
                    log.error(f"Error processing message: {e}")
            
            # Build and save index if we have documents
            if indexer.document_store.document_count > 0:
                log.info("Building and saving index...")
                tfidf_matrix, feature_names, df = indexer.build_index()
                indexer.save_index(tfidf_matrix, feature_names, df)
                log.info("Indexing completed successfully")
            else:
                log.warning("No documents were processed, index not created")
                
        except KeyboardInterrupt:
            log.info("Interrupted by user, shutting down")
        finally:
            consumer.close()
            log.info("Consumer closed")
        
        return 0
        
    except Exception as e:
        log.exception(f"Unhandled exception: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())

