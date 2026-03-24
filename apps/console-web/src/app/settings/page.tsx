"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';

export default function SettingsPage() {
  const { lang } = useLanguage();
  const t = useDict('settings', lang);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
        <p className="mb-2">{t.desc}</p>
        <p className="text-sm">{t.comingSoon}</p>
      </div>
    </div>
  );
}
