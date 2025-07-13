"use client";

import { Suspense, useState, useEffect } from "react";
import { Search, Image, Sparkles } from "lucide-react";
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
  const [loadedOnce, setLoadedOnce] = useState(false);
  const [loadedImgOnce, setLoadedImgOnce] = useState(false);

  useEffect(() => {
    if (!query.trim()) {
      router.push("/");
    }
  }, [query, router]);

  const fetchResults = async (q: string) => {
    setLoading(true);
    setData(null);
    setLoadedOnce(true);

    try {
      const res = await fetch(`/api/search`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q, top_k: 100 }),
      });

      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const json: SearchResponse = await res.json();
      setData(json);
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
    setLoadedImgOnce(true);

    try {
      const res = await fetch(`/api/search-images`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: q, top_k: 100 }),
      });

      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const json: SearchImgResponse = await res.json();
      setImgData(json);
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
        <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-4 md:gap-6">
          {/* Logo */}
          <div className="flex items-center justify-center gap-3">
            <Link
              href="/"
              className="text-2xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-teal-400"
            >
              SneakDex
            </Link>
            <Sparkles className="w-5 h-5 text-amber-400 animate-pulse" />
          </div>

          {/* Search bar */}
          <form className="flex-1 w-full" onSubmit={handleSearch}>
            <div className="relative flex items-center bg-zinc-900/70 backdrop-blur-xl border border-zinc-700/50 rounded-xl p-1 transition-all duration-300 hover:border-zinc-600/50">
              <div className="pl-4 pr-2">
                <Search className="w-4 h-4 text-zinc-400" />
              </div>
              <input
                type="text"
                placeholder="Search for anything…"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="flex-grow px-2 py-3 bg-transparent text-zinc-100 placeholder-zinc-400 focus:outline-none text-sm"
              />
              <button
                type="submit"
                className="ml-2 px-6 py-3 text-white cursor-pointer rounded-lg font-medium transition-all duration-300 hover:brightness-110 active:scale-95"
                style={{
                  background:
                    "linear-gradient(135deg, #065f46, #047857, #059669)",
                }}
              >
                Search
              </button>
            </div>
          </form>

          {/* Tabs */}
          <div className="flex gap-1 justify-center md:justify-start bg-zinc-900/50 backdrop-blur-sm rounded-xl p-1 border border-zinc-700/50 w-full md:w-auto">
            <button
              onClick={() => {
                setTab("text");
                if (!loadedOnce) {
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
                if (!loadedImgOnce) {
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
