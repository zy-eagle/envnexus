"use client";

import { useEffect, useState, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';


/** Agent default heartbeat is 30s; treat as offline after ~3 missed beats (avoid long "ghost" online after exit). */
const ONLINE_THRESHOLD_MS = 90 * 1000;
const POLL_INTERVAL_MS = 20 * 1000;
/** Re-evaluate online vs last_seen using local clock without waiting for the next list fetch. */
const PRESENCE_TICK_MS = 10 * 1000;

function isOnline(lastSeenAt: string | null, nowMs: number): boolean {
  if (!lastSeenAt) return false;
  return nowMs - new Date(lastSeenAt).getTime() < ONLINE_THRESHOLD_MS;
}

function DevicesContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = useDict('devices', lang);
  const ct = useDict('common', lang);
  const [devices, setDevices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [presenceNow, setPresenceNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setPresenceNow(Date.now()), PRESENCE_TICK_MS);
    return () => clearInterval(id);
  }, []);

  const fetchDevices = useCallback(async () => {
    try {
      const data = await api.get<{ items: any[] }>(`/tenants/${tenantId}/devices`);
      setDevices(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch devices:', error);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  const handleDelete = async (id: string) => {
    if (!confirm(t.revokeConfirm)) return;
    try {
      await api.delete(`/tenants/${tenantId}/devices/${id}`);
      fetchDevices();
    } catch (error) {
      console.error('Error deleting device:', error);
    }
  };

  useEffect(() => {
    fetchDevices();
    const interval = setInterval(fetchDevices, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchDevices]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'pending_activation': return 'bg-yellow-100 text-yellow-800';
      case 'quarantined': return 'bg-orange-100 text-orange-800';
      case 'revoked': case 'retired': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const onlineCount = devices.filter((d) => isOnline(d.last_seen_at, presenceNow)).length;
  const offlineCount = devices.length - onlineCount;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      {devices.length > 0 && (
        <div className="flex space-x-4">
          <div className="flex items-center space-x-2 px-3 py-1.5 bg-green-50 border border-green-200 rounded-md">
            <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
            <span className="text-sm text-green-800">{t.online}: {onlineCount}</span>
          </div>
          <div className="flex items-center space-x-2 px-3 py-1.5 bg-gray-50 border border-gray-200 rounded-md">
            <span className="w-2 h-2 bg-gray-400 rounded-full"></span>
            <span className="text-sm text-gray-600">{t.offline}: {offlineCount}</span>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4"></div>
            <p>{ct.loading}</p>
          </div>
        ) : devices.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noDevices}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase"></th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceName}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.hostname}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.platform}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.version}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.lastSeen}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {devices.map((d: any) => {
                  const online = isOnline(d.last_seen_at, presenceNow);
                  const presenceHint =
                    lang === "zh"
                      ? `最近心跳在 ${Math.round(ONLINE_THRESHOLD_MS / 1000)} 秒内视为在线`
                      : `Online if last heartbeat within ${Math.round(ONLINE_THRESHOLD_MS / 1000)}s`;
                  return (
                    <tr key={d.id} className="hover:bg-gray-50">
                      <td className="px-3 py-4 whitespace-nowrap">
                        <span
                          className={`inline-block w-2.5 h-2.5 rounded-full ${online ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`}
                          title={online ? `Online — ${presenceHint}` : `Offline — ${presenceHint}`}
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{d.device_name}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.hostname || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                          {d.platform}/{d.arch}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.distribution_package_version || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor(d.status)}`}>
                          {d.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {d.last_seen_at ? new Date(d.last_seen_at).toLocaleString() : 'Never'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button type="button" onClick={() => handleDelete(d.id)} className="text-red-600 hover:text-red-900">{t.revoke}</button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default function DevicesPage({ params }: { params: { tenantId: string } }) {
  return <DevicesContent tenantId={params.tenantId} />;
}
