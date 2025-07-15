"use client";

import { Suspense, useState, useEffect } from "react";
import { Search, Image, Zap } from "lucide-react";
import SearchClient from "./SearchClient";
import SearchImagesClient from "./SearchImagesClient";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

type Heading = { level: number; text: string };

type SearchResponse = {
  source: string;
  results: Array<{
    id: string | number;
    score?: number;
    payload: {
      url?: string;
      title?: string;
      description?: string;
      cleaned_text?: string;
      word_count?: number;
      headings?: Heading[];
      [key: string]: any;
    };
  }>;
};

type SearchImgResponse = {
  source: string;
  results: Array<{
    id: string | number;
    score: number;
    rank: number;
    payload: {
      src: string;
      alt?: string;
      title?: string;
      page_url?: string;
      page_title?: string;
      caption?: string;
    };
  }>;
};

const SearchPage = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const query = searchParams.get("q") ?? "";
  const [searchQuery, setSearchQuery] = useState(query);
  const [tab, setTab] = useState<"text" | "images">("text");
  const [data, setData] = useState<SearchResponse | null>(null);
  const [imgData, setImgData] = useState<SearchImgResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingImg, setLoadingImg] = useState(false);
  const [loadedOnce, setLoadedOnce] = useState<string | null>(null);
  const [loadedImgOnce, setLoadedImgOnce] = useState<string | null>(null);

  const [isSearchFocused, setIsSearchFocused] = useState(false);

  useEffect(() => {
    if (!query.trim()) {
      router.push("/");
    }
  }, [query, router]);

  const fetchResults = async (q: string) => {
    setLoading(true);
    setData(null);

    try {
      const res = await fetch(`/api/search`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q }),
      });

      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const json: SearchResponse = await res.json();
      setData(json);
      setLoadedOnce(q);
    } catch (err) {
      console.error("Error fetching search results", err);
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  const fetchImgResults = async (q: string) => {
    setLoadingImg(true);
    setImgData(null);

    try {
      const res = await fetch(`/api/search-images`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q }),
      });

      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const json: SearchImgResponse = await res.json();
      setImgData(json);
      setLoadedImgOnce(q);
    } catch (err) {
      console.error("Error fetching image results", err);
      setImgData(null);
    } finally {
      setLoadingImg(false);
    }
  };

  const handleSearch = (e?: React.FormEvent) => {
    e?.preventDefault();
    const trimmedQuery = searchQuery.trim();
    if (!trimmedQuery) return;

    if (trimmedQuery !== query) {
      router.push(`/search?q=${encodeURIComponent(trimmedQuery)}`);
    }
  };

  useEffect(() => {
    if (query.trim() && tab === "text") {
      fetchResults(query);
    } else if (query.trim() && tab === "images") {
      fetchImgResults(query);
    }
  }, [query]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-zinc-950 via-neutral-900 to-stone-900 text-white flex flex-col px-4 py-6 relative overflow-hidden">
      {/* Animated background */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-72 h-72 bg-gradient-to-r from-emerald-500/5 to-teal-500/5 rounded-full blur-2xl animate-pulse" />
        <div
          className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-gradient-to-r from-teal-500/5 to-lime-500/5 rounded-full blur-2xl animate-pulse"
          style={{ animationDelay: "1s" }}
        />
        <div className="absolute top-20 left-20 w-2 h-2 bg-emerald-400/30 rounded-full animate-ping" />
        <div
          className="absolute bottom-20 right-20 w-3 h-3 bg-teal-400/30 rounded-full animate-pulse"
          style={{ animationDelay: "0.5s" }}
        />
        <div
          className="absolute top-1/3 right-16 w-1 h-1 bg-lime-400/40 rounded-full animate-pulse"
          style={{ animationDelay: "1s" }}
        />
      </div>

      {/* Header */}
      <div className="relative z-10 w-full max-w-6xl mx-auto mb-8">
        <div className="flex items-center flex-row flex-wrap justify-center gap-4 md:gap-6">
          {/* Logo with animations */}
          <Link href="/">
            <div className="relative group flex flex-col items-center">
              {/* Main Title with Logo as S */}
              <h1 className="relative flex items-end text-4xl md:text-6xl font-black text-transparent bg-clip-text bg-gradient-to-r from-white via-emerald-100 to-teal-300 text-center tracking-tight mb-4">
                {/* Logo replacing the S */}
                <div className="relative group/logo inline-block">
                  <img
                    src="/favicon.ico"
                    alt="SneakDex Logo"
                    className="min-w-10 min-h-10 h-10 w-10 md:min-h-16 md:min-w-16 md:h-16 md:w-16 object-contain filter drop-shadow-lg transition-all duration-300 group-hover/logo:scale-110 group-hover/logo:filter group-hover/logo:brightness-110"
                    style={{
                      filter: "drop-shadow(0 0 20px rgba(16, 185, 129, 0.3))",
                    }}
                  />
                </div>
                <span className="pb-0.5 md:pb-1">neakDex</span>
              </h1>

              {/* Glowing effect */}
              <div className="absolute inset-0 flex items-center justify-center text-5xl md:text-7xl font-black text-transparent bg-clip-text bg-gradient-to-r from-emerald-400/20 via-teal-400/20 to-lime-400/20 text-center tracking-tight blur-sm group-hover:blur-md transition-all duration-300">
                {/* Glow effect behind logo */}
                <div className="absolute inset-0 bg-gradient-to-r from-emerald-400/20 to-teal-400/20 rounded-full blur-xl opacity-0 group-hover/logo:opacity-100 transition-all duration-300 -z-10" />

                {/* Animated ring around logo */}
                <div className="absolute inset-0 rounded-full border-2 border-emerald-400/0 group-hover/logo:border-emerald-400/30 transition-all duration-300 animate-pulse" />
                <span className="mb-2.5">neakDex</span>
              </div>
            </div>
          </Link>

          {/* Search form */}
          <div className="w-full max-w-2xl min-w-72 transition-all duration-700 delay-200">
            <div className="relative">
              {/* Search input container */}
              <div
                className={`px-4 relative flex flex-wrap items-center bg-zinc-900/70 backdrop-blur-xl border border-zinc-700/50 rounded-2xl py-1 transition-all duration-300 ${
                  isSearchFocused
                    ? "border-emerald-400/50 shadow-lg shadow-emerald-500/20"
                    : "hover:border-zinc-600/50"
                }`}
              >
                {/* Search icon */}
                <div className="pr-3">
                  <Search
                    className={`w-4 h-4 transition-colors duration-300 ${
                      isSearchFocused ? "text-emerald-400" : "text-zinc-400"
                    }`}
                  />
                </div>

                {/* Input field */}
                <input
                  type="text"
                  placeholder="Search for anything…"
                  className="flex-grow px-2 py-4 bg-transparent text-zinc-100 placeholder-zinc-400 focus:outline-none text-lg"
                  value={searchQuery}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setSearchQuery(e.target.value)
                  }
                  onFocus={() => setIsSearchFocused(true)}
                  onBlur={() => setIsSearchFocused(false)}
                  onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                    if (e.key === "Enter") {
                      handleSearch();
                    }
                  }}
                />

                {/* Search button with isolated hover */}
                <button
                  type="submit"
                  onClick={handleSearch}
                  className="relative justify-center cursor-pointer w-full px-8 py-4 text-white rounded-xl font-medium transition-all duration-300 hover:scale-105 hover:shadow-lg hover:shadow-emerald-500/25 active:scale-95 flex items-center gap-2 overflow-hidden group/button hover:brightness-110"
                  style={{
                    background:
                      "linear-gradient(135deg, #065f46, #047857, #059669)",
                  }}
                >
                  {/* Button content */}
                  <span className="relative z-10">Search</span>
                  <Zap className="w-4 h-4 relative z-10" />
                </button>
              </div>

              {/* Animated border glow */}
              <div
                className={`absolute inset-0 bg-gradient-to-r from-emerald-500/20 to-teal-500/20 rounded-2xl blur-xl transition-opacity duration-300 -z-10 ${
                  isSearchFocused ? "opacity-100" : "opacity-0"
                }`}
              />
            </div>
          </div>

          {/* Tabs */}
          <div className="flex gap-1 max-w-2xl min-w-60 justify-center md:justify-start bg-zinc-900/50 backdrop-blur-sm rounded-xl p-1 border border-zinc-700/50 w-full">
            <button
              onClick={() => {
                setTab("text");
                if (!loadedOnce || (loadedOnce && loadedOnce !== query)) {
                  fetchResults(query);
                }
              }}
              className={`flex-1 px-4 py-2 rounded-lg font-medium transition-all duration-300 flex items-center justify-center gap-2 text-sm ${
                tab === "text"
                  ? "text-white shadow-lg"
                  : "text-zinc-400 cursor-pointer hover:text-zinc-200 hover:bg-zinc-800/50"
              }`}
              style={{
                background:
                  tab === "text"
                    ? "linear-gradient(135deg, #065f46, #047857, #059669)"
                    : "transparent",
              }}
            >
              <Search className="w-3 h-3" />
              Text
            </button>
            <button
              onClick={() => {
                setTab("images");
                if (
                  !loadedImgOnce ||
                  (loadedImgOnce && loadedImgOnce !== query)
                ) {
                  fetchImgResults(query);
                }
              }}
              className={`flex-1 px-4 py-2 rounded-lg font-medium transition-all duration-300 flex items-center justify-center gap-2 text-sm ${
                tab === "images"
                  ? "text-white shadow-lg"
                  : "text-zinc-400 cursor-pointer hover:text-zinc-200 hover:bg-zinc-800/50"
              }`}
              style={{
                background:
                  tab === "images"
                    ? "linear-gradient(135deg, #065f46, #047857, #059669)"
                    : "transparent",
              }}
            >
              <Image className="w-3 h-3" />
              Images
            </button>
          </div>
        </div>
      </div>

      {/* Main Content - no grid */}
      <div className="relative z-10 flex-1 w-full max-w-6xl mx-auto flex flex-col">
        <div className="flex-1 flex flex-col">
          <div className="bg-zinc-900/30 backdrop-blur-sm rounded-2xl border border-zinc-700/50 p-6 flex-1 flex flex-col min-h-96">
            <Suspense
              fallback={
                <div className="text-center text-zinc-400 flex flex-col items-center gap-4 py-12">
                  <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                  <p>Loading search results…</p>
                </div>
              }
            >
              {tab === "text" ? (
                loading ? (
                  <div className="text-center text-zinc-400 flex flex-col items-center gap-4 py-12">
                    <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                    <p>Searching…</p>
                  </div>
                ) : !data || data.results.length === 0 ? (
                  <div className="text-center text-zinc-400 py-12">
                    No results found for <strong>{query}</strong>.
                  </div>
                ) : (
                  <SearchClient data={data} />
                )
              ) : loadingImg ? (
                <div className="text-center text-zinc-400 flex flex-col items-center gap-4 py-12">
                  <div className="w-8 h-8 border-2 border-teal-400 border-t-transparent rounded-full animate-spin" />
                  <p>Searching images…</p>
                </div>
              ) : !imgData || imgData.results.length === 0 ? (
                <div className="text-center text-zinc-400 py-12">
                  No image results found for <strong>{query}</strong>.
                </div>
              ) : (
                <SearchImagesClient data={imgData} />
              )}
            </Suspense>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SearchPage;
