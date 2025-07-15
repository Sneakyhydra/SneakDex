"use client";
import { useState, useRef, useEffect } from "react";
import { Image as LucideImage, X } from "lucide-react";
import ReactDOM from "react-dom";

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
  const [preview, setPreview] = useState<null | {
    src?: string;
    alt?: string;
    title?: string;
    caption?: string | null;
    page_url: string | null;
    page_title: string | null;
    page_description: string | null;
  }>(null);
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

  useEffect(() => {
    if (topRef.current) {
      topRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [page]);

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === "Escape") setPreview(null);
    };
    document.addEventListener("keydown", handleEsc);
    return () => document.removeEventListener("keydown", handleEsc);
  }, []);

  return (
    <div className="space-y-6 w-full" ref={topRef}>
      {/* Top bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
        <p className="text-zinc-400 text-sm">
          Showing {startIndex}-{endIndex} of {totalResults} results. Source:{" "}
          {data.source}
        </p>
      </div>

      {/* Grid */}
      <div className="flex flex-wrap gap-4 justify-evenly">
        {paginatedResults.map((result) => (
          <div
            key={result!.id}
            className="group cursor-pointer flex-shrink-0"
            style={{
              minWidth: "150px",
              maxWidth: "300px",
            }}
            onClick={() =>
              setPreview({
                src: result!.src,
                alt: result!.alt || result!.title,
                title: result!.title,
                caption: result!.caption,
                page_url: result!.page_url,
                page_title: result!.page_title,
                page_description: result!.page_description,
              })
            }
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
                  title={result!.title}
                  className="object-contain max-w-full max-h-full m-auto"
                />
              ) : (
                <LucideImage className="w-8 h-8 text-emerald-400/50" />
              )}
            </div>
            <p className="text-zinc-400 text-xs mt-2 truncate text-center">
              {result!.alt || result!.title || result!.caption}
            </p>
          </div>
        ))}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center items-center gap-2 mt-6">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            className={`px-3 py-1 text-sm rounded ${
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
              page === totalPages
                ? "text-zinc-500 cursor-not-allowed"
                : "text-emerald-400 hover:text-emerald-300"
            }`}
          >
            Next
          </button>
        </div>
      )}

      {preview &&
        ReactDOM.createPortal(
          <div
            className="fixed inset-0 bg-black/80 flex items-center justify-center z-50"
            onClick={() => setPreview(null)}
          >
            <div
              className="relative max-w-4xl w-full p-4 flex flex-col items-center justify-center"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => setPreview(null)}
                className="absolute top-2 right-2 text-zinc-300 hover:text-zinc-100"
              >
                <X className="w-6 h-6" />
              </button>

              <div
                className="w-full flex items-center justify-center"
                style={{ maxHeight: "80vh" }}
              >
                <img
                  src={preview!.src}
                  alt={preview!.alt || preview!.caption || ""}
                  title={preview!.title}
                  className="object-contain max-h-[80vh] w-auto rounded shadow-lg"
                />
              </div>

              <div className="mt-4 text-center max-w-lg">
                <p className="text-zinc-200">{preview.page_title}</p>
                {preview.page_url && (
                  <a
                    href={preview.page_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-emerald-400 hover:underline text-sm"
                  >
                    View page
                  </a>
                )}
                <p className="text-zinc-200">{preview.page_description}</p>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
};

export default SearchImagesClient;
