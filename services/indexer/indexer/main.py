#!/usr/bin/env python3
"""
Parallel SneakDex Indexer service with incremental indexing.

This service:
1. Consumes parsed web pages from Kafka in parallel
2. Processes text using NLP techniques with multiprocessing
3. Builds and updates TF-IDF index incrementally
4. Persists the index periodically
"""

import json
import logging
import os
import sys
import time
import threading
from collections import defaultdict, deque
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor, as_completed
from multiprocessing import Manager, Queue, Process, cpu_count
from pathlib import Path
from typing import Dict, List, Optional, Set, Tuple, Union
import queue

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
from sklearn.feature_extraction.text import HashingVectorizer
from tqdm import tqdm
import threading
from threading import Lock, RLock

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
    
    # Parallel processing settings
    num_worker_threads: int = Field(default=4, env="NUM_WORKER_THREADS")
    num_consumer_threads: int = Field(default=2, env="NUM_CONSUMER_THREADS")
    preprocessing_workers: int = Field(default=cpu_count(), env="PREPROCESSING_WORKERS")
    index_save_interval: int = Field(default=5000, env="INDEX_SAVE_INTERVAL")  # Save every N documents
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


class ThreadSafeDocumentStore:
    """Thread-safe document store for parallel processing."""
    
    def __init__(self):
        self._lock = RLock()
        self.documents = []
        self.urls = []
        self.titles = []
        self.document_count = 0
        self.processed_documents = deque()  # For batch processing
    
    def add_document(self, doc_text: str, url: str, title: str) -> int:
        """Add a document to the store. Returns current document count."""
        with self._lock:
            self.documents.append(doc_text)
            self.urls.append(url)
            self.titles.append(title)
            self.document_count += 1
            
            # Add to processing queue
            self.processed_documents.append({
                'text': doc_text,
                'url': url,
                'title': title,
                'index': self.document_count - 1
            })
            
            return self.document_count
    
    def get_batch(self, batch_size: int) -> List[Dict]:
        """Get a batch of unprocessed documents."""
        with self._lock:
            batch = []
            for _ in range(min(batch_size, len(self.processed_documents))):
                if self.processed_documents:
                    batch.append(self.processed_documents.popleft())
            return batch
    
    def get_dataframe(self) -> pd.DataFrame:
        """Convert stored documents to a DataFrame."""
        with self._lock:
            return pd.DataFrame({
                "url": self.urls.copy(),
                "title": self.titles.copy(),
                "text": self.documents.copy()
            })
    
    def get_document_count(self) -> int:
        """Get current document count thread-safely."""
        with self._lock:
            return self.document_count


class ParallelPreprocessor:
    """Parallel text preprocessing using multiprocessing."""
    
    def __init__(self, num_workers: int = None):
        self.num_workers = num_workers or cpu_count()
        
        # Initialize NLTK resources in main process
        try:
            nltk.data.find('tokenizers/punkt')
            nltk.data.find('corpora/stopwords')
        except LookupError:
            nltk.download('punkt')
            nltk.download('stopwords')
    
    @staticmethod
    def _preprocess_text(text: str) -> str:
        """Static method for preprocessing individual text."""
        try:
            # Initialize stemmer and stopwords in worker process
            stemmer = PorterStemmer()
            stop_words = set(stopwords.words('english'))
            
            # Tokenize
            tokens = word_tokenize(text.lower())
            
            # Remove stopwords and stem
            processed_tokens = [
                stemmer.stem(token)
                for token in tokens
                if token.isalnum() and token not in stop_words
            ]
            
            return " ".join(processed_tokens)
        except Exception as e:
            log.error(f"Error preprocessing text: {e}")
            return ""
    
    def preprocess_batch(self, texts: List[str]) -> List[str]:
        """Preprocess a batch of texts in parallel."""
        if not texts:
            return []
        
        with ProcessPoolExecutor(max_workers=self.num_workers) as executor:
            processed_texts = list(executor.map(self._preprocess_text, texts))
        
        return processed_texts


