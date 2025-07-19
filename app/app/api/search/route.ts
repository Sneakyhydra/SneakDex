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
const COLLECTION_NAME = process.env.QDRANT_COLLECTION_NAME || "sneakdex";
const HF_API_KEY = process.env.HUGGINGFACE_API_KEY; // Optional fallback

if (!QDRANT_URL || !QDRANT_API_KEY) {
  throw new Error("Missing QDRANT_URL or QDRANT_API_KEY in environment");
}
if (!SUPABASE_URL || !SUPABASE_API_KEY) {
  throw new Error("Missing SUPABASE_URL or SUPABASE_API_KEY in environment");
}
if (!COLLECTION_NAME) {
  throw new Error("Missing QDRANT_COLLECTION_NAME in environment");
}

// === VERCEL OPTIMIZATIONS ===
export const config = {
  runtime: "nodejs",
  maxDuration: 30, // Increase timeout for model loading
};

// === CLIENTS ===
const qdrant = new QdrantClient({ url: QDRANT_URL, apiKey: QDRANT_API_KEY });
const supabase = createClient(SUPABASE_URL, SUPABASE_API_KEY);

// === OPTIMIZED EMBEDDING SYSTEM ===
let modelPromise: Promise<any> | null = null;
let lastUsed = Date.now();
const embedCache = new Map<string, number[]>();
const CACHE_CLEANUP_INTERVAL = 10 * 60 * 1000; // 10 minutes
const MODEL_CLEANUP_INTERVAL = 30 * 60 * 1000; // 30 minutes

type Heading = { level: number; text: string };
type Image = { src: string; alt?: string; title?: string };

type QdrantPayload = {
  url?: string;
  title?: string;
  description?: string;
  headings?: Heading[];
  images?: Image[];
  language?: string;
  timestamp?: string;
  content_type?: string;
  text_snippet?: string;
  [key: string]: any;
};

type QdrantResult = {
  id: string;
  qdrantScore: number;
  pgScore: number;
  payload: QdrantPayload | null;
};

type PgResponse = {
  id: string;
  rank: number;
  url?: string;
  title?: string;
};

type PgResult = {
  id: string;
  pgScore: number;
  url?: string;
  title?: string;
};

type HybridResult = {
  id: string;
  hybridScore: number;
  qdrantScore: number;
  pgScore: number;
  payload: QdrantPayload | null;
  url?: string;
  title?: string;
};

type SearchRequest = {
  query: string;
  top_k?: number;
  weights?: { qdrant: number; pg: number };
  useEmbeddings?: boolean;
  filters?: Record<string, any>;
};

type FieldSchema = "keyword" | "integer" | "float" | "geo" | "boolean" | "text";

async function ensurePayloadIndexes(
  qdrant: any, // the Qdrant client instance
  collection: string,
  fields: { name: string; schema: FieldSchema }[]
) {
  for (const { name, schema } of fields) {
    try {
      console.log(
        `➡️ Checking/creating index for field: "${name}" as ${schema}`
      );

      await qdrant.createPayloadIndex(collection, {
        field_name: name,
        field_schema: schema,
      });

      console.log(`✅ Index ensured for field: "${name}"`);
    } catch (err: any) {
      if (
        err?.response?.data?.status?.error?.includes("already exists") ||
        err?.message?.includes("already exists")
      ) {
        console.log(`ℹ️ Index already exists for field: "${name}"`);
      } else {
        console.error(`❌ Failed to create index for field: "${name}"`, err);
      }
    }
  }
}

// === OPTIMIZED EMBEDDER ===
async function getEmbedder() {
  if (!modelPromise) {
    console.log("Loading embedding model...");
    modelPromise = pipeline("feature-extraction", "Xenova/all-MiniLM-L12-v2");
  }
  lastUsed = Date.now();
  return await modelPromise;
}

// Cleanup model from memory after extended idle periods
setInterval(() => {
  if (Date.now() - lastUsed > MODEL_CLEANUP_INTERVAL) {
    console.log("Cleaning up model from memory");
    modelPromise = null;
  }
}, MODEL_CLEANUP_INTERVAL);

// Cleanup embedding cache periodically
setInterval(() => {
  if (embedCache.size > 100) {
    console.log("Cleaning up embedding cache");
    embedCache.clear();
  }
}, CACHE_CLEANUP_INTERVAL);

