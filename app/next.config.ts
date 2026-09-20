import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Keep native ONNX binaries out of the Turbopack graph. Next 16 otherwise
  // bundles onnxruntime-node JS without libonnxruntime.so, which crashes
  // /api/search on Vercel.
  serverExternalPackages: ["@xenova/transformers", "onnxruntime-node"],
  outputFileTracingIncludes: {
    "/api/search": ["./node_modules/onnxruntime-node/**/*"],
    "/api/search-images": ["./node_modules/onnxruntime-node/**/*"],
  },
};

export default nextConfig;
