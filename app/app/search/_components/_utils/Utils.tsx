import { useEffect, useState } from "react";
import { Search } from "lucide-react";

export const formatTimestamp = (isoString: string): string => {
  try {
    const date = new Date(isoString);
    return date.toLocaleString(undefined, {
      dateStyle: "medium",
    });
  } catch {
    return isoString;
  }
};

export const urlToTitle = (urlStr: string): string => {
  try {
    const url = new URL(urlStr);
    const hostname = url.hostname.replace(/^www\./, "");
    const domainParts = hostname.split(".");
    const domain =
      domainParts.length > 1 ? domainParts[domainParts.length - 2] : hostname;
    const domainTitle = capitalize(domain);

    const pathSegments = url.pathname
      .split("/")
      .filter(Boolean)
      .map(decodeURIComponent)
      .map((s) => s.replace(/[-_]/g, " "))
      .filter(Boolean);

    let lastSegment = pathSegments[pathSegments.length - 1] || "";
    lastSegment = capitalize(lastSegment);

    if (lastSegment.toLowerCase() === domain.toLowerCase() || !lastSegment) {
      return domainTitle;
    }

    return `${lastSegment} | ${domainTitle}`;
  } catch {
    return "Untitled";
  }
};

export const capitalize = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
};

export const getDomainFromUrl = (url: string): string => {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
};

export const isValidImageSize = (src: string): Promise<boolean> => {
  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => {
      // Consider images smaller than 32x32 as invalid placeholders
      resolve(img.width >= 32 && img.height >= 32);
    };
    img.onerror = () => resolve(false);
    img.src = src;
  });
};

export const ThumbnailImage = ({ result }: { result: any }) => {
  const [showPlaceholder, setShowPlaceholder] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  useEffect(() => {
    if (result.thumbnail?.src) {
      isValidImageSize(result.thumbnail.src).then((isValid) => {
        if (!isValid) {
          setShowPlaceholder(true);
        }
      });
    } else {
      setShowPlaceholder(true);
    }
  }, [result.thumbnail?.src]);

  const handleImageError = () => {
    setShowPlaceholder(true);
  };

  const handleImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget;
    if (img.naturalWidth < 32 || img.naturalHeight < 32) {
      setShowPlaceholder(true);
    } else {
      setImageLoaded(true);
    }
  };

  if (showPlaceholder || !result.thumbnail?.src) {
    return (
      <div className="w-16 h-16 min-w-16 min-h-16 bg-gradient-to-br from-emerald-500/10 to-teal-500/10 border border-emerald-500/20 rounded-xl flex items-center justify-center">
        <Search className="w-6 h-6 text-emerald-400/70" />
      </div>
    );
  }

  return (
    <div className="w-16 h-16 min-w-16 min-h-16 rounded-xl overflow-hidden bg-zinc-800 border border-zinc-700/50">
      <img
        src={result.thumbnail.src}
        alt={result.thumbnail.alt || `Thumbnail for ${result.title}`}
        className={`w-full h-full object-cover transition-opacity duration-200 ${
          imageLoaded ? "opacity-100" : "opacity-0"
        }`}
        onError={handleImageError}
        onLoad={handleImageLoad}
      />
      {!imageLoaded && !showPlaceholder && (
        <div className="w-full h-full bg-zinc-800 animate-pulse" />
      )}
    </div>
  );
};
