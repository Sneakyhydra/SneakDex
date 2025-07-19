"""
Orchestrates the ModernIndexer service:
- Initializes embedding model, Qdrant collections, and Supabase client.
- Provides batch indexing of documents and images into Qdrant & Supabase.
- Ensures idempotency and basic validation.
"""

import logging
import math
import uuid
import re
from urllib.parse import urlparse

from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.http.models import PointStruct, VectorParams, Distance
from supabase import create_client, Client

from src.config import IndexerConfig

log = logging.getLogger("indexer")


class ModernIndexer:
    def __init__(self, config: IndexerConfig, model_name: str = "all-MiniLM-L12-v2"):
        """
        Initialize embedding model, Qdrant collections, and Supabase client.
        """
        self.config = config
        self.collection_name = config.collection_name
        self.collection_name_images = config.collection_name_images

        # SentenceTransformer
        log.info(f"Loading embedding model: {model_name}")
        self.model = SentenceTransformer(model_name)
        vector_size = self.model.get_sentence_embedding_dimension() or 384
        log.info(f"Embedding dimension: {vector_size}")

        # Qdrant
        self.client = QdrantClient(url=config.qdrant_url, api_key=config.qdrant_api_key)

        self._ensure_qdrant_collections(vector_size)

        # Supabase
        self.supabase: Client = create_client(
            config.supabase_url, config.supabase_api_key
        )
        log.info("Connected to Supabase")

    def _ensure_qdrant_collections(self, vector_size: int) -> None:
        """
        Create collections in Qdrant if they don't already exist.
        """
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
                    )
                else:
                    log.info(f"Qdrant collection already exists: {cname}")
        except Exception as e:
            log.error(f"Failed to ensure Qdrant collections: {e}")
            raise

    def normalize_url(self, url: str) -> str:
        """
        Turn a URL into a human-readable string for embedding.
        Example: 'https://en.wikipedia.org/wiki/OpenAI' → 'en wikipedia org wiki OpenAI'
        """
        if not url:
            return ""
        parsed = urlparse(url)
        # break into domain and path
        domain = parsed.netloc.replace(".", " ")
        path = re.sub(r"[^a-zA-Z0-9]", " ", parsed.path)  # replace slashes and symbols
        return f"{domain} {path}".strip()

    def build_embedding_text(self, doc: dict) -> str:
        """
        Build text for embedding: title + first h1 + body text.
        """
        pieces = [doc.get("title", "")]
        h1 = next(
            (h["text"] for h in doc.get("headings", []) if h.get("level") == 1), ""
        )
        if h1:
            pieces.append(h1)

        pieces.append(doc.get("cleaned_text", ""))
        url = doc.get("url", "").strip()
        if url:
            pieces.append(self.normalize_url(url))
        return " ".join(pieces).strip()

    def build_image_caption(self, img: dict) -> str:
        """
        Build text for embedding an image: alt + title.
        """
        return " ".join(
            filter(None, [img.get("alt", ""), img.get("title", "")])
        ).strip()

    def index_batch(self, documents: list[dict]) -> None:
        """
        Index a batch of parsed pages & images into Qdrant & Supabase.
        """
        if not documents:
            log.warning("Received empty batch. Skipping indexing.")
            return

        texts_to_embed = [self.build_embedding_text(doc) for doc in documents]
        embeddings = self.model.encode(texts_to_embed, show_progress_bar=False)

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

        for i, doc in enumerate(documents):
            doc_id = doc.get("id", self.generate_doc_id(doc.get("url", f"doc-{i}")))
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
            img_id = img.get("id", self.generate_doc_id(img.get("src")))
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

        self._upsert_qdrant(self.collection_name, points, "documents")
        self._upsert_supabase(supabase_rows)

        batch_size = getattr(self.config, "batch_size", 100)
        for i in range(0, len(image_points), batch_size):
            batch = image_points[i : i + batch_size]
            self._upsert_qdrant(self.collection_name_images, batch, "images")

    def _upsert_qdrant(self, collection: str, points: list, label: str) -> None:
        """
        Upsert points into Qdrant.
        """
        if not points:
            log.info(f"No {label} to index into Qdrant.")
            return
        try:
            self.client.upsert(collection_name=collection, points=points)
            log.info(
                f"✅ Indexed {len(points)} {label} into Qdrant collection '{collection}'."
            )
        except Exception as e:
            log.error(f"❌ Failed to upsert {label} to Qdrant: {e}")

    def _upsert_supabase(self, rows: list) -> None:
        """
        Upsert rows into Supabase.
        """
        if not rows:
            log.info("No rows to insert into Supabase.")
            return
        try:
            response = self.supabase.table("documents").upsert(rows).execute()
            if not getattr(response, "data", None):
                log.error(f"Supabase insert returned no data. Response: {response}")
            else:
                log.info(f"✅ Inserted {len(response.data)} rows into Supabase.")
        except Exception as e:
            log.error(f"❌ Failed to upsert rows to Supabase: {e}")

    def count(self) -> int:
        """
        Returns the number of vectors in the main Qdrant collection.
        """
        try:
            stats = self.client.get_collection(self.collection_name)
            return stats.vectors_count or 0
        except Exception as e:
            log.error(f"Failed to get vector count from Qdrant: {e}")
            return -1

    @staticmethod
    def generate_doc_id(source: str) -> str:
        """
        Generate a UUID5 for a given string.
        """
        return str(uuid.uuid5(uuid.NAMESPACE_URL, source))
