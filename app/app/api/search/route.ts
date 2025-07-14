import { NextResponse } from "next/server";
import { QdrantClient } from "@qdrant/js-client-rest";
import { pipeline } from "@xenova/transformers";
import { createClient } from "@supabase/supabase-js";
import { Redis } from "@upstash/redis";

// === CONFIG ===
const QDRANT_URL = process.env.QDRANT_URL!;
const QDRANT_API_KEY = process.env.QDRANT_API_KEY!;
const SUPABASE_URL = process.env.SUPABASE_URL!;
const SUPABASE_API_KEY = process.env.SUPABASE_API_KEY!;
const COLLECTION_NAME = process.env.QDRANT_COLLECTION || "sneakdex";

// === CLIENTS ===
const qdrant = new QdrantClient({ url: QDRANT_URL, apiKey: QDRANT_API_KEY });
const supabase = createClient(SUPABASE_URL, SUPABASE_API_KEY);

let embedder: any = null;

// === EMBEDDER ===
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

// === HANDLER ===
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const query: string = body.query?.trim();
    const top_k: number = 100;

    if (!query) {
      return NextResponse.json({ error: "Missing query" }, { status: 400 });
    }

    const redis = Redis.fromEnv();

    const cacheKey = `search:${query}:${top_k}`;
    const cachedResult = await redis.get(cacheKey);

    if (cachedResult) {
      return NextResponse.json({ source: "cache", results: cachedResult });
    }

    // === QDRANT SEARCH ===
    const qdrantPromise = (async () => {
      const vector = await computeEmbedding(query);
      const hits = await qdrant.search(COLLECTION_NAME, {
        vector,
        limit: top_k,
        with_payload: true,
      });

      return hits.map((hit) => ({
        id: String(hit.id),
        qdrantScore: hit.score ?? 0,
        pgScore: 0,
        payload: hit.payload,
      }));
    })();

    // === SUPABASE SEARCH ===
    const pgPromise = (async () => {
      const { data, error } = await supabase.rpc("search_documents", {
        q: query,
        limit_count: top_k,
      });

      if (error) {
        console.error("Supabase error:", error);
        return [];
      }

      return data.map((row: any) => ({
        id: String(row.id),
        pgScore: row.score ?? 0,
        url: row.url,
        title: row.title,
      }));
    })();

    const [qdrantResults, pgResults] = await Promise.all([
      qdrantPromise,
      pgPromise,
    ]);

    // === MERGE RESULTS ===
    const merged = new Map<string, any>();

    // Start with Qdrant
    for (const r of qdrantResults) {
      merged.set(r.id, {
        id: r.id,
        qdrantScore: r.qdrantScore,
        pgScore: 0,
        payload: r.payload,
      });
    }

    // Add Supabase info
    for (const r of pgResults) {
      if (merged.has(r.id)) {
        const m = merged.get(r.id);
        merged.set(r.id, {
          ...m,
          pgScore: r.pgScore,
        });
      } else {
        // no Qdrant — but at least we have URL & title
        merged.set(r.id, {
          id: r.id,
          qdrantScore: 0,
          pgScore: r.pgScore,
          payload: {
            url: r.url,
            title: r.title,
          },
        });
      }
    }

    const weights = { qdrant: 0.7, pg: 0.3 };

    const finalResults = Array.from(merged.values())
      .map((r) => ({
        ...r,
        hybridScore: weights.qdrant * r.qdrantScore + weights.pg * r.pgScore,
      }))
      .sort((a, b) => b.hybridScore - a.hybridScore)
      .slice(0, top_k);

    // Cache the results
    await redis.set(cacheKey, finalResults, { ex: 60 * 60 });

    return NextResponse.json({ source: "hybrid", results: finalResults });
  } catch (err) {
    console.error(err);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
