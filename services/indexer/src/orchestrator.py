"""
Orchestrates the ModernIndexer service:
- Initializes embedding model, Qdrant collections, and Supabase client.
- Provides batch indexing of documents and images into Qdrant & Supabase.
- Ensures idempotency and basic validation.
"""

import logging
import uuid

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

        for i, doc in enumerate(documents):
            doc_id = self.generate_doc_id(doc.get("url", f"doc-{i}"))
            images = [
                img
                for img in doc.get("images", [])
                if img.get("src") and img.get("src") != "about:blank"
            ]

            payload = {
                "url": doc.get("url"),
                "title": doc.get("title", ""),
                "description": doc.get("description", ""),
                "headings": doc.get("headings", [])[:3],
                "images": images[:3],
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

            supabase_rows.append(
                {
                    "id": doc_id,
                    "url": doc.get("url"),
                    "title": doc.get("title"),
                    "lang": doc.get("language", ""),
                    "_tmp_content": doc.get("cleaned_text", ""),
                }
            )

            imgs_to_embed = [self.build_image_caption(img) for img in images]
            img_embeddings = (
                self.model.encode(imgs_to_embed, show_progress_bar=False)
                if imgs_to_embed
                else []
            )

            for j, img in enumerate(images):
                if not imgs_to_embed[j]:
                    continue

                img_id = self.generate_doc_id(img.get("src", f"img-{i}-{j}"))
                img_payload = {
                    "src": img.get("src"),
                    "alt": img.get("alt", ""),
                    "title": img.get("title", ""),
                    "caption": imgs_to_embed[j],
                    "page_url": doc.get("url"),
                    "page_title": doc.get("title", ""),
                    "page_description": doc.get("description", ""),
                    "timestamp": doc.get("timestamp"),
                }

                image_points.append(
                    PointStruct(
                        id=img_id,
                        vector=img_embeddings[j],
                        payload=img_payload,
                    )
                )

        self._upsert_qdrant(self.collection_name, points, "documents")
        self._upsert_qdrant(self.collection_name_images, image_points, "images")
        self._upsert_supabase(supabase_rows)

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
