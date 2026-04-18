"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useAuth } from '@/lib/auth/AuthContext';
import { api } from '@/lib/api/client';

const dict = {
  en: { title: "Dashboard", subtitle: "Overview of your environment", totalDev: "Total Devices", activeSess: "Active Sessions", sysStatus: "System Status", healthy: "All systems operational", recentAct: "Recent Activity", noAct: "No recent activity to display." },
  zh: { title: "仪表盘", subtitle: "环境概览", totalDev: "设备总数", activeSess: "活跃会话", sysStatus: "系统状态", healthy: "所有系统运行正常", recentAct: "最近活动", noAct: "暂无最近活动。" }
};

function StatCard({ label, value, icon, color }: { label: string; value: string | number; icon: JSX.Element; color: string }) {
  return (
    <div className="card p-5">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-slate-500">{label}</p>
          <p className="mt-2 text-3xl font-semibold text-slate-900 tracking-tight">{value}</p>
        </div>
        <div className={`w-10 h-10 rounded-xl ${color} flex items-center justify-center`}>
          {icon}
        </div>
      </div>
    </div>
  );
}

export default function OverviewPage() {
  const { lang } = useLanguage();
  const { activeTenantId } = useAuth();
  const t = dict[lang];
  const [deviceCount, setDeviceCount] = useState<number | string>('...');
  const [sessionCount, setSessionCount] = useState<number | string>('...');
  const [recentActivities, setRecentActivities] = useState<any[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [pageSize, setPageSize] = useState(6);

  useEffect(() => {
    if (!activeTenantId) return;

    // 获取设备总数
    api.get<{ items: any[] }>(`/tenants/${activeTenantId}/devices`)
      .then(data => setDeviceCount(Array.isArray(data) ? data.length : 0))
      .catch(() => setDeviceCount(0));

    // 获取活跃会话数量
    api.get<{ count: number }>(`/tenants/${activeTenantId}/sessions/active-count`)
      .then(data => setSessionCount(data.count))
      .catch(() => setSessionCount(0));

    // 获取最近活动（审计日志）
    api.get<{ items: any[]; total: number; page: number; page_size: number }>(`/tenants/${activeTenantId}/audit-events?page=${currentPage}&page_size=${pageSize}`)
      .then(data => {
        setRecentActivities(Array.isArray(data.items) ? data.items : []);
        setTotalPages(Math.ceil((data.total || 0) / pageSize));
      })
      .catch(() => setRecentActivities([]));
  }, [activeTenantId, currentPage, pageSize]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900 tracking-tight">{t.title}</h1>
        <p className="mt-1 text-sm text-slate-500">{t.subtitle}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard
          label={t.totalDev}
          value={deviceCount}
          color="bg-blue-50"
          icon={<svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25A2.25 2.25 0 0 1 5.25 3h13.5A2.25 2.25 0 0 1 21 5.25Z" /></svg>}
        />
        <StatCard
          label={t.activeSess}
          value={sessionCount}
          color="bg-emerald-50"
          icon={<svg className="w-5 h-5 text-emerald-600" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z" /></svg>}
        />
        <StatCard
          label={t.sysStatus}
          value={t.healthy}
          color="bg-emerald-50"
          icon={<svg className="w-5 h-5 text-emerald-600" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" /></svg>}
        />
      </div>

      {/* 最近活动 */}
      <div className="card overflow-hidden">
        <div className="px-5 py-4 border-b border-slate-100">
          <h2 className="text-sm font-semibold text-slate-900">{t.recentAct}</h2>
        </div>
        {recentActivities.length > 0 ? (
          <div className="divide-y divide-slate-100">
            {recentActivities.map((activity, index) => (
              <div key={index} className="px-5 py-4">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-medium text-slate-900">{activity.event_type || 'Unknown Event'}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {activity.created_at ? new Date(activity.created_at).toLocaleString() : 'Unknown Time'}
                    </p>
                  </div>
                  <span className="text-xs text-slate-400">
                    {activity.device_id ? `Device: ${activity.device_id.substring(0, 8)}...` : 'System'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="px-5 py-12 text-center">
            <svg className="w-10 h-10 text-slate-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" strokeWidth={1} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" /></svg>
            <p className="text-sm text-slate-400">{t.noAct}</p>
          </div>
        )}
        {/* 分页控制 */}
        {recentActivities.length > 0 && totalPages > 1 && (
          <div className="px-5 py-4 border-t border-slate-100">
            <div className="flex justify-between items-center">
              <div className="flex items-center space-x-4">
                <div className="text-sm text-slate-500">
                  共 {totalPages * pageSize} 条记录
                </div>
                <div className="flex items-center space-x-2">
                  <span className="text-sm text-slate-500">每页显示：</span>
                  <select 
                    value={pageSize}
                    onChange={(e) => {
                      setPageSize(parseInt(e.target.value));
                      setCurrentPage(1);
                    }}
                    className="border rounded-md px-2 py-1 text-sm"
                  >
                    <option value="6">6条</option>
                    <option value="10">10条</option>
                    <option value="20">20条</option>
                    <option value="50">50条</option>
                  </select>
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <button 
                  onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                  disabled={currentPage === 1}
                  className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                >
                  上一页
                </button>
                <button 
                  onClick={() => setCurrentPage(prev => Math.min(prev + 1, totalPages))}
                  disabled={currentPage === totalPages}
                  className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                >
                  下一页
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
