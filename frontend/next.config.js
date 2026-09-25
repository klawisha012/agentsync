/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    const api = process.env.API_URL || "http://localhost:8080";
    return [
      { source: "/health", destination: `${api}/health` },
      { source: "/backend/:path*", destination: `${api}/:path*` },
    ];
  },
};

module.exports = nextConfig;
