"use client";
import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { Search, Zap, Globe, Compass } from "lucide-react";
import SearchForm from "@/app/_components/SearchForm";
import { useAppContext } from "./_contexts/AppContext";

const Home = () => {
  const router = useRouter();

  const updateUrl = ({
    newQuery,
    newTab,
    newPage,
  }: {
    newQuery: string;
    newTab: string;
    newPage: number;
  }) => {
    router.push(
      `/search?q=${encodeURIComponent(newQuery)}&t=${newTab}&p=${newPage}`
    );
  };

  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const [showAnimations, setShowAnimations] = useState(false);
  const [particles, setParticles] = useState<
    {
      id: number;
      left: number;
      top: number;
      delay: number;
      duration: number;
    }[]
  >([]);

  const searchInputRef = useRef<HTMLInputElement>(null);
  const {
    isMobile,
    setIsMobile,
    loading,
    loadingImg,
    searchQuery,
    setSearchQuery,
  } = useAppContext();

  useEffect(() => {
    // Generate particles on client side only
    const generatedParticles = [...Array(20)].map((_, i) => ({
      id: i,
      left: Math.random() * 100,
      top: Math.random() * 100,
      delay: Math.random() * 3,
      duration: 2 + Math.random() * 2,
    }));
    setParticles(generatedParticles);

    // Trigger animations after component mounts
    const timer = setTimeout(() => {
      setShowAnimations(true);
    }, 100);

    const handleMouseMove = (e: MouseEvent) => {
      setMousePosition({ x: e.clientX, y: e.clientY });
    };

    window.addEventListener("mousemove", handleMouseMove);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      clearTimeout(timer);
    };
  }, []);

  // Mobile detection
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 640);
    };

    checkMobile();
    window.addEventListener("resize", checkMobile);

    return () => window.removeEventListener("resize", checkMobile);
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Focus search on Ctrl/Cmd + K
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <main className="relative flex flex-col items-center justify-center min-h-screen bg-gradient-to-br from-zinc-950 via-neutral-900 to-stone-900 px-4 overflow-hidden">
      {/* Animated background elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        {/* Floating particles */}
        {particles.map((particle) => (
          <div
            key={particle.id}
            className="absolute w-1 h-1 bg-emerald-400/20 rounded-full animate-pulse"
            style={{
              left: `${particle.left}%`,
              top: `${particle.top}%`,
              animationDelay: `${particle.delay}s`,
              animationDuration: `${particle.duration}s`,
            }}
          />
        ))}

        {/* Gradient orbs */}
        <div
          className="absolute w-96 h-96 bg-gradient-to-r from-emerald-500/10 to-teal-500/10 rounded-full blur-3xl transition-all duration-1000 ease-out"
          style={{
            left: mousePosition.x - 192,
            top: mousePosition.y - 192,
            transform: "translate(-50%, -50%)",
          }}
        />
        <div className="absolute top-1/4 left-1/4 w-72 h-72 bg-gradient-to-r from-lime-500/5 to-emerald-500/5 rounded-full blur-2xl animate-pulse" />
        <div
          className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-gradient-to-r from-teal-500/5 to-cyan-500/5 rounded-full blur-2xl animate-pulse"
          style={{ animationDelay: "1s" }}
        />
      </div>

      {/* Main content */}
      <div className="relative z-10 flex flex-col items-center w-full max-w-4xl">
        {/* Branding + Search section */}
        <div className="relative z-10 flex flex-col items-center w-full max-w-4xl">
          <div
            className={`transform transition-all duration-1000 ${
              showAnimations
                ? "translate-y-0 opacity-100"
                : "translate-y-8 opacity-0"
            }`}
          >
            <div className="relative flex flex-col items-center">
              <div className="relative">
                {/* Foreground text */}
                <h1
                  className="relative font-black text-transparent bg-clip-text bg-gradient-to-r from-white via-emerald-100 to-teal-300 text-center mb-2 tracking-tight
      text-[clamp(3.5rem,16vw,8rem)]"
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

          <SearchForm
            showAnimations={showAnimations}
            searchQuery={searchQuery}
            searchInputRef={searchInputRef}
            query=""
            isMobile={isMobile}
            setSearchQuery={setSearchQuery}
            loading={loading}
            loadingImg={loadingImg}
            tab="text"
            updateUrl={updateUrl}
          />
        </div>

        {/* Feature indicators */}
        <div
          className={`mt-12 flex flex-wrap items-center justify-center gap-8 transition-all duration-700 delay-400 ${
            showAnimations
              ? "translate-y-0 opacity-100"
              : "translate-y-8 opacity-0"
          }`}
        >
          <div className="flex items-center gap-2">
            <Globe className="w-4 h-4 text-emerald-400/70" />
            <span className="text-sm text-zinc-400">Web Search</span>
          </div>
          <div className="flex items-center gap-2">
            <Search className="w-4 h-4 text-teal-400/70" />
            <span className="text-sm text-zinc-400">Image Search</span>
          </div>
          <div className="flex items-center gap-2">
            <Zap className="w-4 h-4 text-amber-400/70" />
            <span className="text-sm text-zinc-400">ML Enhanced</span>
          </div>
        </div>

        {/* Bottom text */}
        <p
          className={`mt-8 text-zinc-400 text-center max-w-md transition-all duration-700 delay-500 flex items-center justify-center gap-1 ${
            showAnimations
              ? "translate-y-0 opacity-100"
              : "translate-y-8 opacity-0"
          }`}
        >
          <span>
            No{" "}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-teal-400 font-semibold">
              promises
            </span>
            , no filters — just the web as it chose to appear to you. &nbsp;
            <Compass className="w-4 h-4 text-emerald-400/70 inline-block" />
          </span>
        </p>
      </div>

      {/* Floating elements */}
      <div className="absolute top-20 left-20 w-2 h-2 bg-emerald-400/30 rounded-full animate-ping" />
      <div
        className="absolute bottom-20 right-20 w-3 h-3 bg-teal-400/30 rounded-full animate-pulse"
        style={{ animationDelay: "0.5s" }}
      />
      <div
        className="absolute top-1/3 right-16 w-1 h-1 bg-lime-400/40 rounded-full animate-pulse"
        style={{ animationDelay: "1s" }}
      />
    </main>
  );
};

export default Home;
