import ConsoleLayout from '@/components/ConsoleLayout';

export default function TenantsRootLayout({ children }: { children: React.ReactNode }) {
  return (
    <ConsoleLayout>
      {children}
    </ConsoleLayout>
  );
}
