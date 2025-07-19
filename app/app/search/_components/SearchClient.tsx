"use client";
import { useEffect, useMemo } from "react";
import {
  ExternalLink,
  Clock,
  Globe,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

import { ImageType } from "../../_types/ResponseTypes";
import {
  urlToTitle,
  ThumbnailImage,
  getDomainFromUrl,
  formatTimestamp,
} from "./_utils/Utils";
import { useAppContext } from "../../_contexts/AppContext";

const RESULTS_PER_PAGE = 15;

const SearchClient = ({
  page,
  query,
  tab,
  updateUrl,
}: {
  page: number;
  query: string;
  tab: string;
  updateUrl: ({
    newQuery,
    newTab,
    newPage,
  }: {
    newQuery: string;
    newTab: string;
    newPage: number;
  }) => void;
}) => {
  const { data } = useAppContext();

  if (!data) {
    return <></>;
  }

  const results = useMemo(() => {
    return data.results
      .map((result) => {
        const payload = result.payload ?? {};
        const id = result.id;
        const score =
          result.hybridScore || result.qdrantScore || result.pgScore || 0;
        const url = payload.url || result.url || null;

        let title = payload.title || result.title || "No title available.";
        if (title === "No Title" || title === "No title available.") {
          if (url) {
            title = urlToTitle(url);
          }
        }

        let description =
          payload.description ||
          payload.text_snippet ||
          (payload.headings && payload.headings[0]?.text) ||
          null;

        if (!description) {
          description = "No preview available. Click to visit.";
        }

        const language = payload.language || null;
        const thumbnail = payload.images && payload.images[0];
        const timestamp = payload.timestamp || null;

        if (!url) return null;

        return {
          id,
          score,
          url,
          title,
          description,
          language,
          thumbnail,
          timestamp,
        };
      })
      .filter(Boolean) as {
      id: string | number;
      score: number;
      url: string;
      title: string;
      description: string;
      language: string | null;
      thumbnail: ImageType | undefined;
      timestamp: string | null;
    }[];
  }, [data.results]);

  const totalResults = results.length;
  const totalPages = Math.ceil(totalResults / RESULTS_PER_PAGE);
  const startIndex = (page - 1) * RESULTS_PER_PAGE + 1;
  const endIndex = Math.min(startIndex + RESULTS_PER_PAGE - 1, totalResults);

  useEffect(() => {
    if (page > totalPages) {
      updateUrl({ newQuery: query, newTab: tab, newPage: totalPages });
    }
  }, [page]);

  const paginatedResults = useMemo(() => {
    return results.slice(startIndex - 1, endIndex);
  }, [results, startIndex, endIndex]);

  const renderPaginationButton = (pageNum: number) => (
    <button
      key={pageNum}
      onClick={() =>
        updateUrl({ newQuery: query, newTab: tab, newPage: pageNum })
      }
      className={`min-w-8 h-8 sm:min-w-10 sm:h-10 rounded-lg text-xs sm:text-sm font-medium transition-all duration-200 ${
        page === pageNum
          ? "bg-emerald-500 text-white shadow-lg shadow-emerald-500/20"
          : "text-zinc-400 hover:text-emerald-300 hover:bg-zinc-800/50"
      }`}
    >
      {pageNum}
    </button>
  );

  const renderPagination = () => {
    if (totalPages <= 1) return null;

    const buttons = [];
    // Responsive max visible pages
    const maxVisible =
      window.innerWidth < 640 ? 3 : window.innerWidth < 768 ? 5 : 7;

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        buttons.push(renderPaginationButton(i));
      }
    } else {
      if (maxVisible <= 3) {
        // Mobile: Show only current page and adjacent pages
        const start = Math.max(1, page - 1);
        const end = Math.min(totalPages, page + 1);

        for (let i = start; i <= end; i++) {
          buttons.push(renderPaginationButton(i));
        }
      } else {
        // Desktop/tablet: Show more pages
        buttons.push(renderPaginationButton(1));

        if (page > 3) {
          buttons.push(
            <span
              key="ellipsis1"
              className="px-1 sm:px-2 text-zinc-500 text-xs"
            >
              ...
            </span>
          );
        }

        const start = Math.max(2, page - 1);
        const end = Math.min(totalPages - 1, page + 1);

        for (let i = start; i <= end; i++) {
          buttons.push(renderPaginationButton(i));
        }

        if (page < totalPages - 2) {
          buttons.push(
            <span
              key="ellipsis2"
              className="px-1 sm:px-2 text-zinc-500 text-xs"
            >
              ...
            </span>
          );
        }

        if (totalPages > 1) {
          buttons.push(renderPaginationButton(totalPages));
        }
      }
    }

    return buttons;
  };

  return (
    <div className="space-y-8 w-full">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 py-4 border-b border-zinc-800/50">
        <div className="flex items-center gap-3">
          <div className="w-2 h-8 bg-gradient-to-b from-emerald-400 to-teal-400 rounded-full"></div>
          <div>
            <p className="text-zinc-300 text-sm font-medium">
              Top {totalResults.toLocaleString()} results. Searched{" "}
              {data.totalAvailable.qdrant + data.totalAvailable.postgres}{" "}
              documents.
            </p>
            <p className="text-zinc-500 text-xs">
              Source: <span className="text-emerald-400">{data.source}</span>
            </p>
          </div>
        </div>

        <div className="text-xs text-zinc-500 bg-zinc-900/50 px-3 py-1.5 rounded-full border border-zinc-800">
          Page {page} of {totalPages}
        </div>
      </div>

      {/* Results */}
      <div className="space-y-6">
        {paginatedResults.map((result, index) => (
          <article
            key={result.id}
            className="group p-4 rounded-xl border border-zinc-800/50 bg-zinc-900/20 hover:bg-zinc-900/40 hover:border-emerald-500/30 transition-all duration-300"
          >
            <div className="flex gap-4">
              <ThumbnailImage result={result} />

              <div className="flex-1 min-w-0 space-y-3">
                {/* Title and URL */}
                <div className="space-y-2">
                  <a
                    href={result.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block group/link"
                  >
                    <h3 className="text-lg font-semibold text-emerald-300 group-hover/link:text-emerald-200 transition-colors duration-200 line-clamp-2">
                      {result.title}
                      <ExternalLink className="inline-block w-4 h-4 ml-2 opacity-0 group-hover/link:opacity-60 transition-opacity duration-200" />
                    </h3>
                  </a>

                  <div className="flex items-center gap-2 text-xs">
                    <Globe className="w-3 h-3 text-zinc-500" />
                    <span className="text-zinc-400 font-mono truncate">
                      {result.url}
                    </span>
                  </div>
                </div>

                {/* Description */}
                <p className="text-zinc-300 text-sm leading-relaxed line-clamp-3">
                  {result.description}
                </p>

                {/* Metadata */}
                <div className="flex flex-wrap items-center gap-4 text-xs text-zinc-500">
                  {result.timestamp && (
                    <div className="flex items-center gap-1.5">
                      <Clock className="w-3 h-3" />
                      <span>Updated {formatTimestamp(result.timestamp)}</span>
                    </div>
                  )}

                  {result.language &&
                    result.language !== "simple" &&
                    result.language !== "en" && (
                      <div className="px-2 py-1 bg-zinc-800/50 rounded-md border border-zinc-700/50">
                        {result.language.toUpperCase()}
                      </div>
                    )}

                  <div className="px-2 py-1 bg-emerald-500/10 text-emerald-400 rounded-md border border-emerald-500/20 font-medium">
                    #{startIndex + index}
                  </div>
                </div>
              </div>
            </div>
          </article>
        ))}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex flex-col sm:flex-row justify-center items-center gap-4 py-8">
          {/* Mobile: Simple prev/next with page info */}
          <div className="flex sm:hidden items-center justify-between w-full max-w-sm">
            <button
              onClick={() =>
                updateUrl({
                  newQuery: query,
                  newTab: tab,
                  newPage: Math.max(1, page - 1),
                })
              }
              disabled={page === 1}
              className={`flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                page === 1
                  ? "text-zinc-600 cursor-not-allowed"
                  : "text-emerald-400 hover:text-emerald-300 hover:bg-zinc-800/50"
              }`}
            >
              <ChevronLeft className="w-4 h-4" />
              Prev
            </button>

            <div className="text-sm text-zinc-400 font-medium">
              {page} / {totalPages}
            </div>

            <button
              onClick={() =>
                updateUrl({
                  newQuery: query,
                  newTab: tab,
                  newPage: Math.min(totalPages, page + 1),
                })
              }
              disabled={page === totalPages}
              className={`flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                page === totalPages
                  ? "text-zinc-600 cursor-not-allowed"
                  : "text-emerald-400 hover:text-emerald-300 hover:bg-zinc-800/50"
              }`}
            >
              Next
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>

          {/* Desktop: Full pagination */}
          <div className="hidden sm:flex justify-center items-center gap-2">
            <button
              onClick={() =>
                updateUrl({
                  newQuery: query,
                  newTab: tab,
                  newPage: Math.max(1, page - 1),
                })
              }
              disabled={page === 1}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                page === 1
                  ? "text-zinc-600 cursor-not-allowed"
                  : "text-emerald-400 hover:text-emerald-300 hover:bg-zinc-800/50"
              }`}
            >
              <ChevronLeft className="w-4 h-4" />
              Previous
            </button>

            <div className="flex items-center gap-1 mx-4">
              {renderPagination()}
            </div>

            <button
              onClick={() =>
                updateUrl({
                  newQuery: query,
                  newTab: tab,
                  newPage: Math.min(totalPages, page + 1),
                })
              }
              disabled={page === totalPages}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                page === totalPages
                  ? "text-zinc-600 cursor-not-allowed"
                  : "text-emerald-400 hover:text-emerald-300 hover:bg-zinc-800/50"
              }`}
            >
              Next
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default SearchClient;
