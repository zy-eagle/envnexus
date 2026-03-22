"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Policy Profiles", addBtn: "Add Policy", empty: "No policy profiles configured for tenant" },
  zh: { title: "策略配置", addBtn: "添加策略", empty: "该租户尚未配置策略：" }
};

export default function PolicyProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors">
          {t.addBtn}
        </button>
      </div>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
        <p>{t.empty} {params.tenantId}.</p>
      </div>
    </div>
  );
}
