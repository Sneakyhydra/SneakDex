import uuid
import logging

from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.http.models import PointStruct, VectorParams, Distance

from indexer.config import IndexerConfig

log = logging.getLogger("indexer")
logging.basicConfig(level=logging.INFO)


class ModernIndexer:
    def __init__(
        self,
        config: IndexerConfig,
        collection_name: str,
        model_name: str = "all-MiniLM-L6-v2",
    ):
        self.collection_name = collection_name
        self.model = SentenceTransformer(model_name)
        vector_size = self.model.get_sentence_embedding_dimension() or 384

        self.client = QdrantClient(
            url="https://eaec3da6-1c69-4c3b-bd7b-f61fde0e2761.eu-central-1-0.aws.cloud.qdrant.io:6333",
            api_key="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3MiOiJtIn0.x7amgrQLYPa5fg56TSBmWYtX1XkuS79wIoYC-tyedDU",
        )

        # Create collection if it doesn’t exist
        existing_collections = [
            c.name for c in self.client.get_collections().collections
        ]
        if collection_name not in existing_collections:
            log.info(f"Creating collection: {collection_name}")
            self.client.create_collection(
                collection_name=collection_name,
                vectors_config=VectorParams(size=vector_size, distance=Distance.COSINE),
            )
        else:
            log.info(f"Collection already exists: {collection_name}")

    def build_embedding_text(self, doc: dict) -> str:
        """Builds text for embedding: title + h1 + body snippet"""
        pieces = [doc.get("title", "")]
        h1 = next(
            (h["text"] for h in doc.get("headings", []) if h.get("level") == 1), ""
        )
        if h1:
            pieces.append(h1)
        pieces.append(doc.get("cleaned_text", "")[:1000])
        return " ".join(pieces).strip()

    def index_batch(self, documents: list[dict]):
        """
        Indexes a batch of parsed pages.

        Args:
            documents: list of dicts (ParsedPage models as JSON)
        """
        if not documents:
            return

        texts_to_embed = [self.build_embedding_text(doc) for doc in documents]
        embeddings = self.model.encode(texts_to_embed, show_progress_bar=False)

        points = []
        for i, doc in enumerate(documents):
            payload = {
                "url": doc.get("url"),
                "title": doc.get("title"),
                "description": doc.get("description"),
                "headings": doc.get("headings"),
                "links": doc.get("links"),
                "images": doc.get("images"),
                "canonical_url": doc.get("canonical_url"),
                "language": doc.get("language"),
                "word_count": doc.get("word_count"),
                "timestamp": doc.get("timestamp"),
                "meta_keywords": doc.get("meta_keywords"),
                "content_type": doc.get("content_type"),
                "encoding": doc.get("encoding"),
                "text_snippet": doc.get("cleaned_text", "")[:500],
            }

            points.append(
                PointStruct(
                    id=str(uuid.uuid4()),
                    vector=embeddings[i],
                    payload=payload,
                )
            )

        self.client.upsert(collection_name=self.collection_name, points=points)
        log.info(
            f"Indexed {len(points)} documents into collection '{self.collection_name}'."
        )

    def count(self) -> int:
        """Returns the number of vectors in the collection."""
        stats = self.client.get_collection(self.collection_name)
        return stats.vectors_count or 0
