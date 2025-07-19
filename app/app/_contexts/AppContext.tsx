"use client";

import { createContext, useContext, useState, ReactNode } from "react";
import { SearchImgResponse, SearchResponse } from "../_types/ResponseTypes";

// define the shape of your context
type AppContextType = {
  data: SearchResponse | null;
  setData: (d: SearchResponse | null) => void;
  dataQuery: string | null;
  setDataQuery: (s: string | null) => void;
  imgData: SearchImgResponse | null;
  setImgData: (d: SearchImgResponse | null) => void;
  imgDataQuery: string | null;
  setImgDataQuery: (s: string | null) => void;
  loading: boolean;
  setLoading: (b: boolean) => void;
  loadingImg: boolean;
  setLoadingImg: (b: boolean) => void;
  isMobile: boolean;
  setIsMobile: (b: boolean) => void;
  searchQuery: string;
  setSearchQuery: (s: string) => void;
};

// initialize context
const AppContext = createContext<AppContextType | undefined>(undefined);

// context provider component
export function AppProvider({ children }: { children: ReactNode }) {
  const [data, setData] = useState<SearchResponse | null>(null);
  const [dataQuery, setDataQuery] = useState<string | null>(null);
  const [imgData, setImgData] = useState<SearchImgResponse | null>(null);
  const [imgDataQuery, setImgDataQuery] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [loadingImg, setLoadingImg] = useState<boolean>(false);
  const [isMobile, setIsMobile] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>("");

  return (
    <AppContext.Provider
      value={{
        data,
        setData,
        dataQuery,
        setDataQuery,
        imgData,
        setImgData,
        imgDataQuery,
        setImgDataQuery,
        loading,
        setLoading,
        loadingImg,
        setLoadingImg,
        isMobile,
        setIsMobile,
        searchQuery,
        setSearchQuery,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}

// custom hook for consuming context
export function useAppContext() {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error("useAppContext must be used inside <AppProvider>");
  }
  return context;
}
