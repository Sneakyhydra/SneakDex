"use client";

import { Suspense, useState, useEffect } from "react";
import { Search, Image, Zap } from "lucide-react";
import SearchClient from "./SearchClient";
import SearchImagesClient from "./SearchImagesClient";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

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
  [key: string]: any;
};

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
  const [showAnimations, setShowAnimations] = useState(false);

  useEffect(() => {
    // Trigger animations after component mounts
    const timer = setTimeout(() => {
      setShowAnimations(true);
    }, 100);

    return () => {
      clearTimeout(timer);
    };
  }, []);

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
        <div className="flex items-center flex-row flex-wrap justify-center gap-4 md:gap-8">
          {/* Logo with animations */}
          <Link href="/">
            <div
              className={`transform transition-all duration-1000 ${
                showAnimations
                  ? "translate-y-0 opacity-100"
                  : "translate-y-8 opacity-0"
              }`}
            >
              <div className="relative flex items-center">
                <div className="relative">
                  {/* Foreground text */}
                  <h1
                    className="relative font-black text-transparent bg-clip-text bg-gradient-to-r from-white via-emerald-100 to-teal-300 text-center tracking-tight
      text-5xl"
                  >
                    SneakDex
                  </h1>

                  {/* Glowing text behind */}
                  <div
                    className="absolute inset-0 font-black text-transparent bg-clip-text bg-gradient-to-r from-emerald-300/50 via-teal-300/50 to-lime-300/50 text-center tracking-tight blur-xl opacity-40 pointer-events-none
      text-[clamp(2.5rem,8vw,8rem)]"
                  >
                    SneakDex
                  </div>

                  {/* Orb behind */}
                  <div className="absolute inset-0 bg-gradient-to-r from-emerald-400/20 to-teal-400/20 rounded-full blur-2xl opacity-40 animate-pulse pointer-events-none" />
                </div>
              </div>
            </div>
          </Link>

          {/* Search form */}
          <div
            className={`w-full max-w-2xl min-w-72 transition-all duration-700 delay-200 ${
              showAnimations
                ? "translate-y-0 opacity-100"
                : "translate-y-8 opacity-0"
            }`}
          >
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSearch();
              }}
              className={`px-3 relative flex flex-col justify-between sm:flex-row items-center bg-zinc-900/70 backdrop-blur-xl border border-zinc-700/50 rounded-2xl py-1.5 gap-2 transition-all duration-300 ${
                isSearchFocused
                  ? "border-emerald-400/50 shadow-md shadow-emerald-500/10"
                  : "hover:border-zinc-600/50"
              }`}
            >
              <div className="flex items-center w-full flex-grow">
                {/* Logo — now animated & interactive */}
                <div
                  className={`pr-2 transition-all duration-700 ${
                    showAnimations
                      ? "opacity-100 translate-x-0"
                      : "opacity-0 -translate-x-4"
                  }`}
                >
                  <img
                    src="/favicon.ico"
                    alt="SneakDex Logo"
                    className="w-6 h-6 transition-transform duration-300 hover:scale-110 hover:drop-shadow-[0_0_6px_#34d399]"
                  />
                </div>

                {/* Input */}
                <input
                  type="text"
                  placeholder="Search for anything…"
                  className="flex-grow w-full px-2 py-2 bg-transparent text-zinc-100 placeholder-zinc-400 focus:outline-none text-sm sm:text-base"
                  value={searchQuery}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setSearchQuery(e.target.value)
                  }
                  onFocus={() => setIsSearchFocused(true)}
                  onBlur={() => setIsSearchFocused(false)}
                />
              </div>

              {/* Button */}
              <button
                type="submit"
                className="relative cursor-pointer w-full sm:w-auto px-4 py-2 text-sm text-white rounded-lg font-medium transition-all duration-300 hover:scale-105 active:scale-95 flex items-center justify-center gap-1 hover:brightness-110"
                style={{
                  background:
                    "linear-gradient(135deg, #065f46, #047857, #059669)",
                }}
              >
                <span>Search</span>
              </button>
            </form>
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
