"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Platform Settings",
    desc: "Global platform configuration, identity providers, and system preferences.",
    comingSoon: "Settings module is under development."
  },
  zh: {
    title: "平台设置",
    desc: "全局平台配置、身份提供商接入和系统偏好设置。",
    comingSoon: "设置模块正在开发中。"
  }
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
        <p className="mb-2">{t.desc}</p>
        <p className="text-sm">{t.comingSoon}</p>
      </div>
    </div>
  );
}
