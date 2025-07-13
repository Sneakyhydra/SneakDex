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
    <div className="space-y-6">
      {/* Top bar: results count & sort */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
        <p className="text-zinc-400 text-sm">
          About {data.results.length} results
        </p>
      </div>

      {/* Search results */}
      <div className="space-y-6">
        {data.results.map((result, i) => {
          const payload = result.payload ?? {};
          const title = payload.title || payload.url || `Result ${i + 1}`;
          const url = payload.url || "#";
          const description =
            payload.description || payload.cleaned_text?.slice(0, 200) || null;
          const score =
            typeof result.score === "number" ? result.score.toFixed(3) : null;

          return (
            <div
              key={result.id}
              className="border-b border-zinc-700/30 pb-6 last:border-b-0"
            >
              <div className="flex items-start gap-4">
                <div className="w-12 h-12 bg-gradient-to-br from-emerald-500/20 to-teal-500/20 rounded-lg flex items-center justify-center">
                  <Search className="w-5 h-5 text-emerald-400" />
                </div>
                <div className="flex-1">
                  <a
                    href={url}
                    className="text-lg font-semibold text-zinc-200 mb-2 hover:text-emerald-400 cursor-pointer block"
                  >
                    {title}
                  </a>
                  <p className="text-zinc-400 text-xs mb-1 break-words">
                    {url} {score && `• score: ${score}`}
                  </p>
                  {description && (
                    <p className="text-zinc-300 text-sm leading-relaxed">
                      {description}…
                    </p>
                  )}
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
