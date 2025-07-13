import { Suspense } from "react";

export default function SearchLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Suspense
      fallback={
        <div className="text-center text-zinc-400 flex flex-col items-center gap-4 py-12">
          <div className="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
          <p>Loading…</p>
        </div>
      }
    >
      {children}
    </Suspense>
  );
}
