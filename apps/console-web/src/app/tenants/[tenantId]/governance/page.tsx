"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Governance", empty: "No governance rules or drifts detected for tenant" },
  zh: { title: "环境治理", empty: "未检测到该租户的治理规则或配置漂移：" }
};

export default function GovernancePage({ params }: { params: { tenantId: string } }) {
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
