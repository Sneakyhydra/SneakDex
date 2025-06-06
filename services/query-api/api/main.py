#!/usr/bin/env python3
"""
Main module for the SneakDex Query API service.

This service:
1. Loads the TF-IDF index created by the Indexer service
2. Provides a FastAPI endpoint for searching
3. Implements Redis caching for frequently used queries
4. Handles error cases and provides a health check endpoint
"""

import json
import logging
import os
import sys
import time
from pathlib import Path
from typing import Dict, List, Optional, Union
import asyncio

import joblib
import nltk
import numpy as np
import pandas as pd
import redis
import uvicorn
from fastapi import FastAPI, HTTPException, Query, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from nltk.corpus import stopwords
from nltk.stem import PorterStemmer
from nltk.tokenize import word_tokenize
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings
from rich.console import Console
from rich.logging import RichHandler
from scipy.sparse import csr_matrix
from sklearn.feature_extraction.text import TfidfVectorizer
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.errors import RateLimitExceeded
from slowapi.util import get_remote_address
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response

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
log = logging.getLogger("query-api")


class APIConfig(BaseSettings):
    """Configuration for the Query API service."""
    
    # Redis settings
    redis_host: str = Field(default="redis", env="REDIS_HOST")
    redis_port: int = Field(default=6379, env="REDIS_PORT")
    redis_db: int = Field(default=0, env="REDIS_DB")
    redis_cache_ttl: int = Field(default=3600, env="REDIS_CACHE_TTL")  # 1 hour
    
    # Index settings
    index_path: str = Field(default="/app/data/index", env="INDEX_PATH")
    
    # API settings
    search_result_limit: int = Field(default=20, env="SEARCH_RESULT_LIMIT")
    enable_cors: bool = Field(default=True, env="ENABLE_CORS")
    allowed_origins: List[str] = Field(default=["*"], env="ALLOWED_ORIGINS")
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


class SearchResult(BaseModel):
    """Model for a single search result."""
    
    url: str
    title: str
    score: float


class SearchResponse(BaseModel):
    """Model for the search response."""
    
    query: str
    results: List[SearchResult]
    total_results: int
    time_ms: float


class Index:
    """Handles loading and searching the TF-IDF index."""
    
    def __init__(self, config: APIConfig):
        self.config = config
        self.index_path = Path(config.index_path)
        self.tfidf_matrix = None
        self.feature_names = None
        self.document_metadata = None
        self.vectorizer = None
        self.initialized = False
        
        # Initialize NLTK resources
        try:
            nltk.data.find('tokenizers/punkt')
            nltk.data.find('corpora/stopwords')
        except LookupError:
            nltk.download('punkt')
            nltk.download('stopwords')
        
        self.stemmer = PorterStemmer()
        self.stop_words = set(stopwords.words('english'))
    
    def initialize(self) -> bool:
        """Load the index from disk."""
        try:
            log.info(f"Loading index from {self.index_path}")
            
            # Load TF-IDF matrix
            tfidf_matrix_path = self.index_path / 'tfidf_matrix.joblib'
            self.tfidf_matrix = joblib.load(tfidf_matrix_path)
            log.info(f"Loaded TF-IDF matrix with shape {self.tfidf_matrix.shape}")
            
            # Load feature names
            feature_names_path = self.index_path / 'feature_names.joblib'
            self.feature_names = joblib.load(feature_names_path)
            log.info(f"Loaded {len(self.feature_names)} feature names")
            
            # Load document metadata
            metadata_path = self.index_path / 'document_metadata.csv'
            self.document_metadata = pd.read_csv(metadata_path)
            log.info(f"Loaded metadata for {len(self.document_metadata)} documents")
            
            # Load vectorizer
            vectorizer_path = self.index_path / 'vectorizer.joblib'
            self.vectorizer = joblib.load(vectorizer_path)
            log.info("Loaded vectorizer")
            
            self.initialized = True
            return True
            
        except Exception as e:
            log.error(f"Failed to load index: {e}")
            return False
    
    def preprocess_query(self, query: str) -> str:
        """Preprocess a query string."""
        # Tokenize
        tokens = word_tokenize(query.lower())
        
        # Remove stopwords and stem
        processed_tokens = [
            self.stemmer.stem(token)
            for token in tokens
            if token.isalnum() and token not in self.stop_words
        ]
        
        return " ".join(processed_tokens)
    
    def search(self, query: str, limit: int = None) -> List[Dict]:
        """Search the index for the given query."""
        if not self.initialized:
            if not self.initialize():
                raise RuntimeError("Index not initialized")
        
        if limit is None:
            limit = self.config.search_result_limit
        
        # Preprocess query
        processed_query = self.preprocess_query(query)
        
        # Transform query to TF-IDF space
        query_vec = self.vectorizer.transform([processed_query])
        
        # Compute cosine similarity
        cosine_similarities = (query_vec * self.tfidf_matrix.T).toarray().flatten()
        
        # Get top results
        top_indices = cosine_similarities.argsort()[-limit:][::-1]
        
        # Prepare results
        results = []
        for idx in top_indices:
            if cosine_similarities[idx] > 0:  # Only include non-zero scores
                results.append({
                    "url": self.document_metadata.iloc[idx]["url"],
                    "title": self.document_metadata.iloc[idx]["title"],
                    "score": float(cosine_similarities[idx])
                })
        
        return results
    
    def get_file_mod_times(self):
        """Return last modified times of index files."""
        return {
            'tfidf': os.path.getmtime(self.index_path / 'tfidf_matrix.joblib'),
            'features': os.path.getmtime(self.index_path / 'feature_names.joblib'),
            'metadata': os.path.getmtime(self.index_path / 'document_metadata.csv'),
            'vectorizer': os.path.getmtime(self.index_path / 'vectorizer.joblib'),
        }
    
    def check_and_reload_if_updated(self):
        """Check if index files have been updated and reload them."""
        try:
            current_mod_times = self.get_file_mod_times()
            if not hasattr(self, 'last_mod_times'):
                self.last_mod_times = current_mod_times
                return
            
            if current_mod_times != self.last_mod_times:
                log.info("Index files changed. Reloading index...")
                if self.initialize():
                    self.last_mod_times = current_mod_times
                    log.info("Index reloaded successfully.")
                else:
                    log.warning("Index reload failed.")
        except Exception as e:
            log.error(f"Error checking index file updates: {e}")