async function getCachedEmbedding(query: string): Promise<number[] | null> {
  // Check in-memory cache first
  if (embedCache.has(query)) {
    return embedCache.get(query)!;
  }

  try {
    const embedding = await computeEmbedding(query);

    // Cache the result
    embedCache.set(query, embedding);

    // Limit cache size
    if (embedCache.size > 200) {
      const firstKey = embedCache.keys().next().value;
      if (firstKey) {
        embedCache.delete(firstKey);
      }
    }

    return embedding;
  } catch (error) {
    console.error("Embedding generation failed:", error);
    return null; // Return null instead of throwing
  }
}

async function computeEmbedding(query: string): Promise<number[]> {
  try {
    // Primary: Use Transformers.js
    console.log("Using Transformers.js for embedding");
    const model = await getEmbedder();
    const output = await model(query, { pooling: "mean", normalize: true });
    return Array.from(output.data);
  } catch (error) {
    console.error("Transformers.js failed:", error);

    // Fallback: Use Hugging Face API if available
    if (HF_API_KEY) {
      try {
        console.log("Using Hugging Face API for embedding");
        return await computeEmbeddingWithHF(query);
      } catch (hfError) {
        console.error("Hugging Face API also failed:", hfError);
        throw hfError;
      }
    }

    console.error("No embedding method succeeded");
    throw error; // This will be caught by getCachedEmbedding
  }
}

async function computeEmbeddingWithHF(query: string): Promise<number[]> {
  const response = await fetch(
    "https://router.huggingface.co/hf-inference/models/intfloat/multilingual-e5-large/pipeline/feature-extraction",
    {
      headers: {
        Authorization: `Bearer ${HF_API_KEY}`,
        "Content-Type": "application/json",
      },
      method: "POST",
      body: JSON.stringify({ inputs: query }),
    }
  );

  if (!response.ok) {
    const data = await response.json();
    console.error("Hugging Face API error:", data.error);
    throw new Error(`HF API failed: ${data.error}`);
  }

  console.log("Hugging Face API embedding successful");
  const data = await response.json();
  return data;
}

// === PAYLOAD-ONLY SEARCH ===
async function searchPayloadOnly(
  query: string,
  top_k: number,
  filters?: Record<string, any>
): Promise<QdrantResult[]> {
  try {
    await ensurePayloadIndexes(qdrant, COLLECTION_NAME, [
      { name: "title", schema: "keyword" },
      { name: "description", schema: "keyword" },
      { name: "text_snippet", schema: "keyword" },
    ]);

    const textConditions: any[] = [
      { key: "title", match: { value: query } },
      { key: "description", match: { value: query } },
      { key: "text_snippet", match: { value: query } },
    ];

    const additionalConditions: any[] = [];

    if (filters) {
      for (const [key, value] of Object.entries(filters)) {
        if (typeof value === "string") {
          additionalConditions.push({ key, match: { value: value } });
        } else {
          additionalConditions.push({ key, match: { value } });
        }
      }
    }

    const filterPayload = {
      min_should: {
        conditions: [...textConditions, ...additionalConditions],
        min_count: 1,
      },
    };

    const requestPayload = {
      filter: filterPayload,
      limit: top_k,
      with_payload: true,
      with_vector: false,
    };

    console.log(`\n=== QDRANT PAYLOAD SEARCH ===`);
    // console.log(`Query:`, query);
    // console.log(`Top_k:`, top_k);
    // console.log(`Filters:`, JSON.stringify(filters, null, 2));
    // console.log(`Filter conditions:`, JSON.stringify(filterPayload, null, 2));
    // console.log(`Full request:`, JSON.stringify(requestPayload, null, 2));

    const results = await qdrant.scroll(COLLECTION_NAME, requestPayload);
    // const response = await fetch(
    //   `${QDRANT_URL}/collections/${COLLECTION_NAME}/points/scroll`,
    //   {
    //     method: "POST",
    //     headers: {
    //       "Content-Type": "application/json",
    //       "api-Key": QDRANT_API_KEY,
    //     },
    //     body: JSON.stringify(requestPayload),
    //   }
    // );

    // const results = await response.json();

    console.log(`Qdrant response length:`, results.points?.length ?? 0);

    return (
      results.points?.map((point: any) => ({
        id: String(point.id),
        qdrantScore: 1.0,
        pgScore: 0,
        payload: point.payload ?? null,
      })) || []
    );
  } catch (error) {
    console.error(`❌ Payload search failed:`, error);
    return [];
  }
}

