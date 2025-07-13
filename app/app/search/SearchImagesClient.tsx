"use client";
import { Image as LucideImage } from "lucide-react";

type SearchResponse = {
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

const SearchImagesClient = ({ data }: { data: SearchResponse }) => {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-4 justify-center">
        {data.results.map((result) => {
          const p = result.payload;

          return (
            <a
              key={result.id}
              href={p.page_url || "#"}
              className="group cursor-pointer flex-shrink-0"
              style={{
                minWidth: "150px",
                maxWidth: "300px",
              }}
            >
              <div
                className="
            bg-gradient-to-br from-emerald-500/10 to-teal-500/10
            rounded-lg border border-zinc-700/50
            hover:border-emerald-400/50 transition-all duration-300
            group-hover:scale-105 flex items-center justify-center
            overflow-hidden
          "
                style={{
                  minWidth: "150px",
                  maxWidth: "300px",
                  minHeight: "150px",
                  maxHeight: "300px",
                }}
              >
                {p.src ? (
                  <img
                    src={p.src}
                    alt={p.alt || p.caption || ""}
                    className="
                object-contain
                max-w-full max-h-full
                m-auto
              "
                  />
                ) : (
                  <LucideImage className="w-8 h-8 text-emerald-400/50" />
                )}
              </div>
              <p className="text-zinc-400 text-xs mt-2 truncate text-center">
                {p.alt || p.title || ""}
              </p>
            </a>
          );
        })}
      </div>
    </div>
  );
};

export default SearchImagesClient;
