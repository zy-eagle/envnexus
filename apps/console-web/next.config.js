/** @type {import('next').NextConfig} */
const apiTarget = process.env.API_PROXY_TARGET || 'http://platform-api:8080';

const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  productionBrowserSourceMaps: false,
  images: {
    unoptimized: true,
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${apiTarget}/api/:path*`,
      },
    ]
  },
};

module.exports = nextConfig;
