import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const nextConfig: NextConfig = {
  // Enable standalone output for Docker deployment
  output: 'standalone',

  // Disable x-powered-by header
  poweredByHeader: false,

  // Enable strict mode for better error detection
  reactStrictMode: true,
};

const withNextIntl = createNextIntlPlugin();

export default withNextIntl(nextConfig);
