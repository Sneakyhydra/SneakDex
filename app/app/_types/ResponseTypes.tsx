export type Heading = { level: number; text: string };
export type ImageType = { src: string; alt?: string; title?: string };
export type SearchResponse = {
  source: string;
  results: Array<{
    id: string | number;
    hybridScore?: number;
    qdrantScore?: number;
    pgScore?: number;
    payload: {
      url?: string;
      title?: string;
      description?: string;
      headings?: Heading[];
      images?: ImageType[];
      language?: string;
      timestamp?: string;
      content_type?: string;
      text_snippet?: string;
      [key: string]: any;
    };
    url?: string;
    title?: string;
  }>;
  totalAvailable: {
    qdrant: number;
    postgres: number;
  };
  [key: string]: any;
};

export type SearchImgResponse = {
  source: string;
  results: Array<{
    id: string | number;
    score: number;
    payload: {
      src?: string;
      alt?: string;
      title?: string;
      caption?: string;
      page_url?: string;
      page_title?: string;
      page_description?: string;
      timestamp?: string;
      [key: string]: any;
    };
  }>;
  totalAvailable: {
    qdrant: number;
  };
  [key: string]: any;
};
