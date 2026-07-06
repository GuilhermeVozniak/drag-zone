import type { NextConfig } from "next";

// Static export for GitHub Pages. The project site is served under /drag-zone,
// so the deploy workflow sets NEXT_PUBLIC_BASE_PATH=/drag-zone; a custom domain
// would set it empty.
const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  basePath: process.env.NEXT_PUBLIC_BASE_PATH ?? "",
};

export default nextConfig;
