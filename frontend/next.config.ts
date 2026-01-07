import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable standalone output for Docker deployment
  output: 'standalone',

  // Disable x-powered-by header
  poweredByHeader: false,

  // Enable strict mode for better error detection
  reactStrictMode: true,
};

export default nextConfig;
