/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Enable standalone output for smaller Docker images and faster builds
  output: 'standalone',
  // Disable source maps in production for faster builds
  productionBrowserSourceMaps: false,
  // Optimize image loading
  images: {
    unoptimized: true,
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://platform-api:8080/api/:path*', // Proxy to Backend
      },
    ]
  },
};

module.exports = nextConfig;
