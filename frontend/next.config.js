/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  reactStrictMode: true,
  experimental: {
    serverActions: {
      bodySizeLimit: "2mb",
    },
  },
  env: {
    NEXT_PUBLIC_API_BASE: process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080",
    NEXT_PUBLIC_NATS_WS_URL: process.env.NEXT_PUBLIC_NATS_WS_URL || "ws://localhost:9222",
  },
};

module.exports = nextConfig;
