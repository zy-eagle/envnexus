import ConsoleLayout from '@/components/ConsoleLayout';

export default function OverviewLayout({ children }: { children: React.ReactNode }) {
  return (
    <ConsoleLayout>
      {children}
    </ConsoleLayout>
  );
}
