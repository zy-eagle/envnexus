"use client";

import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { console: "Console", admin: "Admin User" },
  zh: { console: "控制台", admin: "管理员" }
};

export default function Header() {
  const { lang, setLang } = useLanguage();
  const t = dict[lang];

  return (
    <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
      <div className="text-lg font-medium text-gray-800">{t.console}</div>
      <div className="flex items-center space-x-4">
        <select 
          value={lang} 
          onChange={(e) => setLang(e.target.value as 'en' | 'zh')}
          className="text-sm border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 py-1 px-2"
        >
          <option value="en">English</option>
          <option value="zh">中文</option>
        </select>
        <div className="text-sm text-gray-500">{t.admin}</div>
        <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center text-white font-bold">
          A
        </div>
      </div>
    </header>
  );
}
