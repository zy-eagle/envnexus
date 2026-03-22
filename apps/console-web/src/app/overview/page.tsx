"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Dashboard Overview", totalDev: "Total Devices", activeSess: "Active Sessions", sysStatus: "System Status", healthy: "Healthy", recentAct: "Recent Activity", noAct: "No recent activity to display." },
  zh: { title: "仪表盘总览", totalDev: "设备总数", activeSess: "活跃会话", sysStatus: "系统状态", healthy: "健康", recentAct: "最近活动", noAct: "暂无最近活动。" }
};

export default function OverviewPage() {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [deviceCount, setDeviceCount] = useState<number | string>('...');

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const token = localStorage.getItem('token');
        // For MVP, we use the default tenant
        const res = await fetch('/api/v1/tenants/tenant_default/devices', {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
        if (res.ok) {
          const data = await res.json();
          setDeviceCount(data.data?.length || 0);
        } else {
          setDeviceCount(0);
        }
      } catch (error) {
        console.error('Failed to fetch device stats:', error);
        setDeviceCount(0);
      }
    };

    fetchStats();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Card 1 */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.totalDev}</h3>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-semibold text-gray-900">{deviceCount}</span>
            <span className="text-sm text-green-600 font-medium"></span>
          </div>
        </div>

        {/* Card 2 */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.activeSess}</h3>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-semibold text-gray-900">0</span>
          </div>
        </div>

        {/* Card 3 */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h3 className="text-sm font-medium text-gray-500">{t.sysStatus}</h3>
          <div className="mt-2 flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="text-lg font-medium text-gray-900">{t.healthy}</span>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">{t.recentAct}</h2>
        </div>
        <div className="p-6">
          <p className="text-gray-500 text-sm">{t.noAct}</p>
        </div>
      </div>
    </div>
  )
}
