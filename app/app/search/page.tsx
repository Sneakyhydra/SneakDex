"use client";

import { Suspense, useState, useEffect, useCallback, useRef } from "react";
import { Search, Image, Sparkles, Command } from "lucide-react";
import SearchForm from "../_components/SearchForm";
import SearchClient from "./_components/SearchClient";
import SearchImagesClient from "./_components/SearchImagesClient";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

import { SearchResponse, SearchImgResponse } from "../_types/ResponseTypes";
import { useAppContext } from "../_contexts/AppContext";

const SearchPage = () => {
  const router = useRouter();

  const updateUrl = ({
    newQuery,
    newTab,
    newPage,
  }: {
    newQuery: string;
    newTab: string;
    newPage: number;
  }) => {
    router.push(
      `/search?q=${encodeURIComponent(newQuery)}&t=${newTab}&p=${newPage}`
    );
  };

  const searchParams = useSearchParams();
  const query = searchParams.get("q")?.trim() ?? "";
  const tab = searchParams.get("t")?.trim() ?? "";
  const page = searchParams.get("p")?.trim() ?? "";
  const pageNum = parseInt(page, 10);

  const {
    data,
    setData,
    dataQuery,
    setDataQuery,
    imgData,
    setImgData,
    imgDataQuery,
    setImgDataQuery,
    isMobile,
    setIsMobile,
    loading,
    setLoading,
    loadingImg,
    setLoadingImg,
    searchQuery,
    setSearchQuery,
  } = useAppContext();

  const [showAnimations, setShowAnimations] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const searchInputRef = useRef<HTMLInputElement>(null);
  const topRef = useRef<HTMLDivElement>(null);

  // Animation timing
  useEffect(() => {
    const timer = setTimeout(() => {
      setShowAnimations(true);
    }, 150);

    return () => clearTimeout(timer);
  }, []);

  // Mobile detection
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 640);
    };

    checkMobile();
    window.addEventListener("resize", checkMobile);

    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  // Redirect if no query or invalid tab/page
  useEffect(() => {
    if (!query) {
      router.push("/");
      return;
    }
    let newTab = "text";
    let newPage = 1;
    const pageNum = parseInt(page, 10);

    if (!tab || (tab !== "text" && tab !== "images")) {
      if (!page || isNaN(pageNum) || pageNum < 1) {
        updateUrl({ newQuery: query, newTab, newPage: newPage });
        return;
      }
      updateUrl({ newQuery: query, newTab, newPage: pageNum });
      return;
    }

    if (!page || isNaN(pageNum) || pageNum < 1) {
      updateUrl({ newQuery: query, newTab, newPage: newPage });
      return;
    }
  }, [query, tab, page, router]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Focus search on Ctrl/Cmd + K
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }

      // Switch tabs with Ctrl/Cmd + 1/2
      if ((e.ctrlKey || e.metaKey) && e.key === "1") {
        e.preventDefault();
        updateUrl({ newQuery: query, newTab: "text", newPage: 1 });
      }
      if ((e.ctrlKey || e.metaKey) && e.key === "2") {
        e.preventDefault();
        updateUrl({ newQuery: query, newTab: "images", newPage: 1 });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  useEffect(() => {
    if (topRef.current) {
      topRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [page]);

  useEffect(() => {
    if (tab === "text") {
      if (query !== dataQuery) {
        fetchResults(query);
      }
    } else if (tab === "images") {
      if (query !== imgDataQuery) {
        fetchImgResults(query);
      }
    }
  }, [query, tab]);

  const fetchResults = useCallback(async (q: string) => {
    setLoading(true);
    setError(null);
    setData(null);
    setDataQuery(null);

    try {
      const res = await fetch(`/api/search`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q }),
      });

      if (!res.ok) {
        throw new Error(
          res.status === 429
            ? "Rate limit exceeded. Please try again later."
            : `Search failed with status ${res.status}`
        );
      }

      const json: SearchResponse = await res.json();
      setData(json);
      setDataQuery(q);
    } catch (err) {
      if (err instanceof Error) {
        console.error("Error fetching search results", err);
        setError(err.message || "Failed to fetch search results");
        setData(null);
        setDataQuery(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchImgResults = useCallback(async (q: string) => {
    setLoadingImg(true);
    setError(null);
    setImgData(null);
    setImgDataQuery(null);

    try {
      const res = await fetch(`/api/search-images`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q }),
      });

      if (!res.ok) {
        throw new Error(
          res.status === 429
            ? "Rate limit exceeded. Please try again later."
            : `Image search failed with status ${res.status}`
        );
      }

      const json: SearchImgResponse = await res.json();
      setImgData(json);
      setImgDataQuery(q);
    } catch (err) {
      if (err instanceof Error) {
        console.error("Error fetching image results", err);
        setError(err.message || "Failed to fetch image results");
        setImgData(null);
        setImgDataQuery(null);
      }
    } finally {
      setLoadingImg(false);
    }
  }, []);

  const handleTabSwitch = (newTab: string) => {
    updateUrl({ newQuery: query, newTab: newTab, newPage: 1 });
  };

  const renderLoadingState = (type: string) => (
    <div className="text-center text-zinc-400 flex flex-col items-center gap-6 py-20">
      <div className="relative">
        <div className="w-16 h-16 border-4 border-emerald-400/30 border-t-emerald-400 rounded-full animate-spin" />
        <div className="absolute inset-0 w-16 h-16 border-4 border-emerald-400/10 rounded-full animate-ping" />
        <Sparkles className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-6 h-6 text-emerald-400 animate-pulse" />
      </div>
      <div className="space-y-2">
        <p className="font-semibold text-xl">
          {type === "text" ? "Searching the web" : "Finding images"}…
        </p>
        <p className="text-sm text-zinc-500 max-w-md">
          Analyzing lakhs of results to find exactly what you need
        </p>
      </div>
    </div>
  );

  const renderErrorState = () => (
    <div className="text-center text-zinc-400 py-20 space-y-6">
      <div className="bg-gradient-to-br from-red-500/10 via-red-400/10 to-orange-500/10 border border-red-400/20 rounded-2xl p-6 max-w-md mx-auto backdrop-blur-sm">
        <div className="w-12 h-12 bg-red-500/20 rounded-xl flex items-center justify-center mx-auto mb-4">
          <Search className="w-6 h-6 text-red-400" />
        </div>
        <h3 className="font-semibold text-lg mb-2 text-red-300">
          Search Error
        </h3>
        <p className="text-sm text-zinc-300">{error}</p>
      </div>
      <button
        onClick={() =>
          tab === "text" ? fetchResults(query) : fetchImgResults(query)
        }
        className="px-6 py-3 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-700 hover:to-teal-700 text-white rounded-xl transition-all duration-300 font-medium shadow-lg hover:shadow-emerald-500/25 hover:scale-105 active:scale-95"
      >
        Try Again
      </button>
    </div>
  );

  const renderEmptyState = (type: string) => {
    // Only show empty state if we've actually completed a search and got no results
    const hasSearchCompleted =
      type === "text"
        ? !loading && dataQuery === query
        : !loadingImg && imgDataQuery === query;

    if (!hasSearchCompleted) {
      return renderLoadingState(type);
    }

    return (
      <div className="text-center text-zinc-400 py-20 space-y-6" ref={topRef}>
        <div className="space-y-3">
          <div className="w-16 h-16 bg-gradient-to-br from-zinc-700 to-zinc-800 rounded-2xl flex items-center justify-center mx-auto">
            {type === "text" ? (
              <Search className="w-8 h-8 text-zinc-400" />
            ) : (
              <Image className="w-8 h-8 text-zinc-400" />
            )}
          </div>
          <h3 className="font-semibold text-xl text-zinc-300">
            No {type} results found
          </h3>
          <p className="text-zinc-500 max-w-md mx-auto">
            We couldn't find any {type} results for{" "}
            <span className="font-medium text-zinc-300 bg-zinc-800 px-2 py-1 rounded">
              "{query}"
            </span>
          </p>
        </div>
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <button
            onClick={() => handleTabSwitch(type === "text" ? "images" : "text")}
            className="px-6 py-3 bg-gradient-to-r from-zinc-700 to-zinc-800 hover:from-zinc-600 hover:to-zinc-700 text-white rounded-xl transition-all duration-300 hover:scale-105 active:scale-95"
            disabled={type === "text" ? loadingImg : loading}
          >
            {type === "text"
              ? loadingImg
                ? "Loading Images..."
                : "Try Images Search"
              : loading
              ? "Loading Text..."
              : "Try Text Search"}
          </button>
          <button
            onClick={() => {
              setSearchQuery("");
              searchInputRef.current?.focus();
            }}
            className="px-6 py-3 border-2 border-zinc-600 hover:border-zinc-500 text-zinc-300 hover:text-white rounded-xl transition-all duration-300 hover:bg-zinc-800/50 hover:scale-105 active:scale-95"
          >
            New Search
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-zinc-950 via-slate-900 to-neutral-900 text-white flex flex-col px-4 py-6 relative overflow-hidden">
      {/* Enhanced animated background */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div
          className={`absolute top-1/4 left-1/4 w-96 h-96 bg-gradient-to-r from-emerald-500/8 via-teal-500/8 to-cyan-500/8 rounded-full blur-3xl transition-all duration-[4s] ${
            showAnimations
              ? "animate-pulse opacity-100 scale-100"
              : "opacity-0 scale-75"
          }`}
        />
        <div
          className={`absolute bottom-1/4 right-1/4 w-80 h-80 bg-gradient-to-r from-violet-500/8 via-purple-500/8 to-pink-500/8 rounded-full blur-3xl transition-all duration-[4s] ${
            showAnimations
              ? "animate-pulse opacity-100 scale-100"
              : "opacity-0 scale-75"
          }`}
          style={{ animationDelay: "1s" }}
        />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-transparent via-emerald-950/5 to-transparent" />
      </div>

      {/* Enhanced Header */}
      <div className="relative z-10 w-full max-w-7xl mx-auto mb-8">
        <div className="flex items-center flex-col lg:flex-row gap-6 lg:gap-8">
          {/* Logo with back button */}
          <div className="flex items-center gap-4">
            <Link
              href="/"
              className={`transform transition-all duration-700 hover:scale-105 ${
                showAnimations
                  ? "translate-y-0 opacity-100"
                  : "translate-y-8 opacity-0"
              }`}
              aria-label="Go to homepage"
            >
              <div className="relative flex items-center group">
                <div className="relative">
                  <h1 className="relative font-black text-transparent bg-clip-text bg-gradient-to-r from-white via-emerald-200 to-teal-400 text-center tracking-tight text-4xl lg:text-5xl group-hover:from-emerald-300 group-hover:via-teal-300 group-hover:to-cyan-300 transition-all duration-500">
                    SneakDex
                  </h1>
                  <div className="absolute inset-0 font-black text-transparent bg-clip-text bg-gradient-to-r from-emerald-400/40 via-teal-400/40 to-cyan-400/40 text-center tracking-tight blur-xl opacity-60 pointer-events-none text-4xl lg:text-5xl group-hover:opacity-80 transition-opacity duration-500">
                    SneakDex
                  </div>
                  <div className="absolute -inset-4 bg-gradient-to-r from-emerald-400/10 to-teal-400/10 rounded-2xl blur-2xl opacity-30 animate-pulse pointer-events-none group-hover:opacity-50 transition-opacity duration-500" />
                </div>
              </div>
            </Link>
          </div>

          <SearchForm
            showAnimations={showAnimations}
            searchQuery={searchQuery}
            searchInputRef={searchInputRef}
            query={query}
            isMobile={isMobile}
            setSearchQuery={setSearchQuery}
            loading={loading}
            loadingImg={loadingImg}
            tab={tab}
            updateUrl={updateUrl}
          />

          {/* Enhanced Tabs */}
          <div
            className={`transition-all duration-700 delay-300 ${
              showAnimations
                ? "translate-y-0 opacity-100"
                : "translate-y-8 opacity-0"
            }`}
          >
            <div className="flex bg-white/5 backdrop-blur-xl rounded-2xl p-1.5 border border-white/10 shadow-2xl">
              {[
                { key: "text", icon: Search, label: "Text", shortcut: "1" },
                { key: "images", icon: Image, label: "Images", shortcut: "2" },
              ].map(({ key, icon: Icon, label, shortcut }) => (
                <Link
                  key={key}
                  href={`/search?q=${encodeURIComponent(query)}&t=${key}&p=1`}
                >
                  <button
                    // onClick={() => handleTabSwitch(key)}
                    className={`relative px-4 sm:px-6 py-3 rounded-xl font-medium transition-all duration-300 flex items-center gap-2 text-sm min-w-[80px] sm:min-w-[100px] justify-center group overflow-hidden ${
                      tab === key
                        ? "text-white shadow-lg transform scale-105"
                        : "text-zinc-400 hover:text-zinc-200 hover:bg-white/5 hover:scale-102"
                    }`}
                    style={{
                      background:
                        tab === key
                          ? "linear-gradient(135deg, #059669, #10b981, #34d399)"
                          : "transparent",
                    }}
                    title={`Switch to ${label} (Ctrl+${shortcut})`}
                  >
                    {/* Active tab background effect */}
                    {tab === key && (
                      <div className="absolute inset-0 bg-gradient-to-r from-emerald-400/20 to-teal-400/20 rounded-xl opacity-60 animate-pulse" />
                    )}

                    {/* Tab content */}
                    <div className="relative flex items-center gap-2">
                      <Icon className="w-4 h-4" />
                      <span className="hidden sm:inline">{label}</span>
                    </div>
                  </button>
                </Link>
              ))}
            </div>
            {!isMobile && (
              <div className="absolute -bottom-6 left-0 flex items-center gap-1 text-xs text-zinc-500">
                <span className="font-mono text-zinc-500">Ctrl</span>
                <span>/</span>
                <Command
                  className="w-3 h-3 text-zinc-400 inline-block align-text-bottom"
                  aria-label="Command key"
                />
                <span>
                  + <span className="font-mono text-zinc-500">1 / 2</span> to
                  switch
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Enhanced Main Content */}
      <div
        className={`relative z-10 flex-1 w-full max-w-7xl mx-auto transition-all duration-700 delay-400 ${
          showAnimations
            ? "translate-y-0 opacity-100"
            : "translate-y-8 opacity-0"
        }`}
      >
        <div className="bg-white/5 backdrop-blur-xl rounded-3xl border border-white/10 overflow-hidden shadow-2xl">
          <div className="p-4 sm:p-6 min-h-[500px] flex flex-col">
            <Suspense fallback={renderLoadingState(tab)}>
              {error ? (
                renderErrorState()
              ) : tab === "text" ? (
                loading ? (
                  renderLoadingState("text")
                ) : data && data.results.length > 0 ? (
                  <SearchClient page={pageNum} query={query} tab={tab} />
                ) : (
                  renderEmptyState("text")
                )
              ) : loadingImg ? (
                renderLoadingState("images")
              ) : imgData && imgData.results.length > 0 ? (
                <SearchImagesClient page={pageNum} query={query} tab={tab} />
              ) : (
                renderEmptyState("images")
              )}
            </Suspense>
          </div>
        </div>
      </div>

      {/* Results count indicator */}
      {((tab === "text" && data?.results.length) ||
        (tab === "images" && imgData?.results.length)) && (
        <div className="fixed bottom-6 right-6 bg-white/10 backdrop-blur-xl border border-white/20 rounded-2xl px-4 py-2 text-sm text-zinc-300 shadow-2xl animate-pulse">
          <span className="font-medium">
            {tab === "text" ? data?.results.length : imgData?.results.length}{" "}
            results
          </span>
        </div>
      )}
    </div>
  );
};

export default SearchPage;
