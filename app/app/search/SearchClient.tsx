"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

type SearchResponse = {
  query: string;
  results: Array<{
    url: string;
    title: string;
    score: number;
  }>;
  total_results: number;
  time_ms: number;
};

const SearchClient = () => {
  const router = useRouter();
  const searchParams = useSearchParams();

  const query = searchParams.get("q");
  if (!query) {
    router.push("/");
    return null;
  }

  const [data, setData] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState(query);

  const handleSearch = () => {
    const trimmedQuery = searchQuery.trim();
    if (trimmedQuery === "") {
      return;
    }

    setSearchQuery("");
    router.push(`/search?q=${encodeURIComponent(trimmedQuery)}`);
  };

  useEffect(() => {
    fetch(`http://localhost:8000/search?q=${encodeURIComponent(query)}`)
      .then((res) => res.json())
      .then((data) => {
        setData(data);
        setLoading(false);
      });
  }, [query]);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (!data) {
    return <div>No results found.</div>;
  }

  return (
    <div
      style={{ backgroundColor: "#202124" }}
      className="min-h-screen flex flex-col justify-start w-full px-10 py-5"
    >
      <form
        className="flex items-center w-full gap-4"
        onSubmit={(e) => {
          e.preventDefault();
          handleSearch();
        }}
      >
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
        <p className="text-gray-400 py-5 justify-self-end w-full text-right">
          Total Results: {data.total_results}
        </p>
      </form>

      <ul className="space-y-4 mt-6">
        {data.results.map((result, index) => (
          <li
            key={index}
            className="bg-white dark:bg-zinc-900 p-4 rounded-2xl shadow hover:shadow-md transition"
          >
            <a
              href={result.url}
              rel="noopener noreferrer"
              className="text-blue-600 dark:text-blue-400 hover:underline text-lg font-semibold"
            >
              {result.title}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default SearchClient;
