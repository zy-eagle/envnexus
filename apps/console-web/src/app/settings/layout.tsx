import ConsoleLayout from '@/components/ConsoleLayout';

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return (
    <ConsoleLayout>
      {children}
    </ConsoleLayout>
  );
}
