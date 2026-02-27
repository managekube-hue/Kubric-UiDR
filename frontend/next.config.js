/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  reactStrictMode: true,
  env: {
    NEXT_PUBLIC_API_BASE: process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080",
    NEXT_PUBLIC_KAI_URL: process.env.NEXT_PUBLIC_KAI_URL || "http://localhost:8100",
    NEXT_PUBLIC_NATS_WS_URL: process.env.NEXT_PUBLIC_NATS_WS_URL || "ws://localhost:9222",
  },
};

module.exports = nextConfig;
