"use client";

import { useState } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const translations = {
  en: {
    dashboard: "Dashboard Overview",
    tenantMgmt: "Tenant Management",
    allTenants: "All Tenants",
    devices: "Devices",
    sessions: "Sessions",
    auditEvents: "Audit Events",
    downloadPackages: "Download Packages",
    modelProfiles: "Model Profiles",
    policyProfiles: "Policy Profiles",
    agentProfiles: "Agent Profiles",
    governance: "Governance",
    platform: "Platform",
    settings: "Settings",
    signOut: "Sign Out"
  },
  zh: {
    dashboard: "仪表盘总览",
    tenantMgmt: "租户管理",
    allTenants: "所有租户",
    devices: "设备管理",
    sessions: "会话管理",
    auditEvents: "审计事件",
    downloadPackages: "下载包管理",
    modelProfiles: "模型配置",
    policyProfiles: "策略配置",
    agentProfiles: "Agent 配置",
    governance: "环境治理",
    platform: "平台设置",
    settings: "系统设置",
    signOut: "退出登录"
  }
};

export default function Sidebar() {
  const { lang } = useLanguage();
  const [expanded, setExpanded] = useState({
    tenant: true,
    platform: true
  });

  const t = translations[lang];

  const toggle = (key: keyof typeof expanded) => {
    setExpanded(prev => ({ ...prev, [key]: !prev[key] }));
  };

  return (
    <aside className="w-64 bg-white border-r border-gray-200 flex flex-col">
      <div className="h-16 flex items-center px-6 border-b border-gray-200">
        <h1 className="text-xl font-bold text-blue-600">EnvNexus</h1>
      </div>
      <nav className="flex-1 overflow-y-auto py-4">
        <ul className="space-y-1 px-3">
          <li>
            <Link href="/overview" className="flex items-center px-3 py-2 text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">
              {t.dashboard}
            </Link>
          </li>

          {/* Tenant Management */}
          <li>
            <button 
              onClick={() => toggle('tenant')}
              className="w-full flex items-center justify-between px-3 py-2 mt-4 text-xs font-semibold text-gray-500 uppercase tracking-wider hover:bg-gray-50 rounded-md transition-colors"
            >
              <span>{t.tenantMgmt}</span>
              <span className="text-gray-400 text-[10px]">{expanded.tenant ? '▼' : '▶'}</span>
            </button>
            {expanded.tenant && (
              <ul className="mt-2 space-y-1 pl-4 border-l border-gray-100 ml-3">
                <li><Link href="/tenants" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.allTenants}</Link></li>
                <li><Link href="/tenants/tenant_default/devices" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.devices}</Link></li>
                <li><Link href="/tenants/tenant_default/sessions" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.sessions}</Link></li>
                <li><Link href="/tenants/tenant_default/audit-events" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.auditEvents}</Link></li>
                <li><Link href="/tenants/tenant_default/download-packages" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.downloadPackages}</Link></li>
                <li><Link href="/tenants/tenant_default/model-profiles" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.modelProfiles}</Link></li>
                <li><Link href="/tenants/tenant_default/policy-profiles" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.policyProfiles}</Link></li>
                <li><Link href="/tenants/tenant_default/agent-profiles" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.agentProfiles}</Link></li>
                <li><Link href="/tenants/tenant_default/governance" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.governance}</Link></li>
              </ul>
            )}
          </li>

          {/* Platform */}
          <li>
            <button 
              onClick={() => toggle('platform')}
              className="w-full flex items-center justify-between px-3 py-2 mt-4 text-xs font-semibold text-gray-500 uppercase tracking-wider hover:bg-gray-50 rounded-md transition-colors"
            >
              <span>{t.platform}</span>
              <span className="text-gray-400 text-[10px]">{expanded.platform ? '▼' : '▶'}</span>
            </button>
            {expanded.platform && (
              <ul className="mt-2 space-y-1 pl-4 border-l border-gray-100 ml-3">
                <li><Link href="/settings" className="block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600">{t.settings}</Link></li>
                <li>
                  <button 
                    onClick={() => {
                      localStorage.removeItem('token');
                      localStorage.removeItem('user');
                      window.location.href = '/login';
                    }}
                    className="w-full text-left block px-3 py-2 text-sm text-gray-700 rounded-md hover:bg-gray-100 hover:text-blue-600"
                  >
                    {t.signOut}
                  </button>
                </li>
              </ul>
            )}
          </li>
        </ul>
      </nav>
    </aside>
  );
}
