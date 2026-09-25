/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    const api = process.env.API_URL || "http://localhost:8080";
    return [
      { source: "/health", destination: `${api}/health` },
      { source: "/backend/:path*", destination: `${api}/:path*` },
      { source: "/agent-bridge/:path*", destination: "http://127.0.0.1:49152/:path*" },
    ];
  },
};

module.exports = nextConfig;
