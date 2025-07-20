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
from typing import List, Dict
from urllib.parse import urlparse, unquote
from dataclasses import dataclass

from src.monitor import MESSAGES_CONSUMED

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
    successful_docs_supabase: int = 0
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

    def _clean_headings(self, headings: List[Dict]) -> List[Dict]:
        """Clean and validate headings"""
        cleaned = []
        for heading in headings:
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

        # 2. URL semantic extraction
        url = doc.get("url", "")
        if url:
            url = url.strip()
            url_keywords = self._extract_url_keywords(url)
            if url_keywords:
                pieces.append(url_keywords)

            domain_context = self._extract_domain_context(url)
            if domain_context:
                pieces.extend([domain_context, domain_context, domain_context])

        # 3. Description - crucial for semantic understanding
        description = doc.get("description")
        if description:
            description = description.strip()
        else:
            description = ""
        if description and description != title:  # Avoid duplication
            pieces.append(description)

        # 4. Clean and structure headings
        headings = doc.get("headings", [])
        if headings:
            headings = self._clean_headings(headings)
            heading_texts = [h["text"] for h in headings]
            pieces.extend(heading_texts)

            # Create hierarchical context
            h1_headings = [h["text"] for h in headings if h.get("level") == 1]
            if len(h1_headings) > 1:
                pieces.append(" → ".join(h1_headings))

        # 5. Main content with smart truncation
        content = doc.get("cleaned_text", "")
        if content:
            content = content.strip()
            # For very long content, prioritize beginning and key sections
            if len(content) > 4000:
                # Take first 2000 chars and last 1000 chars with separator
                content = content[:2000] + " ... " + content[-1000:]
            pieces.append(content)

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
        """
        Build text for embedding an image: alt + title.
        """
        return " ".join(
            filter(None, [img.get("alt", ""), img.get("title", "")])
        ).strip()

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

        embedding_start = time.time()

        try:
            points, image_points, supabase_rows = [], [], []
            unique_docs = {}
            unique_imgs = {}

            for i, doc in enumerate(documents):
                doc_id = self.generate_doc_id(doc.get("url", f"doc-{i}"))
                doc["id"] = doc_id
                unique_docs[doc_id] = doc

                images = [
                    img
                    for img in doc.get("images", [])
                    if img.get("src") and img.get("src") != "about:blank"
                ]
                for j, img in enumerate(images):
                    img_id = self.generate_doc_id(img.get("src", f"img-{j}"))
                    img["id"] = img_id
                    img["page_url"] = doc.get("url")
                    img["page_title"] = doc.get("title", "")
                    img["page_description"] = doc.get("description", "")
                    img["timestamp"] = doc.get("timestamp")
                    unique_imgs[img_id] = img

            documents = list(unique_docs.values())
            images = list(unique_imgs.values())

            texts_to_embed = [self.build_embedding_text(doc) for doc in documents]
            embeddings = self.model.encode(
                texts_to_embed,
                show_progress_bar=len(documents) > 10,
                batch_size=min(25, len(documents)),
            )

            for i, doc in enumerate(documents):
                doc_id = doc["id"]
                doc_images = [
                    img
                    for img in doc.get("images", [])
                    if img.get("src") and img.get("src") != "about:blank"
                ]

                payload = {
                    "url": doc.get("url"),
                    "title": doc.get("title", ""),
                    "description": doc.get("description", ""),
                    "headings": doc.get("headings", [])[:3],
                    "images": doc_images[:3],
                    "language": doc.get("language", ""),
                    "timestamp": doc.get("timestamp"),
                    "content_type": doc.get("content_type"),
                    "text_snippet": doc.get("cleaned_text", "")[:500],
                }

                points.append(
                    PointStruct(
                        id=doc_id,
                        vector=embeddings[i],
                        payload=payload,
                    )
                )

                language = doc.get("language", "simple")
                if language in {"chinese", "japanese", None, ""}:
                    language = "simple"
                supabase_rows.append(
                    {
                        "id": doc_id,
                        "url": doc.get("url"),
                        "title": doc.get("title"),
                        "lang": language,
                        "_tmp_content": doc.get("cleaned_text", ""),
                    }
                )

            valid_images = []
            captions = []
            for img in images:
                caption = self.build_image_caption(img)
                if caption:
                    valid_images.append(img)
                    captions.append(caption)

            img_embeddings = (
                self.model.encode(captions, show_progress_bar=False) if captions else []
            )

            for img, embedding, caption in zip(valid_images, img_embeddings, captions):
                img_id = img["id"]
                img_payload = {
                    "src": img.get("src"),
                    "alt": img.get("alt", ""),
                    "title": img.get("title", ""),
                    "caption": caption,
                    "page_url": img.get("page_url"),
                    "page_title": img.get("page_title", ""),
                    "page_description": img.get("page_description", ""),
                    "timestamp": img.get("timestamp"),
                }

                image_points.append(
                    PointStruct(
                        id=img_id,
                        vector=embedding,
                        payload=img_payload,
                    )
                )

            stats.embedding_time = time.time() - embedding_start

            self._upsert_qdrant_with_retry(self.collection_name, points, "documents")
            self._upsert_supabase_with_retry(supabase_rows, stats)

            batch_size = getattr(self.config, "batch_size", 100)
            for i in range(0, len(image_points), batch_size):
                batch = image_points[i : i + batch_size]
                self._upsert_qdrant_with_retry(
                    self.collection_name_images, batch, "images"
                )

            stats.successful_docs += len(points)
        except Exception as e:
            print(e)

        stats.processing_time = time.time() - start_time

        # Log comprehensive statistics
        self._log_indexing_stats(stats)

        return stats

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

    def _upsert_supabase_with_retry(
        self, rows: List[dict], stats: IndexingStats
    ) -> None:
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
                    stats.successful_docs_supabase += len(rows)
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

    @staticmethod
    def generate_doc_id(source: str) -> str:
        """Generate a UUID5 for a given string."""
        return str(uuid.uuid5(uuid.NAMESPACE_URL, source))

    def __enter__(self):
        """Context manager entry"""
        return self
