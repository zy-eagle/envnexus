"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Platform Settings", empty: "Global platform settings go here." },
  zh: { title: "平台设置", empty: "全局平台设置将在这里显示。" }
};

export default function SettingsPage() {
  const { lang } = useLanguage();
  const t = dict[lang];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
        <p>{t.empty}</p>
      </div>
    </div>
  );
}
