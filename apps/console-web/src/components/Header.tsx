"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useAuth } from '@/lib/auth/AuthContext';
import { useDict } from '@/lib/i18n/dictionary';

export default function Header() {
  const { lang, setLang } = useLanguage();
  const {
    user,
    tenantId,
    activeTenantId,
    activeTenantName,
    myRolesInTenant,
    logout,
  } = useAuth();
  const tHeader = useDict('header', lang);
  const tNav = useDict('nav', lang);
  const [userInitial, setUserInitial] = useState('A');
  const [accountOpen, setAccountOpen] = useState(false);

  useEffect(() => {
    if (user?.display_name || user?.email) {
      const name = user.display_name || user.email;
      setUserInitial(name.charAt(0).toUpperCase());
      return;
    }
    const userStr = localStorage.getItem('user');
    if (userStr) {
      try {
        const u = JSON.parse(userStr);
        if (u.display_name || u.username) {
          const name = u.display_name || u.username;
          setUserInitial(name.charAt(0).toUpperCase());
        }
      } catch {
        // ignore
      }
    }
  }, [user]);

  const displayName = user?.display_name || user?.email || tHeader.admin;
  const email = user?.email || '';

  return (
    <header className="h-14 bg-white/80 backdrop-blur-md border-b border-slate-200/60 flex items-center justify-between px-6 sticky top-0 z-30">
      <div className="flex items-center gap-2">
        <h1 className="text-sm font-medium text-slate-500">{tHeader.console}</h1>
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
            type="button"
            onClick={() => setLang('en')}
            className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
              lang === 'en' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'
            }`}
          >
            EN
          </button>
          <button
            type="button"
            onClick={() => setLang('zh')}
            className={`px-2.5 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
              lang === 'zh' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'
            }`}
          >
            中文
          </button>
        </div>
        <div className="h-6 w-px bg-slate-200" />
        <div className="relative">
          <button
            type="button"
            onClick={() => setAccountOpen((o) => !o)}
            className="flex items-center gap-2 rounded-lg pl-1 pr-2 py-1 hover:bg-slate-100 transition-colors"
            aria-expanded={accountOpen}
            aria-haspopup="menu"
          >
            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-400 to-indigo-600 flex items-center justify-center text-white text-xs font-semibold shadow-sm">
              {userInitial}
            </div>
            <span className="text-sm font-medium text-slate-700 max-w-[140px] truncate hidden sm:inline">
              {displayName}
            </span>
            <svg
              className={`w-4 h-4 text-slate-400 transition-transform ${accountOpen ? 'rotate-180' : ''}`}
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
            </svg>
          </button>
          {accountOpen && user && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setAccountOpen(false)} aria-hidden />
              <div
                className="absolute right-0 top-full mt-1 z-50 w-72 rounded-xl border border-slate-200 bg-white shadow-lg py-3 px-3 text-left"
                role="menu"
              >
                <div className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1.5">
                  {tNav.myRoles || 'Your roles'}
                </div>
                <div className="text-sm font-semibold text-slate-900 truncate" title={email}>
                  {displayName}
                </div>
                {email ? (
                  <div className="text-xs text-slate-500 truncate mt-0.5" title={email}>
                    {email}
                  </div>
                ) : null}
                <div className="mt-2 flex flex-wrap gap-1">
                  {user.platform_super_admin && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded-md text-[11px] font-medium bg-amber-50 text-amber-800 border border-amber-200">
                      {tNav.platformSuperAdmin || 'Platform super admin'}
                    </span>
                  )}
                  {myRolesInTenant.map((r) => (
                    <span
                      key={r.id}
                      className="inline-flex items-center px-2 py-0.5 rounded-md text-[11px] font-medium bg-indigo-50 text-indigo-700 border border-indigo-100"
                    >
                      {r.name}
                    </span>
                  ))}
                </div>
                {!user.platform_super_admin && myRolesInTenant.length === 0 && (
                  <p className="text-[11px] text-slate-400 mt-2 leading-snug line-clamp-3">
                    {activeTenantId !== tenantId
                      ? lang === 'zh'
                        ? '已切换租户：角色信息仅随登录租户显示。'
                        : 'Switched tenant: roles reflect your login tenant only.'
                      : lang === 'zh'
                        ? '当前账号在本租户未绑定角色。'
                        : 'No roles bound in this tenant.'}
                  </p>
                )}
                {user.platform_super_admin && myRolesInTenant.length === 0 && (
                  <p className="text-[11px] text-slate-400 mt-2 leading-snug line-clamp-2">
                    {lang === 'zh'
                      ? '本租户未绑定具体角色（仍具备全局审批等超管能力）。'
                      : 'No role bindings in this tenant (global approvals still apply).'}
                  </p>
                )}
                <div className="mt-3 pt-3 border-t border-slate-100">
                  <button
                    type="button"
                    onClick={() => {
                      setAccountOpen(false);
                      logout();
                    }}
                    className="w-full flex items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-slate-600 hover:bg-red-50 hover:text-red-600 transition-colors"
                    role="menuitem"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m3 0 3-3m0 0-3-3m3 3H9" />
                    </svg>
                    {tNav.signOut}
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
