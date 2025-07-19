"use client";

import { useState, useCallback, RefObject } from "react";
import { Search, Command } from "lucide-react";

const SearchForm = ({
  showAnimations,
  searchQuery,
  searchInputRef,
  query,
  isMobile,
  setSearchQuery,
  loading = false,
  loadingImg = false,
  tab,
  updateUrl,
}: {
  showAnimations: boolean;
  searchQuery: string;
  searchInputRef: RefObject<HTMLInputElement | null>;
  query: string;
  isMobile: boolean;
  setSearchQuery: (s: string) => void;
  loading: boolean;
  loadingImg: boolean;
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
  const [isSearchFocused, setIsSearchFocused] = useState(false);

  // Enhanced search handler with validation
  const handleSearch = useCallback(
    (e?: React.FormEvent) => {
      e?.preventDefault();
      const trimmedQuery = searchQuery.trim();

      if (!trimmedQuery) {
        searchInputRef.current?.focus();
        return;
      }

      if (trimmedQuery !== query) {
        updateUrl({ newQuery: trimmedQuery, newTab: tab, newPage: 1 });
      }
    },
    [searchQuery, query]
  );

  return (
    <div
      className={`w-full max-w-2xl transition-all duration-700 delay-200 ${
        showAnimations ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"
      }`}
    >
      <form onSubmit={handleSearch} className="relative">
        <div
          className={`flex items-center bg-white/5 backdrop-blur-xl border rounded-2xl transition-all duration-300 overflow-hidden ${
            isSearchFocused
              ? "border-emerald-400/60 shadow-2xl shadow-emerald-500/10 bg-white/8 ring-2 ring-emerald-400/20"
              : "border-white/10 hover:border-white/20 hover:bg-white/8"
          }`}
        >
          {/* Enhanced Logo - Hidden on very small screens */}
          <div
            className={`hidden sm:flex items-center px-4 transition-all duration-500 ${
              showAnimations
                ? "opacity-100 translate-x-0"
                : "opacity-0 -translate-x-4"
            }`}
          >
            <img
              src="/favicon.ico"
              alt="SneakDex Logo"
              className="w-8 h-8 transition-transform duration-300 hover:scale-110 hover:drop-shadow-[0_0_8px_#10b981]"
            />
          </div>

          {/* Enhanced Input - Responsive padding */}
          <div className="flex-1 flex items-center">
            <input
              ref={searchInputRef}
              type="text"
              placeholder={
                isMobile ? "Search anything…" : "Search for anything…"
              }
              className="w-full px-3 sm:px-4 sm:py-4 bg-transparent text-white placeholder-zinc-400 focus:outline-none text-base transition-all duration-200 focus:placeholder-zinc-500"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onFocus={() => setIsSearchFocused(true)}
              onBlur={() => setIsSearchFocused(false)}
              disabled={loading || loadingImg}
            />
          </div>

          {/* Enhanced Button - Responsive design */}
          <div className="flex-shrink-0 p-2">
            <button
              type="submit"
              disabled={loading || loadingImg || !searchQuery.trim()}
              className="group relative px-4 sm:px-6 py-2.5 text-sm text-white rounded-xl font-semibold transition-all duration-300 hover:scale-105 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 flex items-center gap-2 overflow-hidden"
              style={{
                background:
                  "linear-gradient(135deg, #059669, #10b981, #34d399)",
              }}
            >
              {/* Animated background */}
              <div className="absolute inset-0 bg-gradient-to-r from-emerald-400/20 to-teal-400/20 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

              {/* Button content */}
              <div className="relative flex items-center gap-2">
                {loading || loadingImg ? (
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                ) : (
                  <Search className="w-4 h-4" />
                )}
                <span className="hidden sm:inline">Search</span>
              </div>
            </button>
          </div>
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
              + <span className="font-mono text-zinc-500">K</span> to focus
            </span>
          </div>
        )}
      </form>
    </div>
  );
};

export default SearchForm;
