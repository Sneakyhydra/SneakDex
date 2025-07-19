"""
Enhanced ModernIndexer service with advanced search capabilities:
- Optimized embedding generation with semantic enhancement
- Intelligent batch processing with error handling and retries
- Advanced deduplication and content quality filtering
- Performance monitoring and comprehensive logging
- Flexible configuration and resource management
"""

import logging
import uuid
import re
import time
import hashlib
from typing import List, Dict, Tuple, Set, Union
from urllib.parse import urlparse, unquote
from dataclasses import dataclass

import numpy as np
from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.http.models import (
    PointStruct,
    VectorParams,
    Distance,
    OptimizersConfigDiff,
)
from supabase import create_client, Client

from src.config import IndexerConfig

log = logging.getLogger("indexer")


@dataclass
class IndexingStats:
    """Track indexing performance and statistics"""

    total_docs: int = 0
    successful_docs: int = 0
    failed_docs: int = 0
    duplicate_docs: int = 0
    total_images: int = 0
    successful_images: int = 0
    failed_images: int = 0
    processing_time: float = 0.0
    embedding_time: float = 0.0


class ModernIndexer:
    def __init__(self, config: IndexerConfig, model_name: str = "all-MiniLM-L12-v2"):
        """
        Initialize embedding model, Qdrant collections, and Supabase client.
        """
        self.config = config
        self.collection_name = config.collection_name
        self.collection_name_images = config.collection_name_images

        # Performance settings
        self.max_embedding_length = getattr(config, "max_embedding_length", 8192)
        self.batch_size = getattr(config, "batch_size", 100)
        self.max_workers = getattr(config, "max_workers", 4)
        self.retry_attempts = getattr(config, "retry_attempts", 3)

        # Content quality thresholds
        self.min_content_length = getattr(config, "min_content_length", 0)
        self.max_heading_length = getattr(config, "max_heading_length", 200)

        # Initialize model with error handling
        self._initialize_model(model_name)

        # Initialize clients
        self._initialize_qdrant()
        self._initialize_supabase()

        # Cache for duplicate detection
        self._url_cache: Set[str] = set()
        self._content_hashes: Set[str] = set()

    def _initialize_model(self, model_name: str) -> None:
        """Initialize embedding model with proper error handling"""
        try:
            log.info(f"Loading embedding model: {model_name}")
            self.model = SentenceTransformer(model_name)
            self.vector_size = self.model.get_sentence_embedding_dimension() or 384
            log.info(f"Embedding dimension: {self.vector_size}")
        except Exception as e:
            log.error(f"Failed to initialize embedding model: {e}")
            raise

    def _initialize_qdrant(self) -> None:
        """Initialize Qdrant client and collections"""
        try:
            self.client = QdrantClient(
                url=self.config.qdrant_url,
                api_key=self.config.qdrant_api_key,
                timeout=30,  # Add timeout
            )
            self._ensure_qdrant_collections(self.vector_size)
            log.info("Connected to Qdrant successfully")
        except Exception as e:
            log.error(f"Failed to initialize Qdrant: {e}")
            raise

    def _initialize_supabase(self) -> None:
        """Initialize Supabase client"""
        try:
            self.supabase: Client = create_client(
                self.config.supabase_url, self.config.supabase_api_key
            )
            log.info("Connected to Supabase successfully")
        except Exception as e:
            log.error(f"Failed to initialize Supabase: {e}")
            raise

    def _ensure_qdrant_collections(self, vector_size: int) -> None:
        """Create collections in Qdrant if they don't already exist."""
        try:
            existing = {c.name for c in self.client.get_collections().collections}

            for cname in [self.collection_name, self.collection_name_images]:
                if cname not in existing:
                    log.info(f"Creating Qdrant collection: {cname}")
                    self.client.create_collection(
                        collection_name=cname,
                        vectors_config=VectorParams(
                            size=vector_size, distance=Distance.COSINE
                        ),
                        # Add indexing parameters for better performance
                        optimizers_config=OptimizersConfigDiff(
                            default_segment_number=2
                        ),
                        replication_factor=1,
                    )
                else:
                    log.info(f"Qdrant collection already exists: {cname}")
        except Exception as e:
            log.error(f"Failed to ensure Qdrant collections: {e}")
            raise

    def _validate_document(self, doc: dict) -> Tuple[bool, str]:
        """Validate document quality and completeness"""
        if not doc.get("url"):
            return False, "Missing URL"

        if not doc.get("title") and not doc.get("cleaned_text"):
            return False, "Missing both title and content"

        content = doc.get("cleaned_text", "")
        if content and len(content.strip()) < self.min_content_length:
            return False, f"Content too short ({len(content)} chars)"

        # Check for duplicate URLs
        url = doc.get("url", "")
        if url:
            url = url.strip()
        else:
            url = ""
        if url in self._url_cache:
            return False, "Duplicate URL"

        return True, "Valid"

    def _compute_content_hash(self, doc: dict) -> str:
        """Compute hash for content-based deduplication"""
        content = doc.get("cleaned_text", "")
        title = doc.get("title", "")
        combined = f"{title}|{content}".strip()
        return hashlib.md5(combined.encode()).hexdigest()

    def _clean_headings(self, headings: List[Dict]) -> List[Dict]:
        """Clean and validate headings"""
        cleaned = []
        for heading in headings[:5]:  # Limit to 5 headings
            if isinstance(heading, dict):
                text = heading.get("text", "")
                if text:
                    text = text.strip()
                else:
                    text = ""
                level = heading.get("level", 1)
                if not level:
                    level = 1
            else:
                text = str(heading)
                if text:
                    text = text.strip()
                else:
                    text = ""
                level = 1

            if text and len(text) <= self.max_heading_length:
                cleaned.append({"text": text, "level": level})

        return cleaned

    def build_embedding_text(self, doc: dict) -> str:
        """
        Build optimized text for embedding with enhanced semantic context.
        """
        pieces = []

        # 1. Title (highest weight) - repeat for emphasis
        title = doc.get("title", "")
        if title:
            title = title.strip()
            pieces.extend([title, title])  # Double weight for title

        # 2. Description - crucial for semantic understanding
        description = doc.get("description")
        if description:
            description = description.strip()
        else:
            description = ""
        if description and description != title:  # Avoid duplication
            pieces.append(description)

        # 3. Clean and structure headings
        headings = doc.get("headings", [])
        if headings:
            headings = self._clean_headings(headings)
            heading_texts = [h["text"] for h in headings]
            pieces.extend(heading_texts)

            # Create hierarchical context
            h1_headings = [h["text"] for h in headings if h.get("level") == 1]
            if len(h1_headings) > 1:
                pieces.append(" → ".join(h1_headings))

        # 4. Main content with smart truncation
        content = doc.get("cleaned_text", "")
        if content:
            content = content.strip()
            # For very long content, prioritize beginning and key sections
            if len(content) > 4000:
                # Take first 2000 chars and last 1000 chars with separator
                content = content[:2000] + " ... " + content[-1000:]
            pieces.append(content)

        # 5. URL semantic extraction
        url = doc.get("url", "")
        if url:
            url = url.strip()
            url_keywords = self._extract_url_keywords(url)
            if url_keywords:
                pieces.append(url_keywords)

            domain_context = self._extract_domain_context(url)
            if domain_context:
                pieces.append(domain_context)

        # 6. Metadata context
        self._add_metadata_context(doc, pieces)

        # Join and optimize
        embedding_text = " ".join(filter(None, pieces))
        if embedding_text:
            embedding_text = embedding_text.strip()

        # Smart truncation at sentence boundaries
        if len(embedding_text) > self.max_embedding_length:
            embedding_text = self._smart_truncate(
                embedding_text, self.max_embedding_length
            )

        return embedding_text

    def _add_metadata_context(self, doc: dict, pieces: List[str]) -> None:
        """Add relevant metadata context to embedding text"""
        content_type = doc.get("content_type", "")
        if content_type:
            content_type = content_type.strip()
        else:
            content_type = ""
        if content_type and content_type not in ["text/html", "text/plain"]:
            pieces.append(f"type:{content_type}")

        language = doc.get("language", "")
        if language:
            language = language.strip()
        else:
            language = ""
        if language and language not in ["en", "english", ""]:
            pieces.append(f"lang:{language}")

        # Add domain authority context for better search
        url = doc.get("url", "")
        if url:
            url = url.strip()
            domain = urlparse(url).netloc.lower()
            if any(tld in domain for tld in [".edu", ".gov", ".org"]):
                pieces.append("authoritative_source")

    def _smart_truncate(self, text: str, max_length: int) -> str:
        """Intelligently truncate text at sentence boundaries"""
        if len(text) <= max_length:
            return text

        # Try to truncate at sentence boundary
        truncated = text[:max_length]
        last_sentence = truncated.rfind(". ")
        last_newline = truncated.rfind("\n")

        boundary = max(last_sentence, last_newline)
        if boundary > max_length * 0.8:  # Only if we don't lose too much content
            temp = truncated[: boundary + 1]
            if temp:
                temp.strip()
            else:
                temp = ""
            return temp

        # Fallback to word boundary
        temp = truncated.rsplit(" ", 1)[0]
        if temp:
            temp = temp.strip()
        else:
            temp = ""
        return temp

    def _extract_url_keywords(self, url: str) -> str:
        """Extract meaningful keywords from URL path and parameters."""
        try:
            parsed = urlparse(url.lower())
            keywords = []

            # Extract from path with better filtering
            path_parts = []
            for part in parsed.path.split("/"):
                part = unquote(part)
                if part:
                    part = part.strip()
                else:
                    part = ""
                if part and not part.isdigit() and len(part) > 1:
                    # Clean common file extensions and IDs
                    part = re.sub(r"\.(html|htm|php|asp|jsp)$", "", part)
                    if not re.match(r"^[a-f0-9]{8,}$", part):  # Skip hex IDs
                        path_parts.append(part)

            keywords.extend(path_parts[:6])  # Limit path parts

            # Extract meaningful query parameters
            if parsed.query:
                meaningful_params = {
                    "q",
                    "query",
                    "search",
                    "category",
                    "type",
                    "tag",
                    "topic",
                    "subject",
                    "keyword",
                    "term",
                    "filter",
                    "section",
                }
                for param in parsed.query.split("&"):
                    if "=" in param:
                        key, value = param.split("=", 1)
                        if key.lower() in meaningful_params and len(value) > 1:
                            clean_value = unquote(value).replace("+", " ")
                            if len(clean_value) > 1 and not clean_value.isdigit():
                                keywords.append(clean_value)

            # Clean and normalize
            clean_keywords = []
            for kw in keywords:
                kw = re.sub(r"[-_]+", " ", kw)  # Replace separators with spaces
                kw = re.sub(r"\s+", " ", kw)  # Normalize whitespace
                if kw:
                    kw.strip()
                else:
                    kw = ""
                if len(kw) > 1 and not kw.isdigit():
                    clean_keywords.append(kw)

            return " ".join(clean_keywords[:8])  # Limit to prevent noise

        except Exception as e:
            log.warning(f"Failed to extract URL keywords from {url}: {e}")
            return ""

    def _extract_domain_context(self, url: str) -> str:
        """Extract semantic context from domain"""
        try:
            domain = urlparse(url.lower()).netloc.replace("www.", "")

            # Extract meaningful domain parts
            domain_parts = domain.split(".")
            if len(domain_parts) >= 2:
                main_domain = domain_parts[-2]  # e.g., 'github' from 'github.com'

                # Skip generic domains
                if main_domain not in {"www", "blog", "news", "docs", "wiki"}:
                    return f"site:{main_domain}"

            return ""
        except Exception:
            return ""

    def build_image_caption(self, img: dict) -> str:
        """Build enhanced text for embedding an image"""
        parts = []

        alt_text = img.get("alt", "")
        if alt_text:
            alt_text = alt_text.strip()
        else:
            alt_text = ""
        title_text = img.get("title", "")
        if title_text:
            title_text = title_text.strip()
        else:
            title_text = ""

        if alt_text:
            parts.append(alt_text)
        if title_text and title_text != alt_text:
            parts.append(title_text)

        # Add page context for better image search
        page_title = img.get("page_title", "")
        if page_title:
            page_title = page_title.strip()
        else:
            page_title = ""

        if page_title and len(parts) > 0:  # Only if we have image text
            parts.append(f"from: {page_title}")

        temp = " ".join(parts)
        if temp:
            temp.strip()
        else:
            temp = ""
        return temp

    def index_batch(self, documents: List[dict]) -> IndexingStats:
        """
        Enhanced batch indexing with comprehensive error handling and statistics.
        """
        stats = IndexingStats()
        stats.total_docs = len(documents)
        start_time = time.time()

        if not documents:
            log.warning("Received empty batch. Skipping indexing.")
            return stats

        log.info(f"Starting batch indexing of {len(documents)} documents")

        # Phase 1: Validation and deduplication
        valid_docs, valid_images = self._process_and_validate_batch(documents, stats)

        if not valid_docs:
            log.warning("No valid documents to index after validation")
            return stats

        # Phase 2: Generate embeddings
        embedding_start = time.time()
        doc_embeddings, img_embeddings = self._generate_embeddings(
            valid_docs, valid_images
        )
        stats.embedding_time = time.time() - embedding_start

        # Phase 3: Create points and upsert
        self._create_and_upsert_points(
            valid_docs, valid_images, doc_embeddings, img_embeddings, stats
        )

        stats.processing_time = time.time() - start_time

        # Log comprehensive statistics
        self._log_indexing_stats(stats)

        return stats

    def _process_and_validate_batch(
        self, documents: List[dict], stats: IndexingStats
    ) -> Tuple[List[dict], List[dict]]:
        """Process and validate documents and images"""
        valid_docs = []
        all_images = []

        for doc in documents:
            # Validate document
            is_valid, reason = self._validate_document(doc)
            if not is_valid:
                log.debug(f"Skipping document {doc.get('url', 'unknown')}: {reason}")
                if "duplicate" in reason.lower():
                    stats.duplicate_docs += 1
                else:
                    stats.failed_docs += 1
                continue

            # Content-based deduplication
            content_hash = self._compute_content_hash(doc)
            if content_hash in self._content_hashes:
                log.debug(f"Skipping duplicate content: {doc.get('url')}")
                stats.duplicate_docs += 1
                continue

            # Process document
            doc_id = self.generate_doc_id(doc.get("url", "empty_url"))
            doc["id"] = doc_id
            doc["content_hash"] = content_hash

            valid_docs.append(doc)
            self._url_cache.add(doc.get("url", "empty_url"))
            self._content_hashes.add(content_hash)

            # Process images
            images = self._extract_and_process_images(doc)
            all_images.extend(images)
            stats.total_images += len(images)

        stats.successful_docs = len(valid_docs)
        return valid_docs, all_images

    def _extract_and_process_images(self, doc: dict) -> List[dict]:
        """Extract and process images from document"""
        images = []

        for img in doc.get("images", []):
            src = img.get("src", "")
            if src:
                src = src.strip()
            else:
                src = ""
            if not src or src == "about:blank":
                continue

            img_id = self.generate_doc_id(src)
            enhanced_img = {
                **img,
                "id": img_id,
                "page_url": doc.get("url"),
                "page_title": doc.get("title", ""),
                "page_description": doc.get("description", ""),
                "timestamp": doc.get("timestamp"),
            }
            images.append(enhanced_img)

        return images

    def _generate_embeddings(
        self, documents: List[dict], images: List[dict]
    ) -> Tuple[Union[np.ndarray, List], Union[np.ndarray, List]]:
        """Generate embeddings with error handling and batching"""
        doc_embeddings = []
        img_embeddings = []

        if documents:
            log.info(f"Generating embeddings for {len(documents)} documents")
            try:
                texts = [self.build_embedding_text(doc) for doc in documents]
                doc_embeddings = self.model.encode(
                    texts,
                    show_progress_bar=len(documents) > 10,
                    batch_size=min(32, len(documents)),  # Optimize batch size
                )
            except Exception as e:
                log.error(f"Failed to generate document embeddings: {e}")
                raise

        if images:
            log.info(f"Generating embeddings for {len(images)} images")
            try:
                captions = [self.build_image_caption(img) for img in images]
                valid_captions = [c for c in captions if c and c.strip()]

                if valid_captions:
                    img_embeddings = self.model.encode(
                        valid_captions,
                        show_progress_bar=len(valid_captions) > 10,
                        batch_size=min(32, len(valid_captions)),
                    )
            except Exception as e:
                log.error(f"Failed to generate image embeddings: {e}")
                # Don't raise for images, continue without them

        return doc_embeddings, img_embeddings

    def _create_and_upsert_points(
        self,
        documents: List[dict],
        images: List[dict],
        doc_embeddings: Union[np.ndarray, List],
        img_embeddings: Union[np.ndarray, List],
        stats: IndexingStats,
    ) -> None:
        """Create points and upsert to Qdrant"""

        # Create document points
        doc_points = []
        supabase_rows = []

        for i, doc in enumerate(documents):
            try:
                payload = self._create_document_payload(doc)
                # Handle both numpy array and list cases
                embedding_vector = doc_embeddings[i]
                if hasattr(embedding_vector, "tolist"):
                    embedding_vector = embedding_vector.tolist()

                point = PointStruct(
                    id=doc["id"],
                    vector=embedding_vector,
                    payload=payload,
                )
                doc_points.append(point)

                # Prepare Supabase row
                supabase_rows.append(self._create_supabase_row(doc))

            except Exception as e:
                log.error(f"Failed to create point for document {doc.get('url')}: {e}")
                stats.failed_docs += 1

        # Create image points
        img_points = []
        for i, img in enumerate(images):
            if i < len(img_embeddings):  # Ensure we have embedding
                try:
                    payload = self._create_image_payload(img)
                    # Handle both numpy array and list cases
                    embedding_vector = img_embeddings[i]
                    if hasattr(embedding_vector, "tolist"):
                        embedding_vector = embedding_vector.tolist()

                    point = PointStruct(
                        id=img["id"],
                        vector=embedding_vector,
                        payload=payload,
                    )
                    img_points.append(point)
                    stats.successful_images += 1
                except Exception as e:
                    log.error(f"Failed to create point for image {img.get('src')}: {e}")
                    stats.failed_images += 1

        # Upsert to Qdrant
        self._upsert_qdrant_with_retry(self.collection_name, doc_points, "documents")

        if img_points:
            # Batch image upserts for better performance
            for i in range(0, len(img_points), self.batch_size):
                batch = img_points[i : i + self.batch_size]
                self._upsert_qdrant_with_retry(
                    self.collection_name_images, batch, "images"
                )

        # Upsert to Supabase (optional, commented out in original)
        self._upsert_supabase_with_retry(supabase_rows)

    def _create_document_payload(self, doc: dict) -> dict:
        """Create optimized document payload"""
        return {
            "url": doc.get("url"),
            "title": doc.get("title", ""),
            "description": doc.get("description", ""),
            "headings": self._clean_headings(doc.get("headings", []))[:3],
            "images": doc.get("images", [])[:3],  # Limit payload size
            "language": doc.get("language", ""),
            "timestamp": doc.get("timestamp"),
            "content_type": doc.get("content_type"),
            "text_snippet": doc.get("cleaned_text", "")[:500],
            "word_count": len(doc.get("cleaned_text", "").split()),
            "content_hash": doc.get("content_hash"),
        }

    def _create_image_payload(self, img: dict) -> dict:
        """Create optimized image payload"""
        return {
            "src": img.get("src"),
            "alt": img.get("alt", ""),
            "title": img.get("title", ""),
            "caption": self.build_image_caption(img),
            "page_url": img.get("page_url"),
            "page_title": img.get("page_title", ""),
            "page_description": img.get("page_description", ""),
            "timestamp": img.get("timestamp"),
        }

    def _create_supabase_row(self, doc: dict) -> dict:
        """Create Supabase row with proper language handling"""
        language = doc.get("language", "simple")
        if language in {"chinese", "japanese", None, ""}:
            language = "simple"

        return {
            "id": doc["id"],
            "url": doc.get("url"),
            "title": doc.get("title"),
            "lang": language,
            "_tmp_content": doc.get("cleaned_text", ""),
        }

    def _upsert_qdrant_with_retry(
        self, collection: str, points: List[PointStruct], label: str
    ) -> None:
        """Upsert points to Qdrant with retry logic"""
        if not points:
            log.info(f"No {label} to index into Qdrant.")
            return

        for attempt in range(self.retry_attempts):
            try:
                self.client.upsert(collection_name=collection, points=points)
                log.info(
                    f"✅ Indexed {len(points)} {label} into Qdrant collection '{collection}'."
                )
                return
            except Exception as e:
                log.warning(
                    f"Attempt {attempt + 1} failed to upsert {label} to Qdrant: {e}"
                )
                if attempt == self.retry_attempts - 1:
                    log.error(
                        f"❌ Failed to upsert {label} to Qdrant after {self.retry_attempts} attempts"
                    )
                    raise
                time.sleep(2**attempt)  # Exponential backoff

    def _upsert_supabase_with_retry(self, rows: List[dict]) -> None:
        """Upsert rows to Supabase with retry logic"""
        if not rows:
            log.info("No rows to insert into Supabase.")
            return

        for attempt in range(self.retry_attempts):
            try:
                response = self.supabase.table("documents").upsert(rows).execute()
                if not getattr(response, "data", None):
                    log.error(f"Supabase insert returned no data. Response: {response}")
                else:
                    log.info(f"✅ Inserted {len(response.data)} rows into Supabase.")
                return
            except Exception as e:
                log.warning(
                    f"Attempt {attempt + 1} failed to upsert rows to Supabase: {e}"
                )
                if attempt == self.retry_attempts - 1:
                    log.error(
                        f"❌ Failed to upsert rows to Supabase after {self.retry_attempts} attempts"
                    )
                    raise
                time.sleep(2**attempt)

    def _log_indexing_stats(self, stats: IndexingStats) -> None:
        """Log comprehensive indexing statistics"""
        success_rate = (stats.successful_docs / max(stats.total_docs, 1)) * 100

        log.info("=" * 60)
        log.info("INDEXING STATISTICS")
        log.info("=" * 60)
        log.info(
            f"📊 Documents: {stats.successful_docs}/{stats.total_docs} successful ({success_rate:.1f}%)"
        )
        log.info(
            f"📊 Images: {stats.successful_images}/{stats.total_images} successful"
        )
        log.info(f"📊 Duplicates: {stats.duplicate_docs} documents")
        log.info(
            f"📊 Failed: {stats.failed_docs} documents, {stats.failed_images} images"
        )
        log.info(f"⏱️  Total time: {stats.processing_time:.2f}s")
        log.info(
            f"⏱️  Embedding time: {stats.embedding_time:.2f}s ({stats.embedding_time/stats.processing_time*100:.1f}%)"
        )
        log.info(
            f"⚡ Processing rate: {stats.successful_docs/max(stats.processing_time, 0.01):.2f} docs/sec"
        )
        log.info("=" * 60)

    def get_collection_info(self) -> dict:
        """Get detailed information about collections"""
        try:
            doc_info = self.client.get_collection(self.collection_name)
            img_info = self.client.get_collection(self.collection_name_images)

            return {
                "documents": {
                    "count": doc_info.vectors_count or 0,
                    "status": doc_info.status,
                },
                "images": {
                    "count": img_info.vectors_count or 0,
                    "status": img_info.status,
                },
            }
        except Exception as e:
            log.error(f"Failed to get collection info: {e}")
            return {}

    def count(self) -> int:
        """Returns the number of vectors in the main Qdrant collection."""
        try:
            stats = self.client.get_collection(self.collection_name)
            return stats.vectors_count or 0
        except Exception as e:
            log.error(f"Failed to get vector count from Qdrant: {e}")
            return -1

    def cleanup_cache(self) -> None:
        """Clear internal caches to free memory"""
        self._url_cache.clear()
        self._content_hashes.clear()
        log.info("Cleared internal caches")

    @staticmethod
    def generate_doc_id(source: str) -> str:
        """Generate a UUID5 for a given string."""
        return str(uuid.uuid5(uuid.NAMESPACE_URL, source))

    def __enter__(self):
        """Context manager entry"""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - cleanup resources"""
        self.cleanup_cache()
        log.info("ModernIndexer resources cleaned up")
