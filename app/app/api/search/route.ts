import { NextResponse } from "next/server";
import { QdrantClient } from "@qdrant/js-client-rest";
import { pipeline } from "@xenova/transformers";
import { createClient } from "@supabase/supabase-js";

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
    const top_k: number = body.top_k ?? 10;

    if (!query) {
      return NextResponse.json({ error: "Missing query" }, { status: 400 });
    }

    // === QDRANT SEARCH ===
    const qdrantPromise = (async () => {
      const vector = await computeEmbedding(query);
      const hits = await qdrant.search(COLLECTION_NAME, {
        vector,
        limit: top_k,
        with_payload: true,
      });

      return hits.map((hit, idx) => ({
        id: String(hit.id),
        qdrantScore: hit.score ?? 0,
        pgScore: 0,
        payload: hit.payload,
        rank_qdrant: idx + 1,
      }));
    })();

    // === SUPABASE SEARCH ===
    const pgPromise = (async () => {
      const { data, error } = await supabase
        .from("documents")
        .select("id, title, url, ts_rank(content_tsv, plainto_tsquery($$query$$)) AS score", {
          count: "exact",
        })
        .textSearch("content_tsv", query, {
          type: "plain",
          config: "english",
        })
        .order("score", { ascending: false })
        .limit(top_k);

      if (error) {
        console.error("Supabase error:", error);
        return [];
      }

      return data.map((row: any, idx: number) => ({
        id: String(row.id),
        pgScore: row.score ?? 0,
        qdrantScore: 0,
        payload: row,
        rank_pg: idx + 1,
      }));
    })();

    const [qdrantResults, pgResults] = await Promise.all([qdrantPromise, pgPromise]);

    // === MERGE RESULTS ===
    const merged = new Map<string, any>();

    for (const r of qdrantResults) {
      merged.set(r.id, { ...r });
    }

    for (const r of pgResults) {
      if (merged.has(r.id)) {
        const m = merged.get(r.id);
        merged.set(r.id, {
          ...m,
          pgScore: r.pgScore,
          rank_pg: r.rank_pg,
        });
      } else {
        merged.set(r.id, { ...r });
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

    return NextResponse.json({ source: "hybrid", results: finalResults });
  } catch (err) {
    console.error(err);
    return NextResponse.json({ error: "Internal server error" }, { status: 500 });
  }
}
