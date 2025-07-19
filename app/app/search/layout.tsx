import { Suspense } from "react";
import { Sparkles } from "lucide-react";

export default function SearchLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-zinc-950 via-slate-900 to-neutral-900">
      {/* Animated background elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-gradient-to-r from-emerald-500/8 via-teal-500/8 to-cyan-500/8 rounded-full blur-3xl animate-pulse opacity-60" />
        <div
          className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-gradient-to-r from-violet-500/8 via-purple-500/8 to-pink-500/8 rounded-full blur-3xl animate-pulse opacity-60"
          style={{ animationDelay: "1s" }}
        />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-transparent via-emerald-950/5 to-transparent" />
      </div>

      <Suspense
        fallback={
          <div className="relative z-10 min-h-screen flex items-center justify-center text-white">
            <div className="text-center flex flex-col items-center gap-6">
              {/* Modern loading spinner */}
              <div className="relative">
                <div className="w-16 h-16 border-4 border-emerald-400/30 border-t-emerald-400 rounded-full animate-spin" />
                <div className="absolute inset-0 w-16 h-16 border-4 border-emerald-400/10 rounded-full animate-ping" />
                <Sparkles className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-6 h-6 text-emerald-400 animate-pulse" />
              </div>

              {/* Loading text with gradient */}
              <div className="space-y-2">
                <h2 className="text-xl font-semibold bg-gradient-to-r from-white via-emerald-200 to-teal-400 bg-clip-text text-transparent">
                  Loading SneakDex
                </h2>
                <p className="text-sm text-zinc-400 max-w-md">
                  Preparing your AI-powered search experience
                </p>
              </div>

              {/* Animated dots */}
              <div className="flex gap-1">
                {[0, 1, 2].map((i) => (
                  <div
                    key={i}
                    className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse"
                    style={{
                      animationDelay: `${i * 0.2}s`,
                      animationDuration: "1s",
                    }}
                  />
                ))}
              </div>
            </div>
          </div>
        }
      >
        <div className="relative z-10">{children}</div>
      </Suspense>
    </div>
  );
}