class IncrementalTfIdfIndexer:
    """Incremental TF-IDF indexer that can update the index as new documents arrive."""
    
    def __init__(self, config: IndexerConfig):
        self.config = config
        self.lock = RLock()
        
        # Use HashingVectorizer for incremental processing
        self.vectorizer = HashingVectorizer(
            n_features=config.max_features or 100000,
            stop_words='english',
            norm='l2',
            alternate_sign=False
        )
        
        # For final TF-IDF computation
        self.tfidf_vectorizer = None
        
        # Document storage
        self.document_texts = []
        self.document_urls = []
        self.document_titles = []
        
        # Index matrices
        self.hash_matrix = None
        self.document_count = 0
    
    def add_documents_batch(self, batch_data: List[Dict]) -> None:
        """Add a batch of documents to the index."""
        if not batch_data:
            return
        
        with self.lock:
            texts = [doc['text'] for doc in batch_data]
            urls = [doc['url'] for doc in batch_data]
            titles = [doc['title'] for doc in batch_data]
            
            # Transform texts to hash vectors
            hash_vectors = self.vectorizer.transform(texts)
            
            # Store document data
            self.document_texts.extend(texts)
            self.document_urls.extend(urls)
            self.document_titles.extend(titles)
            
            # Update hash matrix
            if self.hash_matrix is None:
                self.hash_matrix = hash_vectors
            else:
                # Stack vertically to add new documents
                from scipy import sparse
                self.hash_matrix = sparse.vstack([self.hash_matrix, hash_vectors])
            
            self.document_count += len(batch_data)
            
            log.info(f"Added batch of {len(batch_data)} documents. Total: {self.document_count}")
    
    def build_final_tfidf_index(self) -> Tuple[np.ndarray, List[str], pd.DataFrame]:
        """Build final TF-IDF index from all collected documents."""
        with self.lock:
            if not self.document_texts:
                raise ValueError("No documents to index")
            
            log.info(f"Building final TF-IDF index from {len(self.document_texts)} documents")
            
            # Create proper TF-IDF vectorizer
            self.tfidf_vectorizer = TfidfVectorizer(
                min_df=self.config.tf_idf_min_df,
                max_features=self.config.max_features,
                stop_words='english',
                use_idf=True,
                norm='l2'
            )
            
            # Create TF-IDF matrix
            tfidf_matrix = self.tfidf_vectorizer.fit_transform(self.document_texts)
            
            # Get feature names
            feature_names = self.tfidf_vectorizer.get_feature_names_out()
            
            # Create document dataframe
            df = pd.DataFrame({
                'url': self.document_urls,
                'title': self.document_titles,
                'text': self.document_texts
            })
            
            log.info(f"Created final TF-IDF matrix with shape {tfidf_matrix.shape}")
            log.info(f"Vocabulary size: {len(feature_names)}")
            
            return tfidf_matrix, feature_names, df
    
    def save_intermediate_index(self, output_path: Path) -> None:
        """Save intermediate index state."""
        with self.lock:
            if self.hash_matrix is None or self.document_count == 0:
                return
            
            output_path.mkdir(parents=True, exist_ok=True)
            
            # Save hash matrix and document metadata
            joblib.dump(self.hash_matrix, output_path / 'intermediate_hash_matrix.joblib')
            
            df = pd.DataFrame({
                'url': self.document_urls,
                'title': self.document_titles
            })
            df.to_csv(output_path / 'intermediate_metadata.csv', index=False)
            
            # Save progress info
            with open(output_path / 'progress.json', 'w') as f:
                json.dump({
                    'document_count': self.document_count,
                    'timestamp': time.time()
                }, f)
            
            log.info(f"Saved intermediate index with {self.document_count} documents")


