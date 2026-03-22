"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Sessions", empty: "No active sessions found for tenant" },
  zh: { title: "会话管理", empty: "未找到该租户的活跃会话：" }
};

export default function SessionsPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
        <p>{t.empty} {params.tenantId}.</p>
      </div>
    </div>
  );
}
