"use client";
import { Search } from "lucide-react";

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

const SearchClient = ({ data }: { data: SearchResponse }) => {
  return (
    <div className="space-y-6 w-full">
      {/* Top bar: results count & sort */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
        <p className="text-zinc-400 text-sm">
          About {data.results.length} results
        </p>
      </div>

      {/* Search results */}
      <div className="space-y-6 w-full">
        {data.results.map((result, i) => {
          const payload = result.payload ?? {};
          const title = payload.title || payload.url || `Result ${i + 1}`;
          const url = payload.url || "#";
          const description =
            payload.description || payload.cleaned_text?.slice(0, 200) || null;

          return (
            <div
              key={result.id}
              className="border-b border-zinc-700/30 pb-6 last:border-b-0 h-40"
            >
              <div className="flex items-start gap-4 max-w-6xl">
                <div className="w-8 h-8 min-h-8 min-w-8 md:w-14 md:h-14 md:min-h-14 md:min-w-14 bg-gradient-to-br from-emerald-500/20 to-teal-500/20 rounded-lg flex items-center justify-center">
                  <Search className="w-4 h-4 md:w-8 md:h-8 text-emerald-400" />
                </div>
                <div className="flex flex-col flex-1 overflow-hidden">
                  <a
                    href={url}
                    className="text-lg font-semibold hover:text-zinc-200 mb-2 text-emerald-300 cursor-pointer block break-words"
                  >
                    {title}
                  </a>
                  <p className="text-zinc-400 text-xs mb-1 break-words overflow-hidden">
                    {url}
                  </p>
                  <p className="text-zinc-300 text-sm leading-relaxed line-clamp-2 break-words">
                    {description || "No Description Available"}
                  </p>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default SearchClient;
