"use client";
import { useState, useRef, useEffect } from "react";
import { Search } from "lucide-react";

type Heading = { level: number; text: string };
type Image = { src: string; alt?: string; title?: string };
type SearchResponse = {
  source: string;
  results: Array<{
    id: string | number;
    hybridScore?: number;
    qdrantScore?: number;
    pgScore?: number;
    payload: {
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
    url?: string;
    title?: string;
  }>;
};

const RESULTS_PER_PAGE = 15;

const SearchClient = ({ data }: { data: SearchResponse }) => {
  const [page, setPage] = useState(1);
  const topRef = useRef<HTMLDivElement>(null);

  const results = data.results
    .map((result) => {
      const payload = result.payload ?? {};
      const id = result.id;
      const score =
        result.hybridScore || result.qdrantScore || result.pgScore || 0;
      const url = payload.url || result.url || null;
      const title = payload.title || result.title || "No title available.";
      const description =
        payload.description ||
        payload.text_snippet ||
        (payload.headings && payload.headings[0]?.text) ||
        "No information available.";
      const language = payload.language || null;
      const thumbnail = payload.images && payload.images[0];

      if (!url) return null;

      return {
        id,
        score,
        url,
        title,
        description,
        language,
        thumbnail,
      };
    })
    .filter(Boolean);

  const totalResults = results.length;
  const totalPages = Math.ceil(totalResults / RESULTS_PER_PAGE);

  const startIndex = (page - 1) * RESULTS_PER_PAGE + 1;
  const endIndex = Math.min(startIndex + RESULTS_PER_PAGE - 1, totalResults);

  const paginatedResults = results.slice(startIndex - 1, endIndex);

  // scroll to top when page changes
  useEffect(() => {
    if (topRef.current) {
      topRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [page]);

  return (
    <div className="space-y-6 w-full" ref={topRef}>
      {/* Top bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
        <p className="text-zinc-400 text-sm">
          Showing {startIndex}-{endIndex} of {totalResults} results. Source:{" "}
          {data.source}
        </p>
      </div>

      {/* Search results */}
      <div className="space-y-6 w-full">
        {paginatedResults.map((result) => (
          <div
            key={result!.id}
            className="border-b border-zinc-700/30 pb-6 last:border-b-0"
          >
            <div className="flex items-start gap-4 max-w-6xl">
              {result!.thumbnail?.src ? (
                <img
                  src={result!.thumbnail.src}
                  alt={result!.thumbnail.alt || "thumbnail"}
                  className="w-14 h-14 md:w-20 md:h-20 rounded-lg object-cover bg-zinc-800"
                />
              ) : (
                <div className="w-8 h-8 min-h-8 min-w-8 md:w-14 md:h-14 md:min-h-14 md:min-w-14 bg-gradient-to-br from-emerald-500/20 to-teal-500/20 rounded-lg flex items-center justify-center">
                  <Search className="w-4 h-4 md:w-8 md:h-8 text-emerald-400" />
                </div>
              )}

              <div className="flex flex-col flex-1 overflow-hidden">
                <a
                  href={result!.url}
                  className="text-lg font-semibold hover:text-zinc-200 mb-2 text-emerald-300 cursor-pointer block break-words"
                >
                  {result!.title}
                </a>
                <p className="text-zinc-400 text-xs mb-1 break-words overflow-hidden">
                  {result!.url}
                </p>
                <p className="text-zinc-300 text-sm leading-relaxed line-clamp-2 break-words">
                  {result!.description || "No Description Available"}
                </p>
                {result!.language && result!.language !== "simple" && (
                  <p className="text-xs text-zinc-500 mt-1">
                    Language:{" "}
                    <span className="text-zinc-400">{result!.language}</span>
                  </p>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Pagination Controls */}
      {totalPages > 1 && (
        <div className="flex justify-center items-center gap-2 mt-6">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            className={`px-3 py-1 text-sm rounded ${
              page === 1 ? "" : "cursor-pointer"
            } ${
              page === 1
                ? "text-zinc-500 cursor-not-allowed"
                : "text-emerald-400 hover:text-emerald-300"
            }`}
          >
            Prev
          </button>

          {[...Array(totalPages)].map((_, idx) => (
            <button
              key={idx + 1}
              onClick={() => setPage(idx + 1)}
              className={`px-2 py-1 text-sm rounded ${
                page === idx + 1 ? "" : "cursor-pointer"
              } ${
                page === idx + 1
                  ? "bg-emerald-500/20 text-emerald-300"
                  : "text-zinc-400 hover:text-emerald-300"
              }`}
            >
              {idx + 1}
            </button>
          ))}

          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className={`px-3 py-1 text-sm rounded ${
              page === totalPages ? "" : "cursor-pointer"
            } ${
              page === totalPages
                ? "text-zinc-500 cursor-not-allowed"
                : "text-emerald-400 hover:text-emerald-300"
            }`}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};

export default SearchClient;
