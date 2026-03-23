"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useAuth } from '@/lib/auth/AuthContext';
import { api } from '@/lib/api/client';
import ConsoleLayout from '@/components/ConsoleLayout';

const dict = {
  en: { title: "Dashboard Overview", totalDev: "Total Devices", activeSess: "Active Sessions", sysStatus: "System Status", healthy: "Healthy", recentAct: "Recent Activity", noAct: "No recent activity to display." },
  zh: { title: "仪表盘总览", totalDev: "设备总数", activeSess: "活跃会话", sysStatus: "系统状态", healthy: "健康", recentAct: "最近活动", noAct: "暂无最近活动。" }
};

function OverviewContent() {
  const { lang } = useLanguage();
  const { tenantId } = useAuth();
  const t = dict[lang];
  const [deviceCount, setDeviceCount] = useState<number | string>('...');
  const [sessionCount, setSessionCount] = useState<number | string>('...');

  useEffect(() => {
    if (!tenantId) return;

    api.get<{ items: any[] }>(`/tenants/${tenantId}/devices`)
      .then(data => setDeviceCount(data.items?.length ?? 0))
      .catch(() => setDeviceCount(0));

    api.get<{ items: any[] }>(`/tenants/${tenantId}/sessions`)
      .then(data => {
        const active = data.items?.filter((s: any) => !['completed', 'aborted', 'expired'].includes(s.status)) || [];
        setSessionCount(active.length);
      })
      .catch(() => setSessionCount(0));
  }, [tenantId]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.totalDev}</h3>
          <div className="mt-2">
            <span className="text-3xl font-semibold text-gray-900">{deviceCount}</span>
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.activeSess}</h3>
          <div className="mt-2">
            <span className="text-3xl font-semibold text-gray-900">{sessionCount}</span>
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.sysStatus}</h3>
          <div className="mt-2 flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="text-lg font-medium text-gray-900">{t.healthy}</span>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">{t.recentAct}</h2>
        </div>
        <div className="p-6">
          <p className="text-gray-500 text-sm">{t.noAct}</p>
        </div>
      </div>
    </div>
  );
}

export default function OverviewPage() {
  return (
    <ConsoleLayout>
      <OverviewContent />
    </ConsoleLayout>
  );
}
