/** @type {import('next').NextConfig} */
const nextConfig = {
  // VitePress builds the docs into public/docs. A redirect, not a rewrite: the VitePress router
  // needs the page path in the URL, and the browser keeps an old #section anchor across it.
  async redirects() {
    return [{ source: "/docs", destination: "/docs/index.html", permanent: false }];
  },
  async rewrites() {
    return [
      { source: "/api/:path*", destination: "http://api:8080/:path*" },
    ];
  },
};

module.exports = nextConfig;
