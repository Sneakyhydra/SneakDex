"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";

const Home = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const router = useRouter();

  const handleSearch = () => {
    const trimmedQuery = searchQuery.trim();
    if (trimmedQuery === "") {
      return;
    }

    setSearchQuery("");
    router.push(`/search?q=${encodeURIComponent(trimmedQuery)}`);
  };

  return (
    <form
      className="flex flex-col items-center justify-center h-screen w-full"
      style={{ backgroundColor: "#202124" }}
      onSubmit={(e) => {
        e.preventDefault();
        handleSearch();
      }}
    >
      <h1 className="text-8xl font-bold mb-9 text-white">SneakDex</h1>
      <input
        type="text"
        placeholder="Search"
        className="w-full max-w-xl p-3 border border-gray-300 rounded-full mb-7"
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
      />
      <button
        type="submit"
        className="text-white px-4 py-2 cursor-pointer transition-colors duration-300 border rounded-sm border-neutral-700 hover:border-white bg-neutral-700"
      >
        Search
      </button>
    </form>
  );
};

export default Home;