class ParallelIndexer:
    """Main parallel indexer orchestrating all components."""
    
    def __init__(self, config: IndexerConfig):
        self.config = config
        self.document_store = ThreadSafeDocumentStore()
        self.preprocessor = ParallelPreprocessor(config.preprocessing_workers)
        self.indexer = IncrementalTfIdfIndexer(config)
        
        # Threading control
        self.stop_event = threading.Event()
        self.processing_queue = queue.Queue(maxsize=1000)
        
        # Statistics
        self.processed_count = 0
        self.last_save_count = 0
    
    def process_message(self, message: dict) -> None:
        """Process a single parsed page message."""
        try:
            url = message.get('url', '')
            title = message.get('title', 'No Title')
            body_text = message.get('body_text', '')
            
            if not body_text or not url:
                log.warning(f"Skipping document with empty body or URL: {url}")
                return
            
            # Add to processing queue
            self.processing_queue.put({
                'url': url,
                'title': title,
                'body_text': body_text
            })
            
        except Exception as e:
            log.error(f"Error processing message: {e}")
    
    def _preprocessing_worker(self):
        """Worker thread for preprocessing documents."""
        batch = []
        batch_size = self.config.batch_size // 4  # Smaller batches for more frequent updates
        
        while not self.stop_event.is_set():
            try:
                # Collect batch
                timeout = 2.0
                while len(batch) < batch_size and not self.stop_event.is_set():
                    try:
                        item = self.processing_queue.get(timeout=timeout)
                        batch.append(item)
                        self.processing_queue.task_done()
                    except queue.Empty:
                        break
                
                if not batch:
                    continue
                
                # Preprocess batch
                texts = [item['body_text'] for item in batch]
                processed_texts = self.preprocessor.preprocess_batch(texts)
                
                # Create batch data for indexer
                batch_data = []
                for i, item in enumerate(batch):
                    if processed_texts[i]:  # Only add if preprocessing succeeded
                        batch_data.append({
                            'text': processed_texts[i],
                            'url': item['url'],
                            'title': item['title']
                        })
                
                # Add to indexer
                if batch_data:
                    self.indexer.add_documents_batch(batch_data)
                    self.processed_count += len(batch_data)
                    
                    # Check if we should save intermediate index
                    if (self.processed_count - self.last_save_count) >= self.config.index_save_interval:
                        self._save_intermediate_index()
                        self.last_save_count = self.processed_count
                
                # Clear batch
                batch = []
                
            except Exception as e:
                log.error(f"Error in preprocessing worker: {e}")
                batch = []  # Reset batch on error
    
    def _save_intermediate_index(self):
        """Save intermediate index state."""
        try:
            output_path = Path(self.config.index_output_path)
            self.indexer.save_intermediate_index(output_path)
        except Exception as e:
            log.error(f"Error saving intermediate index: {e}")
    
    def start_workers(self):
        """Start worker threads."""
        self.workers = []
        
        # Start preprocessing workers
        for i in range(self.config.num_worker_threads):
            worker = threading.Thread(
                target=self._preprocessing_worker,
                name=f"PreprocessWorker-{i}"
            )
            worker.daemon = True
            worker.start()
            self.workers.append(worker)
            
        log.info(f"Started {len(self.workers)} worker threads")
    
    def stop_workers(self):
        """Stop all worker threads."""
        log.info("Stopping worker threads...")
        self.stop_event.set()
        
        # Wait for workers to finish
        for worker in self.workers:
            worker.join(timeout=5.0)
        
        # Process any remaining items in queue
        remaining_items = []
        while not self.processing_queue.empty():
            try:
                remaining_items.append(self.processing_queue.get_nowait())
            except queue.Empty:
                break
        
        if remaining_items:
            log.info(f"Processing {len(remaining_items)} remaining items...")
            texts = [item['body_text'] for item in remaining_items]
            processed_texts = self.preprocessor.preprocess_batch(texts)
            
            batch_data = []
            for i, item in enumerate(remaining_items):
                if processed_texts[i]:
                    batch_data.append({
                        'text': processed_texts[i],
                        'url': item['url'],
                        'title': item['title']
                    })
            
            if batch_data:
                self.indexer.add_documents_batch(batch_data)
                self.processed_count += len(batch_data)
        
        log.info("All workers stopped")
    
    def build_and_save_final_index(self) -> None:
        """Build and save the final TF-IDF index."""
        try:
            log.info("Building final TF-IDF index...")
            tfidf_matrix, feature_names, df = self.indexer.build_final_tfidf_index()
            
            # Save final index
            output_path = Path(self.config.index_output_path)
            output_path.mkdir(parents=True, exist_ok=True)
            
            log.info(f"Saving final index to {output_path}")
            
            # Save TF-IDF matrix
            joblib.dump(tfidf_matrix, output_path / 'tfidf_matrix.joblib')
            
            # Save feature names (vocabulary)
            joblib.dump(feature_names, output_path / 'feature_names.joblib')
            
            # Save document metadata
            df[['url', 'title']].to_csv(output_path / 'document_metadata.csv', index=False)
            
            # Save vectorizer
            joblib.dump(self.indexer.tfidf_vectorizer, output_path / 'vectorizer.joblib')
            
            # Clean up intermediate files
            for intermediate_file in ['intermediate_hash_matrix.joblib', 'intermediate_metadata.csv', 'progress.json']:
                try:
                    (output_path / intermediate_file).unlink(missing_ok=True)
                except Exception:
                    pass
            
            log.info("Final index saved successfully")
            
        except Exception as e:
            log.error(f"Error building final index: {e}")
            raise


