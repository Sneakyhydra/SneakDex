"use client";
import { useState, useRef, useEffect } from "react";
import { Image as LucideImage } from "lucide-react";

type SearchImgResponse = {
  source: string;
  results: Array<{
    id: string | number;
    score: number;
    payload: {
      src?: string;
      alt?: string;
      title?: string;
      caption?: string;
      page_url?: string;
      page_title?: string;
      page_description?: string;
      timestamp?: string;
      [key: string]: any;
    };
  }>;
  [key: string]: any;
};

const RESULTS_PER_PAGE = 25;

const SearchImagesClient = ({ data }: { data: SearchImgResponse }) => {
  const [page, setPage] = useState(1);
  const topRef = useRef<HTMLDivElement>(null);

  const results = data.results
    .map((result) => {
      const payload = result.payload ?? {};
      const id = result.id;
      const score = result.score || 0;
      const src = payload.src || null;
      const alt = payload.alt || null;
      const title = payload.title || "No title available.";
      const caption = payload.caption || "No caption available.";
      const page_url = payload.page_url || null;
      const page_title = payload.page_title || "No page title available.";
      const page_description =
        payload.page_description || "No page description available.";

      if (!src) return null;

      return {
        id,
        score,
        src,
        alt,
        title,
        caption,
        page_url,
        page_title,
        page_description,
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
      <div className="flex flex-wrap gap-4 justify-evenly">
        {paginatedResults.map((result) => {
          return (
            <a
              key={result!.id}
              href={result!.page_url || "#"}
              className="group cursor-pointer flex-shrink-0"
              style={{
                minWidth: "150px",
                maxWidth: "300px",
              }}
            >
              <div
                className="
            bg-gradient-to-br from-emerald-500/10 to-teal-500/10
            rounded-lg border border-zinc-700/50
            hover:border-emerald-400/50 transition-all duration-300
            group-hover:scale-105 flex items-center justify-center
            overflow-hidden
          "
                style={{
                  minWidth: "150px",
                  maxWidth: "300px",
                  minHeight: "150px",
                  maxHeight: "300px",
                }}
              >
                {result!.src ? (
                  <img
                    src={result!.src}
                    alt={result!.alt || result!.caption || ""}
                    className="
                object-contain
                max-w-full max-h-full
                m-auto
              "
                  />
                ) : (
                  <LucideImage className="w-8 h-8 text-emerald-400/50" />
                )}
              </div>
              <p className="text-zinc-400 text-xs mt-2 truncate text-center">
                {result!.alt || result!.title || ""}
              </p>
            </a>
          );
        })}
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

export default SearchImagesClient;
