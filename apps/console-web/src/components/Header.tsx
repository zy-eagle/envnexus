"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useAuth } from '@/lib/auth/AuthContext';
import { useDict } from '@/lib/i18n/dictionary';

export default function Header() {
  const { lang, setLang } = useLanguage();
  const { activeTenantName } = useAuth();
  const t = useDict('header', lang);
  const [userName, setUserName] = useState(t.admin);
  const [userInitial, setUserInitial] = useState('A');

  useEffect(() => {
    const userStr = localStorage.getItem('user');
    if (userStr) {
      try {
        const user = JSON.parse(userStr);
        if (user.display_name || user.username) {
          const name = user.display_name || user.username;
          setUserName(name);
          setUserInitial(name.charAt(0).toUpperCase());
        }
      } catch {
        // ignore
      }
    }
  }, []);

  return (
    <header className="h-14 bg-white/80 backdrop-blur-md border-b border-slate-200/60 flex items-center justify-between px-6 sticky top-0 z-30">
      <div className="flex items-center gap-2">
        <h1 className="text-sm font-medium text-slate-500">{t.console}</h1>
        {activeTenantName && (
          <>
            <svg className="w-4 h-4 text-slate-300" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" /></svg>
            <span className="text-sm font-semibold text-slate-900">{activeTenantName}</span>
          </>
        )}
      </div>
      <div className="flex items-center gap-3">
        <div className="flex items-center bg-slate-100 rounded-lg p-0.5">
          <button
            onClick={() => setLang('en')}
            className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
              lang === 'en' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'
            }`}
          >
            EN
          </button>
          <button
            onClick={() => setLang('zh')}
            className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
              lang === 'zh' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'
            }`}
          >
            中文
          </button>
        </div>
        <div className="h-6 w-px bg-slate-200" />
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-400 to-indigo-600 flex items-center justify-center text-white text-xs font-semibold shadow-sm">
            {userInitial}
          </div>
          <span className="text-sm font-medium text-slate-700">{userName}</span>
        </div>
      </div>
    </header>
  );
}
