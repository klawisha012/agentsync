/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    return [
      { source: "/api/:path*", destination: "http://api:8080/:path*" },
    ];
  },
};

module.exports = nextConfig;
