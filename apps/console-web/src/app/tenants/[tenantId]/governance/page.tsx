"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Governance",
    desc: "Environment governance and configuration drift detection will be displayed here.",
    comingSoon: "Coming Soon"
  },
  zh: {
    title: "环境治理",
    desc: "环境治理基线与配置漂移检测结果将在此处展示。",
    comingSoon: "敬请期待"
  }
};

export default function GovernancePage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-100 text-blue-600 mb-4">
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-2">{t.comingSoon}</h3>
        <p className="text-gray-500 max-w-md mx-auto">
          {t.desc}
        </p>
      </div>
    </div>
  );
}
