"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface DownloadPackage {
  id: string;
  tenant_id: string;
  agent_profile_id: string;
  distribution_mode: string;
  platform: string;
  arch: string;
  version: string;
  package_name: string;
  download_url: string;
  checksum: string;
  sign_status: string;
  activation_mode: string;
  activation_key?: string;
  max_devices: number;
  bound_count: number;
  created_at: string;
}

interface DeviceBinding {
  id: string;
  device_code: string;
  device_info?: { os: string; hostname: string; cpu_model: string };
  status: string;
  bound_by: string;
  bound_at: string;
  last_heartbeat?: string;
}

interface AuditLog {
  id: string;
  package_id: string;
  device_code: string;
  action: string;
  actor: string;
  detail?: string;
  created_at: string;
}

export default function DownloadPackagesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('downloadPackages', lang);
  const ct = useDict('common', lang);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [packages, setPackages] = useState<DownloadPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [createdKey, setCreatedKey] = useState('');
  const [formData, setFormData] = useState({
    agent_profile_id: '',
    distribution_mode: 'standard',
    platform: 'linux',
    arch: 'amd64',
    version: '0.1.0',
    activation_mode: 'auto',
    max_devices: 1,
  });

  // Binding modal state
  const [bindingPkg, setBindingPkg] = useState<DownloadPackage | null>(null);
  const [bindings, setBindings] = useState<DeviceBinding[]>([]);
  const [bindingLoading, setBindingLoading] = useState(false);
  const [bindDeviceCode, setBindDeviceCode] = useState('');

  // Audit log modal state
  const [auditPkg, setAuditPkg] = useState<DownloadPackage | null>(null);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);

  const fetchPackages = useCallback(async () => {
    try {
      const data = await api.get<DownloadPackage[]>(`/tenants/${params.tenantId}/download-packages`);
      setPackages(Array.isArray(data) ? data : []);
    } catch {
      setPackages([]);
    } finally {
      setLoading(false);
    }
  }, [params.tenantId]);

  useEffect(() => {
    fetchPackages();
  }, [fetchPackages]);

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setCreatedKey('');
    try {
      const resp = await api.post<DownloadPackage>(`/tenants/${params.tenantId}/download-packages`, formData);
      if (resp.activation_key) {
        setCreatedKey(resp.activation_key);
      } else {
        setIsModalOpen(false);
      }
      fetchPackages();
    } catch (err: any) {
      alert(err.message || ct.error);
    } finally {
      setSubmitting(false);
    }
  };

  const fetchBindings = async (pkg: DownloadPackage) => {
    setBindingPkg(pkg);
    setBindingLoading(true);
    try {
      const data = await api.get<DeviceBinding[]>(`/tenants/${params.tenantId}/download-packages/${pkg.id}/bindings`);
      setBindings(Array.isArray(data) ? data : []);
    } catch {
      setBindings([]);
    } finally {
      setBindingLoading(false);
    }
  };

  const handleBind = async () => {
    if (!bindingPkg || !bindDeviceCode) return;
    try {
      await api.post(`/tenants/${params.tenantId}/download-packages/${bindingPkg.id}/bind`, {
        device_code: bindDeviceCode,
      });
      setBindDeviceCode('');
      fetchBindings(bindingPkg);
      fetchPackages();
    } catch (err: any) {
      alert(err.message || ct.error);
    }
  };

  const handleUnbind = async (bindingId: string) => {
    if (!bindingPkg || !confirm(t.unbindConfirm)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/download-packages/${bindingPkg.id}/bindings/${bindingId}`);
      fetchBindings(bindingPkg);
      fetchPackages();
    } catch (err: any) {
      alert(err.message || ct.error);
    }
  };

  const fetchAuditLogs = async (pkg: DownloadPackage) => {
    setAuditPkg(pkg);
    setAuditLoading(true);
    try {
      const resp = await api.get<{ data: AuditLog[]; total: number }>(`/tenants/${params.tenantId}/download-packages/${pkg.id}/audit-logs`);
      setAuditLogs(Array.isArray(resp?.data) ? resp.data : []);
    } catch {
      setAuditLogs([]);
    } finally {
      setAuditLoading(false);
    }
  };

  const modeLabel = (mode: string) =>
    mode === 'manual' ? t.activationModeManual : t.activationModeAuto;

  const actionBadge = (action: string) => {
    const colors: Record<string, string> = {
      activate: 'bg-green-100 text-green-800',
      bind: 'bg-blue-100 text-blue-800',
      unbind: 'bg-yellow-100 text-yellow-800',
      revoke: 'bg-red-100 text-red-800',
      heartbeat_fail: 'bg-red-100 text-red-800',
    };
    return colors[action] || 'bg-gray-100 text-gray-800';
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button
          onClick={() => { setIsModalOpen(true); setCreatedKey(''); }}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.generateBtn}
        </button>
      </div>

      {/* Create Package Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
            {createdKey ? (
              <div className="space-y-4">
                <h2 className="text-xl font-semibold text-green-700">{t.activationKey}</h2>
                <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
                  <p className="text-sm text-yellow-800 mb-2">{t.activationKeyWarning}</p>
                  <code className="block bg-white border rounded px-3 py-2 text-lg font-mono select-all break-all">
                    {createdKey}
                  </code>
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={() => { setIsModalOpen(false); setCreatedKey(''); }}
                    className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                  >
                    {ct.close || 'Close'}
                  </button>
                </div>
              </div>
            ) : (
              <>
                <h2 className="text-xl font-semibold mb-4">{t.modalTitle}</h2>
                <p className="text-sm text-gray-600 mb-6">{t.modalDesc}</p>
                <form onSubmit={handleGenerate} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">{t.agentProfileId}</label>
                    <input
                      type="text"
                      value={formData.agent_profile_id}
                      onChange={e => setFormData({ ...formData, agent_profile_id: e.target.value })}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                      placeholder="(optional)"
                    />
                  </div>

                  {/* Activation Mode */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">{t.activationMode}</label>
                    <div className="grid grid-cols-2 gap-3">
                      {(['auto', 'manual'] as const).map(mode => (
                        <label
                          key={mode}
                          className={`relative flex flex-col p-3 border-2 rounded-lg cursor-pointer transition-colors ${
                            formData.activation_mode === mode
                              ? 'border-blue-500 bg-blue-50'
                              : 'border-gray-200 hover:border-gray-300'
                          }`}
                        >
                          <input
                            type="radio"
                            name="activation_mode"
                            value={mode}
                            checked={formData.activation_mode === mode}
                            onChange={() => setFormData({ ...formData, activation_mode: mode })}
                            className="sr-only"
                          />
                          <span className="text-sm font-medium text-gray-900">
                            {mode === 'auto' ? t.activationModeAuto : t.activationModeManual}
                          </span>
                          <span className="text-xs text-gray-500 mt-1">
                            {mode === 'auto' ? t.activationModeAutoDesc : t.activationModeManualDesc}
                          </span>
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Max Devices */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">{t.maxDevices}</label>
                    <input
                      type="number"
                      min={1}
                      max={9999}
                      value={formData.max_devices}
                      onChange={e => setFormData({ ...formData, max_devices: parseInt(e.target.value) || 1 })}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                    />
                    <p className="text-xs text-gray-500 mt-1">{t.maxDevicesDesc}</p>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">{t.platform}</label>
                      <select
                        value={formData.platform}
                        onChange={e => setFormData({ ...formData, platform: e.target.value })}
                        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                      >
                        <option value="linux">Linux</option>
                        <option value="windows">Windows</option>
                        <option value="darwin">macOS</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">{t.arch}</label>
                      <select
                        value={formData.arch}
                        onChange={e => setFormData({ ...formData, arch: e.target.value })}
                        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                      >
                        <option value="amd64">amd64</option>
                        <option value="arm64">arm64</option>
                      </select>
                    </div>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">{t.version}</label>
                    <input
                      type="text"
                      required
                      value={formData.version}
                      onChange={e => setFormData({ ...formData, version: e.target.value })}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                  <div className="flex justify-end space-x-3 mt-6">
                    <button
                      type="button"
                      onClick={() => setIsModalOpen(false)}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      {ct.cancel}
                    </button>
                    <button
                      type="submit"
                      disabled={submitting}
                      className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                    >
                      {submitting ? t.generating : ct.create}
                    </button>
                  </div>
                </form>
              </>
            )}
          </div>
        </div>
      )}

      {/* Bindings Modal */}
      {bindingPkg && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">{t.bindings} — {bindingPkg.package_name}</h2>
              <button onClick={() => setBindingPkg(null)} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
            </div>

            {bindingPkg.activation_mode === 'manual' && (
              <div className="flex space-x-2 mb-4">
                <input
                  type="text"
                  value={bindDeviceCode}
                  onChange={e => setBindDeviceCode(e.target.value)}
                  placeholder={t.deviceCodePlaceholder}
                  className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:ring-blue-500 focus:border-blue-500"
                />
                <button
                  onClick={handleBind}
                  disabled={!bindDeviceCode}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                >
                  {t.bindDevice}
                </button>
              </div>
            )}

            {bindingLoading ? (
              <div className="p-4 text-center text-gray-500">{ct.loading}</div>
            ) : bindings.length === 0 ? (
              <div className="p-4 text-center text-gray-500">{t.noBindings}</div>
            ) : (
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceCode}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.boundBy}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.lastHeartbeat}</th>
                    <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {bindings.map(b => (
                    <tr key={b.id}>
                      <td className="px-4 py-2 text-sm font-mono">{b.device_code}</td>
                      <td className="px-4 py-2 text-sm">
                        <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${
                          b.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>{b.status}</span>
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500">{b.bound_by || 'system'}</td>
                      <td className="px-4 py-2 text-sm text-gray-500">
                        {b.last_heartbeat ? new Date(b.last_heartbeat).toLocaleString() : '-'}
                      </td>
                      <td className="px-4 py-2 text-right">
                        {b.status === 'active' && (
                          <button
                            onClick={() => handleUnbind(b.id)}
                            className="text-red-600 hover:text-red-800 text-sm font-medium"
                          >
                            {t.unbind}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {/* Audit Logs Modal */}
      {auditPkg && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">{t.auditLogs} — {auditPkg.package_name}</h2>
              <button onClick={() => setAuditPkg(null)} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
            </div>

            {auditLoading ? (
              <div className="p-4 text-center text-gray-500">{ct.loading}</div>
            ) : auditLogs.length === 0 ? (
              <div className="p-4 text-center text-gray-500">{t.noAuditLogs}</div>
            ) : (
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.action}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceCode}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">{t.actor}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {auditLogs.map(log => (
                    <tr key={log.id}>
                      <td className="px-4 py-2 text-sm">
                        <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${actionBadge(log.action)}`}>
                          {log.action}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-sm font-mono">{log.device_code || '-'}</td>
                      <td className="px-4 py-2 text-sm text-gray-500">{log.actor || 'system'}</td>
                      <td className="px-4 py-2 text-sm text-gray-500">{new Date(log.created_at).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {/* Package List */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : packages.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noPackages}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.packageName}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.platform}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.version}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.activationMode}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.boundCount}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.signStatus}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider"></th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {packages.map((pkg) => (
                <tr key={pkg.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{pkg.package_name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{pkg.platform}/{pkg.arch}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{pkg.version}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${
                      pkg.activation_mode === 'manual' ? 'bg-purple-100 text-purple-800' : 'bg-blue-100 text-blue-800'
                    }`}>
                      {modeLabel(pkg.activation_mode)}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {pkg.bound_count} / {pkg.max_devices}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      pkg.sign_status === 'signed' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                    }`}>
                      {pkg.sign_status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-3">
                    <button
                      onClick={() => fetchBindings(pkg)}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      {t.bindings}
                    </button>
                    <button
                      onClick={() => fetchAuditLogs(pkg)}
                      className="text-gray-600 hover:text-gray-900"
                    >
                      {t.auditLogs}
                    </button>
                    {pkg.download_url && (
                      <a href={pkg.download_url} target="_blank" rel="noopener noreferrer" className="text-green-600 hover:text-green-900">
                        Download
                      </a>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