// === ENHANCED CACHE STRATEGY ===
function generateCacheKey(
  query: string,
  top_k: number,
  useEmbeddings?: boolean,
  filters?: Record<string, any>
): string {
  const baseKey = Buffer.from(query.toLowerCase().trim()).toString("base64");
  const filtersKey = filters
    ? Buffer.from(JSON.stringify(filters)).toString("base64")
    : "none";
  const embeddingsKey = useEmbeddings ? "vec" : "payload";

  return `search:${baseKey}:${top_k}:${embeddingsKey}:${filtersKey}`;
}

// === MAIN HANDLER ===
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const {
      query,
      top_k = 50, // Reduced default for better performance
      useEmbeddings = true,
      filters,
    }: SearchRequest = body;

    // Enhanced validation
    if (!query || typeof query !== "string") {
      return NextResponse.json(
        { error: "Missing or invalid query" },
        { status: 400 }
      );
    }

    const cleanQuery = query.trim();

    if (cleanQuery.length < 1 || cleanQuery.length > 100) {
      return NextResponse.json(
        { error: "Query must be between 1 and 100 characters" },
        { status: 400 }
      );
    }

    if (top_k < 1 || top_k > 200 || isNaN(top_k)) {
      return NextResponse.json(
        { error: "top_k must be between 1 and 200" },
        { status: 400 }
      );
    }

    const redis = Redis.fromEnv();
    const cacheKey = generateCacheKey(
      cleanQuery,
      top_k,
      useEmbeddings,
      filters
    );

    // Check cache first
    try {
      const cachedResult = await redis.get(cacheKey);
      if (cachedResult) {
        return NextResponse.json({
          source: "cache",
          results: cachedResult,
          query: cleanQuery,
          top_k,
        });
      }
    } catch (cacheError) {
      console.error("Cache read failed:", cacheError);
      // Continue without cache
    }

    // Determine weights based on query characteristics
    const finalWeights = {
      qdrant: 0.75,
      pg: 0.25,
    };

    let qdrantResults: QdrantResult[] = [];
    let pgResults: PgResult[] = [];
    let payloadSeachedAlready = false;

    // === QDRANT SEARCH ===
    const qdrantPromise = (async (): Promise<QdrantResult[]> => {
      try {
        if (!useEmbeddings) {
          payloadSeachedAlready = true;
          return await searchPayloadOnly(cleanQuery, top_k, filters);
        }

        const vector = await getCachedEmbedding(cleanQuery);

        // If embedding generation failed, fallback to payload search
        if (!vector) {
          console.log("Embedding failed, falling back to payload search");
          payloadSeachedAlready = true;
          return await searchPayloadOnly(cleanQuery, top_k, filters);
        }

        const searchParams: any = {
          vector,
          limit: top_k,
          with_payload: true,
          with_vector: false,
        };

        // Add filters if provided
        if (filters) {
          searchParams.filter = {
            must: Object.entries(filters).map(([key, value]) => ({
              key,
              match: { value },
            })),
          };
        }

        const hits = await qdrant.search(COLLECTION_NAME, searchParams);

        return hits.map(
          (hit): QdrantResult => ({
            id: String(hit.id),
            qdrantScore: hit.score ?? 0,
            pgScore: 0,
            payload: hit.payload ?? null,
          })
        );
      } catch (error) {
        console.error("Qdrant search failed:", error);

        try {
          if (!payloadSeachedAlready) {
            // Fallback to payload search
            console.log("Falling back to payload search");
            return await searchPayloadOnly(cleanQuery, top_k, filters);
          } else {
            console.error("Payload search also failed, returning empty array");
            return []; // Return empty array instead of crashing
          }
        } catch (fallbackError) {
          console.error("Payload search also failed:", fallbackError);
          return []; // Return empty array instead of crashing
        }
      }
    })();

    // === SUPABASE SEARCH ===
    const pgPromise = (async (): Promise<PgResult[]> => {
      try {
        const { data, error } = await supabase.rpc("search_documents", {
          q: cleanQuery,
          limit_count: top_k,
        });

        if (error) {
          console.error("Supabase error:", error);
          return [];
        }

        return (data ?? []).map(
          (row: PgResponse): PgResult => ({
            id: String(row.id),
            pgScore: row.rank ?? 0,
            url: row.url,
            title: row.title,
          })
        );
      } catch (error) {
        console.error("Supabase search failed:", error);
        return [];
      }
    })();

    // Execute searches in parallel
    [qdrantResults, pgResults] = await Promise.all([qdrantPromise, pgPromise]);

    // Handle empty results
    if (qdrantResults.length === 0 && pgResults.length === 0) {
      return NextResponse.json({
        source: "empty",
        results: [],
        query: cleanQuery,
        top_k,
      });
    }

    let finalResults: HybridResult[];
    let source: string;

    if (qdrantResults.length > 0 && pgResults.length === 0) {
      // Only Qdrant results
      finalResults = qdrantResults
        .map((r) => ({
          ...r,
          hybridScore: r.qdrantScore,
        }))
        .sort((a, b) => b.hybridScore - a.hybridScore)
        .slice(0, top_k);
      source = payloadSeachedAlready ? "Qdrant payload" : "Qdrant vector";
    } else if (pgResults.length > 0 && qdrantResults.length === 0) {
      // Only Supabase results - enrich with Qdrant payloads
      const ids = pgResults.map((r) => r.id);
      let enrichedPgResults: HybridResult[];

      try {
        const points = await qdrant.retrieve(COLLECTION_NAME, {
          ids,
          with_payload: true,
        });

        const payloadMap = new Map(
          points.map((point) => [String(point.id), point.payload])
        );

        enrichedPgResults = pgResults.map((r) => ({
          ...r,
          hybridScore: r.pgScore,
          qdrantScore: 0,
          payload: payloadMap.get(r.id) || null,
        }));
      } catch (err) {
        console.error("Failed to fetch Qdrant payloads:", err);
        enrichedPgResults = pgResults.map((r) => ({
          ...r,
          hybridScore: r.pgScore,
          qdrantScore: 0,
          payload: null,
        }));
      }

      finalResults = enrichedPgResults.sort(
        (a, b) => b.hybridScore - a.hybridScore
      );
      source = "Supabase";
    } else {
      // === HYBRID MERGE ===
      const merged = new Map<string, HybridResult>();

      // Add Qdrant results
      for (const r of qdrantResults) {
        merged.set(r.id, {
          id: r.id,
          qdrantScore: r.qdrantScore,
          pgScore: 0,
          payload: r.payload,
          hybridScore: 0,
        });
      }

      // Merge with PostgreSQL results
      for (const r of pgResults) {
        if (merged.has(r.id)) {
          const existing = merged.get(r.id)!;
          merged.set(r.id, {
            ...existing,
            pgScore: r.pgScore,
            url: r.url,
            title: r.title,
          });
        } else {
          merged.set(r.id, {
            id: r.id,
            qdrantScore: 0,
            pgScore: r.pgScore,
            payload: null,
            hybridScore: 0,
            url: r.url,
            title: r.title,
          });
        }
      }

      // Calculate hybrid scores
      finalResults = Array.from(merged.values())
        .map((r) => ({
          ...r,
          hybridScore:
            finalWeights.qdrant * r.qdrantScore + finalWeights.pg * r.pgScore,
        }))
        .sort((a, b) => b.hybridScore - a.hybridScore)
        .slice(0, top_k);

      // Fetch missing payloads
      const idsToFetch = finalResults
        .filter((r) => !r.payload)
        .map((r) => r.id);
      if (idsToFetch.length > 0) {
        try {
          const points = await qdrant.retrieve(COLLECTION_NAME, {
            ids: idsToFetch,
            with_payload: true,
          });
          const payloadMap = new Map(
            points.map((point) => [String(point.id), point.payload])
          );
          for (const r of finalResults) {
            if (!r.payload && payloadMap.has(r.id)) {
              r.payload = payloadMap.get(r.id) ?? null;
            }
          }
        } catch (err) {
          console.error("Failed to fetch missing payloads:", err);
        }
      }

      source = payloadSeachedAlready
        ? "Qdrant payload + Supabase"
        : "Qdrant vector + Supabase";
    }

    // Cache the results
    try {
      await redis.set(cacheKey, finalResults, {
        ex: 60 * 60 * 2, // 2 hours cache
      });
    } catch (cacheError) {
      console.error("Cache write failed:", cacheError);
      // Continue without caching
    }

    return NextResponse.json({
      source,
      results: finalResults,
      query: cleanQuery,
      top_k,
      useEmbeddings,
    });
  } catch (err) {
    console.error("Search endpoint error:", err);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
