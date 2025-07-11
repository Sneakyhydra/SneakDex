"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

type SearchResponse = {
  source: string;
  results: Array<{
    id: string | number;
    score: number;
    payload: {
      url: string;
      title: string;
      description?: string;
      cleaned_text?: string;
      word_count?: number;
      headings?: Array<{ level: number; text: string }>;
      [key: string]: any;
    };
  }>;
};

const SearchClient = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const query = searchParams.get("q") ?? "";
  const [searchQuery, setSearchQuery] = useState(query);
  const [data, setData] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);

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
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ query: q, top_k: 10 }),
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

  useEffect(() => {
    if (query.trim()) {
      setSearchQuery(query);
      fetchResults(query);
    }
  }, [query]);

  const handleSearch = (e?: React.FormEvent) => {
    e?.preventDefault();
    const trimmedQuery = searchQuery.trim();
    if (!trimmedQuery) return;

    if (trimmedQuery !== query) {
      router.push(`/search?q=${encodeURIComponent(trimmedQuery)}`);
    }
  };

  return (
    <div
      style={{ backgroundColor: "#202124" }}
      className="min-h-screen flex flex-col justify-start w-full px-10 py-5"
    >
      <form className="flex items-center w-full gap-4" onSubmit={handleSearch}>
        <Link href="/" className="text-xl font-bold text-white">
          SneakDex
        </Link>

        <input
          type="text"
          placeholder="Search"
          className="w-full max-w-md p-2.5 border border-gray-300 rounded-full"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />

        <button
          type="submit"
          className="text-white px-4 py-2 cursor-pointer transition-colors duration-300 border rounded-sm border-neutral-700 hover:border-white bg-neutral-700"
        >
          Search
        </button>

        {data && (
          <p className="text-gray-400 py-5 justify-self-end w-full text-right">
            Results: {data.results.length} | Source: {data.source}
          </p>
        )}
      </form>

      {loading && <div className="text-white mt-10">Loading...</div>}

      {!loading && data && data.results.length === 0 && (
        <div className="text-white mt-10">No results found.</div>
      )}

      {!loading && data && data.results.length > 0 && (
        <ul className="space-y-4 mt-6">
          {data.results.map((result) => (
            <li
              key={result.id}
              className="bg-white dark:bg-zinc-900 p-4 rounded-2xl shadow hover:shadow-md transition"
            >
              <a
                href={result.payload.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-600 dark:text-blue-400 hover:underline text-lg font-semibold"
              >
                {result.payload.title || result.payload.url}
              </a>

              <div className="text-gray-500 text-sm">
                Score: {result.score.toFixed(3)} | Words:{" "}
                {result.payload.word_count ?? "?"}
              </div>

              {result.payload.description && (
                <p className="text-gray-700 dark:text-gray-300 mt-2">
                  {result.payload.description}
                </p>
              )}

              {result.payload.headings &&
                result.payload.headings.length > 0 && (
                  <div className="mt-2">
                    <strong className="text-gray-600 dark:text-gray-400">
                      Headings:
                    </strong>
                    <ul className="ml-4 list-disc text-sm text-gray-600 dark:text-gray-400">
                      {result.payload.headings.slice(0, 3).map((h, idx) => (
                        <li key={idx}>
                          H{h.level}: {h.text}
                        </li>
                      ))}
                      {result.payload.headings.length > 3 && <li>…and more</li>}
                    </ul>
                  </div>
                )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default SearchClient;
