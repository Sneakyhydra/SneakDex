"use client";

import { useState, useMemo, useEffect } from "react";
import {
  Image as LucideImage,
  X,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
  Download,
  Eye,
  Globe,
  Maximize2,
} from "lucide-react";
import ReactDOM from "react-dom";
import clsx from "clsx";

import { useAppContext } from "../../_contexts/AppContext";

const RESULTS_PER_PAGE = 25;

const SearchImagesClient = ({
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
  const { imgData } = useAppContext();

  if (!imgData) {
    return <></>;
  }

  const [previewIndex, setPreviewIndex] = useState<number | null>(null);
  const [fade, setFade] = useState(false);
  const [gridSize, setGridSize] = useState<"small" | "medium" | "large">(
    "medium"
  );
  const [loaded, setLoaded] = useState<Record<string, boolean>>({});
  const [loadError, setLoadError] = useState<Record<string, boolean>>({});

  const results = useMemo(() => {
    return imgData.results
      .map((result) => {
        const p = result.payload ?? {};
        if (!p.src) return null;

        return {
          id: result.id,
          score: result.score ?? 0,
          src: p.src,
          alt: p.alt || p.caption || p.title || "Image",
          title: p.title || "Untitled",
          caption: p.caption || "",
          page_url: p.page_url || null,
          page_title: p.page_title || "Untitled page",
          page_description: p.page_description || "",
        };
      })
      .filter(Boolean) as {
      id: string | number;
      score: number;
      src: string;
      alt: string;
      title: string;
      caption: string;
      page_url: string | null;
      page_title: string;
      page_description: string;
    }[];
  }, [imgData.results]);

  const totalResults = results.length;
  const totalPages = Math.ceil(totalResults / RESULTS_PER_PAGE);
  const startIndex = (page - 1) * RESULTS_PER_PAGE + 1;
  const endIndex = Math.min(startIndex + RESULTS_PER_PAGE - 1, totalResults);
  const paginatedResults = results.slice(startIndex - 1, endIndex);

  useEffect(() => {
    if (page > totalPages) {
      updateUrl({ newQuery: query, newTab: tab, newPage: totalPages });
    }
  }, [page]);

  const openModal = (index: number) => {
    setPreviewIndex(index);
    setFade(true);
    document.body.style.overflow = "hidden";
  };

  const closeModal = () => {
    setFade(false);
    document.body.style.overflow = "unset";
    setTimeout(() => setPreviewIndex(null), 300);
  };

  const handleImageLoad = (id: string | number) => {
    setLoaded((prev) => ({ ...prev, [id]: true }));
  };

  const handleImageError = (id: string | number) => {
    setLoadError((prev) => ({ ...prev, [id]: true }));
  };

  const navigate = (direction: "prev" | "next") => {
    if (previewIndex === null) return;
    const nextIndex =
      direction === "prev"
        ? (previewIndex - 1 + paginatedResults.length) % paginatedResults.length
        : (previewIndex + 1) % paginatedResults.length;
    setPreviewIndex(nextIndex);
  };

  const downloadImage = (src: string, filename: string) => {
    const link = document.createElement("a");
    link.href = src;
    link.download = filename || "image";
    link.target = "_blank";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const getDomainFromUrl = (url: string): string => {
    try {
      return new URL(url).hostname.replace(/^www\./, "");
    } catch {
      return url;
    }
  };

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (previewIndex === null) return;
      if (e.key === "Escape") closeModal();
      if (e.key === "ArrowLeft") navigate("prev");
      if (e.key === "ArrowRight") navigate("next");
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [previewIndex]);

  const getGridClasses = () => {
    switch (gridSize) {
      case "small":
        return "grid-cols-4 sm:grid-cols-6 lg:grid-cols-8";
      case "large":
        return "grid-cols-2 sm:grid-cols-3 lg:grid-cols-4";
      default:
        return "grid-cols-3 sm:grid-cols-4 lg:grid-cols-6";
    }
  };

  const getImageClasses = () => {
    switch (gridSize) {
      case "small":
        return "aspect-square";
      case "large":
        return "aspect-[4/3]";
      default:
        return "aspect-square";
    }
  };

  // Responsive pagination logic
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
    const maxVisible = 5; // Reduced for images

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        buttons.push(renderPaginationButton(i));
      }
    } else {
      // Show current page and adjacent pages
      const start = Math.max(1, page - 2);
      const end = Math.min(totalPages, page + 2);

      if (start > 1) {
        buttons.push(renderPaginationButton(1));
        if (start > 2) {
          buttons.push(
            <span
              key="ellipsis1"
              className="px-1 sm:px-2 text-zinc-500 text-xs"
            >
              ...
            </span>
          );
        }
      }

      for (let i = start; i <= end; i++) {
        buttons.push(renderPaginationButton(i));
      }

      if (end < totalPages) {
        if (end < totalPages - 1) {
          buttons.push(
            <span
              key="ellipsis2"
              className="px-1 sm:px-2 text-zinc-500 text-xs"
            >
              ...
            </span>
          );
        }
        buttons.push(renderPaginationButton(totalPages));
      }
    }

    return buttons;
  };

  const preview = previewIndex !== null ? paginatedResults[previewIndex] : null;

  return (
    <div className="space-y-6 w-full relative">
      {/* Enhanced Header */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 py-4 border-b border-zinc-800/50">
        <div className="flex items-center gap-3">
          <div className="w-2 h-8 bg-gradient-to-b from-emerald-400 to-teal-400 rounded-full"></div>
          <div>
            <p className="text-zinc-300 text-sm font-medium">
              Top {totalResults.toLocaleString()} images
            </p>
            <p className="text-zinc-500 text-xs">
              Source: <span className="text-emerald-400">{imgData.source}</span>
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Grid Size Controls */}
          <div className="flex items-center gap-2 bg-zinc-900/50 p-1 rounded-lg border border-zinc-800">
            <button
              onClick={() => setGridSize("small")}
              className={`p-2 rounded text-xs transition-all ${
                gridSize === "small"
                  ? "bg-emerald-500 text-white"
                  : "text-zinc-400 hover:text-emerald-300"
              }`}
            >
              Small
            </button>
            <button
              onClick={() => setGridSize("medium")}
              className={`p-2 rounded text-xs transition-all ${
                gridSize === "medium"
                  ? "bg-emerald-500 text-white"
                  : "text-zinc-400 hover:text-emerald-300"
              }`}
            >
              Medium
            </button>
            <button
              onClick={() => setGridSize("large")}
              className={`p-2 rounded text-xs transition-all ${
                gridSize === "large"
                  ? "bg-emerald-500 text-white"
                  : "text-zinc-400 hover:text-emerald-300"
              }`}
            >
              Large
            </button>
          </div>

          <div className="text-xs text-zinc-500 bg-zinc-900/50 px-3 py-1.5 rounded-full border border-zinc-800">
            Page {page} of {totalPages}
          </div>
        </div>
      </div>

      {/* Enhanced Image Grid */}
      <div className={`grid gap-3 ${getGridClasses()}`}>
        {paginatedResults.map((result, idx) => (
          <div
            key={result!.id}
            className="group cursor-pointer relative"
            onClick={() => openModal(idx)}
          >
            <div
              className={`
                bg-zinc-900/50 rounded-xl border border-zinc-800/50
                hover:border-emerald-400/50 transition-all duration-300
                group-hover:scale-[1.02] overflow-hidden relative
                ${getImageClasses()}
              `}
            >
              {/* Loading State */}
              {!loaded[result!.id] && !loadError[result!.id] && (
                <div className="absolute inset-0 animate-pulse bg-zinc-800/50 flex items-center justify-center">
                  <LucideImage className="w-8 h-8 text-zinc-600" />
                </div>
              )}

              {/* Error State */}
              {loadError[result!.id] && (
                <div className="absolute inset-0 bg-zinc-800/50 flex flex-col items-center justify-center text-zinc-500">
                  <X className="w-6 h-6 mb-2" />
                  <span className="text-xs">Failed to load</span>
                </div>
              )}

              {/* Image */}
              <img
                src={result!.src}
                alt={result!.alt}
                title={result!.title}
                onLoad={() => handleImageLoad(result!.id)}
                onError={() => handleImageError(result!.id)}
                className={clsx(
                  "w-full h-full object-cover transition-all duration-300",
                  loaded[result!.id] ? "opacity-100" : "opacity-0"
                )}
              />

              {/* Hover Overlay */}
              <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center justify-center">
                <div className="flex gap-2">
                  <div className="p-2 bg-emerald-500 rounded-full text-white">
                    <Eye className="w-4 h-4" />
                  </div>
                  <div className="p-2 bg-zinc-800/80 rounded-full text-white">
                    <Maximize2 className="w-4 h-4" />
                  </div>
                </div>
              </div>
            </div>

            {/* Image Title */}
            {gridSize === "large" && (
              <div className="mt-2 px-1">
                <p
                  className="text-zinc-400 text-xs truncate"
                  title={result!.alt}
                >
                  {result!.alt}
                </p>
              </div>
            )}
          </div>
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

      {/* Enhanced Preview Modal */}
      {preview &&
        ReactDOM.createPortal(
          <div
            className={clsx(
              "fixed inset-0 z-50 flex items-center justify-center p-4 transition-all duration-300",
              fade ? "opacity-100 backdrop-blur-sm" : "opacity-0"
            )}
            style={{ background: "rgba(0,0,0,0.95)" }}
            onClick={closeModal}
          >
            <div
              className="relative max-w-7xl max-h-full w-full bg-zinc-900/95 backdrop-blur-sm rounded-2xl border border-zinc-800 shadow-2xl overflow-hidden"
              onClick={(e) => e.stopPropagation()}
            >
              {/* Modal Header */}
              <div className="flex items-center justify-between p-4 border-b border-zinc-800">
                <div className="flex items-center gap-3">
                  <div className="text-emerald-400 text-sm font-medium">
                    {previewIndex! + 1} / {paginatedResults.length}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => downloadImage(preview!.src, preview!.title)}
                    className="p-2 hover:bg-zinc-800 rounded-lg transition-colors text-zinc-400 hover:text-emerald-300"
                    title="Download image"
                  >
                    <Download className="w-5 h-5" />
                  </button>
                  <button
                    onClick={closeModal}
                    className="p-2 hover:bg-zinc-800 rounded-lg transition-colors text-zinc-400 hover:text-emerald-300"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>

              <div className="flex flex-col lg:flex-row h-full max-h-[80vh]">
                {/* Image Container */}
                <div className="relative flex-1 flex items-center justify-center p-4 min-h-[400px]">
                  <img
                    src={preview!.src}
                    alt={preview!.alt}
                    className="max-w-full max-h-full object-contain rounded-lg shadow-2xl"
                  />

                  {/* Navigation Buttons */}
                  <button
                    onClick={() => navigate("prev")}
                    className="absolute left-4 top-1/2 -translate-y-1/2 p-3 bg-black/70 hover:bg-black/90 text-white rounded-full transition-all hover:scale-105"
                  >
                    <ChevronLeft className="w-6 h-6" />
                  </button>

                  <button
                    onClick={() => navigate("next")}
                    className="absolute right-4 top-1/2 -translate-y-1/2 p-3 bg-black/70 hover:bg-black/90 text-white rounded-full transition-all hover:scale-105"
                  >
                    <ChevronRight className="w-6 h-6" />
                  </button>
                </div>

                {/* Image Details Sidebar */}
                <div className="w-full lg:w-80 border-t lg:border-t-0 lg:border-l border-zinc-800 p-6 space-y-6 overflow-y-auto">
                  <div>
                    <h2 className="text-lg font-semibold text-emerald-300 mb-2">
                      {preview!.title}
                    </h2>
                    {preview!.caption && (
                      <p className="text-zinc-400 text-sm leading-relaxed">
                        {preview!.caption}
                      </p>
                    )}
                  </div>

                  {preview!.page_url && (
                    <div className="space-y-3">
                      <h3 className="text-md font-medium text-emerald-400 flex items-center gap-2">
                        <Globe className="w-4 h-4" />
                        Source Page
                      </h3>

                      <div className="space-y-2">
                        <p className="text-zinc-300 text-sm font-medium">
                          {preview!.page_title}
                        </p>

                        <p className="text-xs text-zinc-500 font-mono">
                          {getDomainFromUrl(preview!.page_url)}
                        </p>

                        {preview!.page_description && (
                          <p className="text-zinc-400 text-sm leading-relaxed">
                            {preview!.page_description}
                          </p>
                        )}

                        <a
                          href={preview!.page_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-2 text-emerald-300 hover:text-emerald-200 text-sm transition-colors"
                        >
                          <ExternalLink className="w-4 h-4" />
                          Visit Source Page
                        </a>
                      </div>
                    </div>
                  )}

                  <div className="pt-4 border-t border-zinc-800">
                    <a
                      href={preview!.src}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-emerald-300 hover:text-emerald-200 text-sm transition-colors"
                    >
                      <ExternalLink className="w-4 h-4" />
                      Open Original Image
                    </a>
                  </div>
                </div>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
};

export default SearchImagesClient;
