import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'EnvNexus Console',
  description: 'AI-native platform for environment governance',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
