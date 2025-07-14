import { NextResponse } from "next/server";
import { QdrantClient } from "@qdrant/js-client-rest";
import { pipeline } from "@xenova/transformers";
import { Redis } from "@upstash/redis";

// === CONFIG ===
const QDRANT_URL = process.env.QDRANT_URL!;
const QDRANT_API_KEY = process.env.QDRANT_API_KEY!;
const COLLECTION_NAME_IMAGES =
  process.env.COLLECTION_NAME_IMAGES || "sneakdex-images";

// === Qdrant Client ===
const qdrant = new QdrantClient({ url: QDRANT_URL, apiKey: QDRANT_API_KEY });

let embedder: any = null;

// === Embedder ===
async function getEmbedder() {
  if (!embedder) {
    embedder = await pipeline("feature-extraction", "Xenova/all-MiniLM-L6-v2");
  }
  return embedder;
}

async function computeEmbedding(query: string): Promise<number[]> {
  const model = await getEmbedder();
  const output = await model(query, { pooling: "mean", normalize: true });
  return Array.from(output.data);
}

// === POST Handler ===
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const query: string = body.query?.trim();
    const top_k: number = 100;

    if (!query) {
      return NextResponse.json({ error: "Missing query" }, { status: 400 });
    }

    const redis = Redis.fromEnv();

    const cacheKey = `search-images:${query}:${top_k}`;
    const cachedResult = await redis.get(cacheKey);

    if (cachedResult) {
      return NextResponse.json({ source: "cache", results: cachedResult });
    }

    const vector = await computeEmbedding(query);

    const hits = await qdrant.search(COLLECTION_NAME_IMAGES, {
      vector,
      limit: top_k,
      with_payload: true,
    });

    const results = hits.map((hit, idx) => ({
      id: String(hit.id),
      score: hit.score ?? 0,
      payload: hit.payload,
      rank: idx + 1,
    }));

    // Cache the results
    await redis.set(cacheKey, results, { ex: 60 * 60 });

    return NextResponse.json({ source: "qdrant-images", results });
  } catch (err) {
    console.error(err);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
