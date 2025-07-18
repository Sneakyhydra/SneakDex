import logging
import uuid

from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.http.models import PointStruct, VectorParams, Distance
from supabase import create_client, Client

from src.config import IndexerConfig

log = logging.getLogger("indexer")
logging.basicConfig(level=logging.INFO)


class ModernIndexer:
    def __init__(self, config: IndexerConfig, model_name: str = "all-MiniLM-L12-v2"):
        self.config = config
        self.collection_name = config.collection_name
        self.collection_name_images = config.collection_name_images

        # SentenceTransformer
        self.model = SentenceTransformer(model_name)
        vector_size = self.model.get_sentence_embedding_dimension() or 384

        # Qdrant
        self.client = QdrantClient(
            url=config.qdrant_url,
            api_key=config.qdrant_api_key,
        )

        existing_collections = [
            c.name for c in self.client.get_collections().collections
        ]
        if self.collection_name not in existing_collections:
            log.info(f"Creating Qdrant collection: {self.collection_name}")
            self.client.create_collection(
                collection_name=self.collection_name,
                vectors_config=VectorParams(size=vector_size, distance=Distance.COSINE),
            )
        else:
            log.info(f"Qdrant collection already exists: {self.collection_name}")

        if self.collection_name_images not in existing_collections:
            log.info(f"Creating Qdrant collection: {self.collection_name_images}")
            self.client.create_collection(
                collection_name=self.collection_name_images,
                vectors_config=VectorParams(size=vector_size, distance=Distance.COSINE),
            )
        else:
            log.info(f"Qdrant collection already exists: {self.collection_name_images}")

        # Supabase
        self.supabase: Client = create_client(
            config.supabase_url, config.supabase_api_key
        )

    def build_embedding_text(self, doc: dict) -> str:
        """Builds text for embedding: title + h1 + full body text"""
        pieces = [doc.get("title", "")]
        h1 = next(
            (h["text"] for h in doc.get("headings", []) if h.get("level") == 1), ""
        )
        if h1:
            pieces.append(h1)
        pieces.append(doc.get("cleaned_text", ""))
        return " ".join(pieces).strip()

    def build_image_caption(self, img: dict) -> str:
        """Builds text for embedding an image: alt + title"""
        return " ".join([img.get("alt", "") or "", img.get("title", "") or ""]).strip()

    def index_batch(self, documents: list[dict]):
        """
        Indexes a batch of parsed pages into Qdrant & Supabase.
        """
        if not documents:
            return

        texts_to_embed = [self.build_embedding_text(doc) for doc in documents]
        embeddings = self.model.encode(texts_to_embed, show_progress_bar=False)

        points = {}
        image_points = {}
        supabase_rows = {}

        for i, doc in enumerate(documents):
            doc_id = self.generate_doc_id(doc.get("url", ""))
            images = [
                image
                for image in doc.get("images", [])
                if image.get("src") and image.get("src") != "about:blank"
            ]

            language_pg = doc.get("language")
            if language_pg and language_pg == "chinese" and language_pg == "japanese":
                language_pg = "simple"

            payload = {
                "url": doc.get("url"),
                "title": doc.get("title", ""),
                "description": doc.get("description", ""),
                "headings": doc.get("headings", [])[:3],
                "images": images[:3],
                "language": doc.get("language"),
                "timestamp": doc.get("timestamp"),
                "content_type": doc.get("content_type"),
                "text_snippet": doc.get("cleaned_text", "")[:500],
            }

            points[doc_id] = PointStruct(
                id=doc_id,
                vector=embeddings[i],
                payload=payload,
            )

            supabase_rows[doc_id] = {
                "id": doc_id,
                "url": doc.get("url"),
                "title": doc.get("title"),
                "lang": language_pg,
                "_tmp_content": doc.get("cleaned_text", ""),
            }

            imgs_to_embed = [self.build_image_caption(img) for img in images]
            if imgs_to_embed:
                img_embeddings = self.model.encode(
                    imgs_to_embed, show_progress_bar=False
                )
            else:
                img_embeddings = []

            # index images
            for j, img in enumerate(images):
                if not imgs_to_embed[j]:
                    continue  # skip empty captions

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

                img_id = self.generate_doc_id(img.get("src", ""))
                image_points[img_id] = PointStruct(
                    id=img_id,
                    vector=img_embeddings[j],
                    payload=img_payload,
                )

        # Convert dicts to lists
        points = list(points.values())
        image_points = list(image_points.values())
        supabase_rows = list(supabase_rows.values())

        # Qdrant
        try:
            if points:
                self.client.upsert(collection_name=self.collection_name, points=points)
                log.info(
                    f"Indexed {len(points)} documents into Qdrant collection '{self.collection_name}'."
                )
        except Exception as e:
            log.error(f"Failed to upsert points to Qdrant: {e}")

        try:
            if image_points:
                self.client.upsert(
                    collection_name=self.collection_name_images, points=image_points
                )
                log.info(
                    f"Indexed {len(image_points)} images into Qdrant collection '{self.collection_name_images}'."
                )
        except Exception as e:
            log.error(f"Failed to upsert image points to Qdrant: {e}")

        # Supabase
        try:
            if supabase_rows:
                response = (
                    self.supabase.table("documents").upsert(supabase_rows).execute()
                )
                if not response.data:
                    log.error(
                        f"Supabase insert failed. Response: {response.model_dump_json()}"
                    )
                else:
                    log.info(f"Inserted {len(response.data)} rows into Supabase.")
        except Exception as e:
            log.error(f"Failed to upsert rows to Supabase: {e}")

    def count(self) -> int:
        """Returns the number of vectors in the Qdrant collection."""
        stats = self.client.get_collection(self.collection_name)
        return stats.vectors_count or 0

    def generate_doc_id(self, url: str) -> str:
        return str(uuid.uuid5(uuid.NAMESPACE_URL, url))
