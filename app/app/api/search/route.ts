import { NextResponse } from "next/server";
import { QdrantClient } from "@qdrant/js-client-rest";
import Redis from "ioredis";
import { pipeline } from "@xenova/transformers";

// Initialize Qdrant
const qdrant = new QdrantClient({
  url: process.env.QDRANT_URL,
  apiKey: process.env.QDRANT_API_KEY,
});

// Initialize Redis
// const redis = new Redis("redis:6379");

let embedder: any = null;

// Load SentenceTransformer model
async function getEmbedder() {
  if (embedder) return embedder;

  embedder = await pipeline(
    "feature-extraction",
    "Xenova/all-MiniLM-L6-v2"
  );
  return embedder;
}

// Compute embedding from text
async function computeEmbedding(text: string): Promise<number[]> {
  const model = await getEmbedder();
  const output = await model(text, { pooling: "mean", normalize: true });
  return Array.from(output.data);
}

// POST handler
export async function POST(req: Request) {
  try {
    const { query, top_k = 10 } = await req.json();

    if (!query || typeof query !== "string") {
      return NextResponse.json({ error: "Invalid query" }, { status: 400 });
    }

    // Check Redis cache
    // const cacheKey = `query:${query}`;
    // const cached = await redis.get(cacheKey);
    // if (cached) {
    //   return NextResponse.json({
    //     source: "cache",
    //     results: JSON.parse(cached),
    //   });
    // }

    // Compute embedding
    const vector = await computeEmbedding(query);

    // Query Qdrant
    const searchResult = await qdrant.search("sneakdex", {
      vector,
      limit: top_k,
      with_payload: true,
    });

    const results = searchResult.map((hit) => ({
      id: hit.id,
      score: hit.score,
      payload: hit.payload,
    }));

    // Save in Redis
    // await redis.setex(cacheKey, 300, JSON.stringify(results)); // TTL 5min

    return NextResponse.json({
      source: "qdrant",
      results,
    });
  } catch (err) {
    console.error(err);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
