"use client";

import { useState, useEffect } from 'react';
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
  const [formData, setFormData] = useState({
    agent_profile_id: '',
    distribution_mode: 'standard',
    platform: 'linux',
    arch: 'amd64',
    version: '0.1.0',
  });

  const fetchPackages = async () => {
    try {
      const data = await api.get<DownloadPackage[]>(`/tenants/${params.tenantId}/download-packages`);
      setPackages(Array.isArray(data) ? data : []);
    } catch {
      setPackages([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPackages();
  }, [params.tenantId]);

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await api.post(`/tenants/${params.tenantId}/download-packages`, formData);
      setIsModalOpen(false);
      fetchPackages();
    } catch (err: any) {
      alert(err.message || ct.error);
    } finally {
      setSubmitting(false);
    }
  };

  // Also generate a download link (enrollment token)
  const handleGenerateLink = async () => {
    setSubmitting(true);
    try {
      const resp = await api.post<{ token: string; expires_at: string }>(`/tenants/${params.tenantId}/download-links`, {
        agent_profile_id: formData.agent_profile_id || undefined,
        max_uses: 10,
        expires_in: 72,
      });
      alert(`Download link token: ${resp.token}\nExpires: ${resp.expires_at}`);
      setIsModalOpen(false);
    } catch (err: any) {
      alert(err.message || ct.error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <div className="flex space-x-3">
          <button
            onClick={() => setIsModalOpen(true)}
            className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
          >
            {t.generateBtn}
          </button>
        </div>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
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
                  type="button"
                  onClick={handleGenerateLink}
                  disabled={submitting}
                  className="px-4 py-2 border border-blue-600 text-blue-600 rounded-md text-sm font-medium hover:bg-blue-50 disabled:opacity-50"
                >
                  {t.generateBtn}
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
          </div>
        </div>
      )}

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
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.arch}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.version}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.signStatus}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {packages.map((pkg) => (
                <tr key={pkg.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{pkg.package_name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{pkg.platform}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{pkg.arch}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{pkg.version}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      pkg.sign_status === 'signed' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                    }`}>
                      {pkg.sign_status}
                    </span>
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