def main():
    """Main entry point for the parallel indexer service."""
    try:
        # Load configuration
        config = IndexerConfig()
        log.info(f"Starting parallel indexer service with config: {config.dict()}")
        
        # Create parallel indexer
        indexer = ParallelIndexer(config)
        
        # Start worker threads
        indexer.start_workers()
        
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
        
        last_checkpoint = 0  # Track when we last saved

        try:
            while True:
                if config.max_docs and message_count >= config.max_docs:
                    log.info(f"Reached maximum document count ({config.max_docs}), stopping consumption")
                    break
                
                msg = consumer.poll(timeout=5.0)  # wait up to 5 seconds for a new message
                
                if msg is None:
                    log.info("No message received. Waiting for new messages...")
                    continue  # Instead of retry_count, just keep listening
                
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        log.info("Reached end of partition")
                        continue
                    else:
                        log.error(f"Consumer error: {msg.error()}")
                        continue
                
                try:
                    value = msg.value().decode('utf-8')
                    parsed_page = json.loads(value)
                    
                    indexer.process_message(parsed_page)
                    message_count += 1
                    
                    # Save every 100 newly processed docs
                    current_checkpoint = indexer.processed_count
                    if current_checkpoint - last_checkpoint >= 100:
                        log.info(f"Checkpoint: Consumed {message_count} messages, processed {current_checkpoint} documents")
                        indexer.build_and_save_final_index()
                        last_checkpoint = current_checkpoint
                        log.info("Partial index saved")

                except Exception as e:
                    log.error(f"Error processing message: {e}")
            
            # Stop workers and process remaining items
            indexer.stop_workers()
            
            # Build and save final index if we have documents
            if indexer.processed_count > 0:
                indexer.build_and_save_final_index()
                log.info(f"Parallel indexing completed successfully. Total documents: {indexer.processed_count}")
            else:
                log.warning("No documents were processed, index not created")
                
        except KeyboardInterrupt:
            log.info("Interrupted by user, shutting down...")
            indexer.stop_workers()
            
            # Still try to save what we have
            if indexer.processed_count > 0:
                log.info("Saving partial index...")
                indexer.build_and_save_final_index()
                
        finally:
            consumer.close()
            log.info("Consumer closed")
        
        return 0
        
    except Exception as e:
        log.exception(f"Unhandled exception: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())