# Create FastAPI app
app = FastAPI(
    title="SneakDex Search API",
    description="API for searching the SneakDex index",
    version="0.1.0"
)

# Set up rate limiting
limiter = Limiter(key_func=get_remote_address)
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

# Initialize config
config = APIConfig()

# Set up CORS middleware
if config.enable_cors:
    app.add_middleware(
        CORSMiddleware,
        allow_origins=config.allowed_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

# Initialize Redis client
redis_client = redis.Redis(
    host=config.redis_host,
    port=config.redis_port,
    db=config.redis_db,
    decode_responses=True
)

# Initialize index
index = Index(config)

async def index_reloader():
    """Background task to reload index if files are updated."""
    while True:
        await asyncio.sleep(10)  # Check every 10 seconds
        index.check_and_reload_if_updated()

@app.on_event("startup")
async def startup_event():
    """Initialize the index on startup."""
    log.info("Starting up Query API service")
    
    # Test Redis connection
    try:
        redis_client.ping()
        log.info("Connected to Redis successfully")
    except redis.ConnectionError:
        log.warning("Could not connect to Redis, caching will be disabled")
    
    # Initialize index
    if not index.initialize():
        log.warning("Index initialization failed, search will not work until index is available")

    asyncio.create_task(index_reloader())

@app.get("/health")
async def health_check():
    """Health check endpoint."""
    status = {
        "status": "healthy",
        "redis_connected": False,
        "index_initialized": index.initialized
    }
    
    # Check Redis connection
    try:
        if redis_client.ping():
            status["redis_connected"] = True
    except:
        pass
    
    if not status["index_initialized"] or not status["redis_connected"]:
        status["status"] = "degraded"
    
    return status


@app.get("/search", response_model=SearchResponse)
@limiter.limit("30/minute")
async def search(
    request: Request,
    q: str = Query(..., description="Search query", min_length=1),
    limit: Optional[int] = Query(None, description="Maximum number of results to return", ge=1, le=100)
):
    """Search endpoint."""
    start_time = time.time()
    
    if limit is None:
        limit = config.search_result_limit
    
    # Check cache
    cache_key = f"search:{q}:{limit}"
    cached_result = None
    
    try:
        cached = redis_client.get(cache_key)
        if cached:
            cached_result = json.loads(cached)
            log.info(f"Cache hit for query: {q}")
    except Exception as e:
        log.warning(f"Redis error: {e}")
    
    # If not in cache, perform search
    if not cached_result:
        try:
            results = index.search(q, limit)
            
            response = {
                "query": q,
                "results": results,
                "total_results": len(results),
                "time_ms": (time.time() - start_time) * 1000
            }
            
            # Cache result
            try:
                redis_client.setex(
                    cache_key,
                    config.redis_cache_ttl,
                    json.dumps(response)
                )
            except Exception as e:
                log.warning(f"Redis caching error: {e}")
            
            return response
            
        except Exception as e:
            log.error(f"Search error: {e}")
            raise HTTPException(status_code=500, detail="Search failed")
    
    # Return cached result
    cached_result["time_ms"] = (time.time() - start_time) * 1000
    return cached_result


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    """Handle HTTP exceptions."""
    return JSONResponse(
        status_code=exc.status_code,
        content={"detail": exc.detail}
    )


@app.exception_handler(Exception)
async def general_exception_handler(request: Request, exc: Exception):
    """Handle general exceptions."""
    log.exception(f"Unhandled exception: {exc}")
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error"}
    )


if __name__ == "__main__":
    uvicorn.run("api.main:app", host="0.0.0.0", port=8000, log_level="info")

