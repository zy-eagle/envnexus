/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
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
