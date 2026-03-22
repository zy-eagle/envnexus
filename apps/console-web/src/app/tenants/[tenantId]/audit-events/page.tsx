"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Audit Events", empty: "No audit events found for tenant" },
  zh: { title: "审计事件", empty: "未找到该租户的审计事件：" }
};

export default function AuditEventsPage({ params }: { params: { tenantId: string } }) {
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
