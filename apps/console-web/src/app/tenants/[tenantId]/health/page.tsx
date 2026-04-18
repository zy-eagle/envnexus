"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface HealthSummary {
  tenant_id: string;
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  degraded_count: number;
  drift_count: number;
}

interface DeviceHealth {
  device_id: string;
  status: string;
  last_seen: string | null;
  agent_version: string;
}

export default function HealthDashboardPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('healthDashboard', lang);
  const ct = useDict('common', lang);
  const [summary, setSummary] = useState<HealthSummary | null>(null);
  const [devices, setDevices] = useState<DeviceHealth[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [sum, devs] = await Promise.all([
          api.get<HealthSummary>(`/tenants/${params.tenantId}/health/summary`),
          api.get<{ items: DeviceHealth[] }>(`/tenants/${params.tenantId}/health/devices`),
        ]);
        setSummary(sum);
        setDevices(Array.isArray(devs) ? devs : (devs as any)?.items || []);
      } catch {
        setSummary(null);
        setDevices([]);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [params.tenantId]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      {loading ? (
        <div className="p-8 text-center text-gray-500">{ct.loading}</div>
      ) : (
        <>
          {summary && (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                <div className="text-sm text-gray-500">{t.totalDevices}</div>
                <div className="text-2xl font-bold text-gray-900">{summary.total_devices}</div>
              </div>
              <div className="bg-white rounded-lg shadow-sm border border-green-200 p-4">
                <div className="text-sm text-green-600">{t.online}</div>
                <div className="text-2xl font-bold text-green-700">{summary.online_devices}</div>
              </div>
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                <div className="text-sm text-gray-500">{t.offline}</div>
                <div className="text-2xl font-bold text-gray-700">{summary.offline_devices}</div>
              </div>
              <div className="bg-white rounded-lg shadow-sm border border-orange-200 p-4">
                <div className="text-sm text-orange-600">{t.drifts}</div>
                <div className="text-2xl font-bold text-orange-700">{summary.drift_count}</div>
              </div>
            </div>
          )}

          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            {devices.length === 0 ? (
              <div className="p-8 text-center text-gray-500">{ct.noData}</div>
            ) : (
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceId}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.agentVersion}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.lastSeen}</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {devices.map(d => (
                    <tr key={d.device_id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{d.device_id}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                          d.status === 'online' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {d.status === 'online' ? t.online : t.offline}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.agent_version || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.last_seen ? new Date(d.last_seen).toLocaleString() : '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  );
}
