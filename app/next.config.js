const { i18n } = require("./next-i18next.config");

/** @type {import('next').NextConfig} */
const nextConfig = {
  swcMinify: true,
  reactStrictMode: true,
  images: {
    domains: ["https://i.scdn.co/image"],
  },
  i18n,
  async rewrites() {
    return [
      {
        source: "/webapi/:path*",
        destination: "http://localhost:5555/:path*",
      },
    ];
  },
};

module.exports = nextConfig;
