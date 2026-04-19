"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';

const ONLINE_THRESHOLD_MS = 90 * 1000;
const POLL_INTERVAL_MS = 20 * 1000;
const PRESENCE_TICK_MS = 10 * 1000;

function isOnline(lastSeenAt: string | null, nowMs: number): boolean {
  if (!lastSeenAt) return false;
  return nowMs - new Date(lastSeenAt).getTime() < ONLINE_THRESHOLD_MS;
}

interface HealthSummary {
  tenant_id: string;
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  degraded_count: number;
  drift_count: number;
}

export default function HealthDashboardPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('healthDashboard', lang);
  const dt = useDict('devices', lang);
  const ct = useDict('common', lang);

  const [summary, setSummary] = useState<HealthSummary | null>(null);
  const [devices, setDevices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [presenceNow, setPresenceNow] = useState(() => Date.now());
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const [editingDeviceId, setEditingDeviceId] = useState<string | null>(null);
  const [editNameDraft, setEditNameDraft] = useState('');
  const [renameSaving, setRenameSaving] = useState(false);
  const [renameError, setRenameError] = useState('');

  useEffect(() => {
    const id = setInterval(() => setPresenceNow(Date.now()), PRESENCE_TICK_MS);
    return () => clearInterval(id);
  }, []);

  const fetchData = useCallback(async (initial = false, page?: number, pageSize?: number) => {
    if (initial) setLoading(true);
    try {
      const currentPage = page || pagination.page;
      const currentPageSize = pageSize || pagination.pageSize;
      const [sum, devList] = await Promise.all([
        api.get<HealthSummary>(`/tenants/${params.tenantId}/health/summary`).catch(() => null),
        api.get<any>(`/tenants/${params.tenantId}/devices?page=${currentPage}&page_size=${currentPageSize}`),
      ]);
      setSummary(sum);
      const devicesList = Array.isArray(devList) ? devList : (devList?.items ?? []);
      setDevices(devicesList);
      setPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: devList?.total || devicesList.length
      }));
    } catch {
      setDevices([]);
    } finally {
      if (initial) setLoading(false);
    }
  }, [params.tenantId, pagination.page, pagination.pageSize]);

  useEffect(() => {
    fetchData(true);
    const interval = setInterval(() => fetchData(false), POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleRevoke = async (id: string) => {
    if (!confirm(dt.revokeConfirm)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/devices/${id}`);
      if (editingDeviceId === id) cancelRename();
      fetchData(false);
    } catch (error) {
      console.error('Error revoking device:', error);
    }
  };

  const startRename = (d: { id: string; device_name: string }) => {
    setEditingDeviceId(d.id);
    setEditNameDraft(d.device_name || '');
    setRenameError('');
  };

  const cancelRename = () => {
    setEditingDeviceId(null);
    setEditNameDraft('');
    setRenameError('');
  };

  const saveRename = async () => {
    if (!editingDeviceId) return;
    const trimmed = editNameDraft.trim();
    if (!trimmed) {
      setRenameError((dt as any).deviceNameRequired);
      return;
    }
    setRenameSaving(true);
    setRenameError('');
    try {
      await api.put(`/tenants/${params.tenantId}/devices/${editingDeviceId}`, { device_name: trimmed });
      cancelRename();
      fetchData(false);
    } catch (error) {
      setRenameError(error instanceof APIError ? error.message : (dt as any).renameFailed);
    } finally {
      setRenameSaving(false);
    }
  };

  const handlePageChange = (newPage: number) => {
    fetchData(false, newPage, pagination.pageSize);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    fetchData(false, 1, newPageSize);
  };

  const statusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'pending_activation': return 'bg-yellow-100 text-yellow-800';
      case 'quarantined': return 'bg-orange-100 text-orange-800';
      case 'revoked': case 'retired': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const onlineCount = devices.filter(d => isOnline(d.last_seen_at, presenceNow)).length;
  const offlineCount = devices.length - onlineCount;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      {loading ? (
        <div className="p-8 text-center text-gray-500">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4" />
          <p>{ct.loading}</p>
        </div>
      ) : (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
              <div className="text-sm text-gray-500">{t.totalDevices}</div>
              <div className="text-2xl font-bold text-gray-900">{summary?.total_devices ?? devices.length}</div>
            </div>
            <div className="bg-white rounded-lg shadow-sm border border-green-200 p-4">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                <span className="text-sm text-green-600">{t.online}</span>
              </div>
              <div className="text-2xl font-bold text-green-700">{summary?.online_devices ?? onlineCount}</div>
            </div>
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 bg-gray-400 rounded-full" />
                <span className="text-sm text-gray-500">{t.offline}</span>
              </div>
              <div className="text-2xl font-bold text-gray-700">{summary?.offline_devices ?? offlineCount}</div>
            </div>
            {summary && summary.drift_count > 0 && (
              <div className="bg-white rounded-lg shadow-sm border border-orange-200 p-4">
                <div className="text-sm text-orange-600">{t.drifts}</div>
                <div className="text-2xl font-bold text-orange-700">{summary.drift_count}</div>
              </div>
            )}
          </div>

          {/* Devices table */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            {devices.length === 0 ? (
              <div className="p-8 text-center text-gray-500">{dt.noDevices}</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-3 py-3 text-left text-xs font-medium text-gray-500 uppercase w-8"></th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{dt.deviceName}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{dt.hostname}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{dt.platform}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{dt.version}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{dt.lastSeen}</th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {devices.map((d: any) => {
                      const online = isOnline(d.last_seen_at, presenceNow);
                      return (
                        <tr key={d.id} className="hover:bg-gray-50">
                          <td className="px-3 py-4 whitespace-nowrap">
                            <span
                              className={`inline-block w-2.5 h-2.5 rounded-full ${online ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`}
                              title={online ? dt.online : dt.offline}
                            />
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-900 min-w-[200px]">
                            {editingDeviceId === d.id ? (
                              <div className="flex flex-col gap-1">
                                <div className="flex flex-wrap items-center gap-2">
                                  <input
                                    type="text"
                                    value={editNameDraft}
                                    onChange={e => setEditNameDraft(e.target.value)}
                                    onKeyDown={e => {
                                      if (e.key === 'Enter') saveRename();
                                      if (e.key === 'Escape') cancelRename();
                                    }}
                                    className="border border-gray-300 rounded-md px-2 py-1 text-sm min-w-[10rem] max-w-[20rem]"
                                    disabled={renameSaving}
                                    autoFocus
                                  />
                                  <button type="button" onClick={saveRename} disabled={renameSaving} className="text-blue-600 hover:text-blue-800 text-sm font-medium disabled:opacity-50">{ct.save}</button>
                                  <button type="button" onClick={cancelRename} disabled={renameSaving} className="text-gray-600 hover:text-gray-800 text-sm disabled:opacity-50">{ct.cancel}</button>
                                </div>
                                {renameError && <p className="text-xs text-red-600">{renameError}</p>}
                              </div>
                            ) : (
                              <div className="flex items-center gap-1.5">
                                <span className="font-medium">{d.device_name}</span>
                                <button
                                  type="button"
                                  onClick={() => startRename(d)}
                                  className="inline-flex items-center justify-center rounded p-1 text-gray-400 hover:text-blue-600 hover:bg-blue-50 shrink-0"
                                  title={(dt as any).renameDeviceHint}
                                >
                                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0 1 15.75 21H5.25A2.25 2.25 0 0 1 3 18.75V8.25A2.25 2.25 0 0 1 5.25 6H10" />
                                  </svg>
                                </button>
                              </div>
                            )}
                          </td>
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
                            <button type="button" onClick={() => handleRevoke(d.id)} className="text-red-600 hover:text-red-900">{dt.revoke}</button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
            {!loading && devices.length > 0 && (
              <div className="flex justify-between items-center px-6 py-4 border-t border-gray-200">
                <div className="flex items-center space-x-4">
                  <div className="text-sm text-gray-500">
                    共 {pagination.total} 条记录
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-sm text-gray-500">每页显示：</span>
                    <select 
                      value={pagination.pageSize} 
                      onChange={(e) => handlePageSizeChange(parseInt(e.target.value))}
                      className="border rounded-md px-2 py-1 text-sm"
                    >
                      <option value="10">10条</option>
                      <option value="20">20条</option>
                      <option value="50">50条</option>
                      <option value="100">100条</option>
                    </select>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <button 
                    onClick={() => handlePageChange(pagination.page - 1)}
                    disabled={pagination.page === 1}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    上一页
                  </button>
                  <span className="text-sm">{pagination.page}</span>
                  <button 
                    onClick={() => handlePageChange(pagination.page + 1)}
                    disabled={pagination.page * pagination.pageSize >= pagination.total}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
