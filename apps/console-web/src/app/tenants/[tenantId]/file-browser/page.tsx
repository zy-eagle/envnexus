"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface FileAccessRequest {
  id: string;
  tenant_id: string;
  device_id: string;
  requested_by: string;
  path: string;
  action: string;
  status: string;
  note: string;
  created_at: string;
}

export default function FileBrowserPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('fileBrowser', lang);
  const ct = useDict('common', lang);
  const [requests, setRequests] = useState<FileAccessRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('');
  const [formDeviceId, setFormDeviceId] = useState('');
  const [formPath, setFormPath] = useState('');
  const [formAction, setFormAction] = useState<'browse' | 'preview' | 'download'>('browse');
  const [creating, setCreating] = useState(false);

  const fetchRequests = async () => {
    setLoading(true);
    try {
      const qs = statusFilter ? `?status=${statusFilter}` : '';
      const data = await api.get<{ items: FileAccessRequest[] }>(`/tenants/${params.tenantId}/file-access-requests${qs}`);
      setRequests(Array.isArray(data) ? data : (data as any)?.items || []);
    } catch {
      setRequests([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchRequests(); }, [params.tenantId, statusFilter]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formDeviceId || !formPath) return;
    setCreating(true);
    try {
      await api.post(`/tenants/${params.tenantId}/file-access-requests`, {
        device_id: formDeviceId,
        path: formPath,
        action: formAction,
      });
      setFormDeviceId('');
      setFormPath('');
      fetchRequests();
    } catch (error) {
      console.error('Failed to create file access request:', error);
    } finally {
      setCreating(false);
    }
  };

  const handleAction = async (id: string, action: 'approve' | 'deny') => {
    try {
      await api.post(`/tenants/${params.tenantId}/file-access-requests/${id}/${action}`);
      fetchRequests();
    } catch (error) {
      console.error(`Failed to ${action} request:`, error);
    }
  };

  const statusColor = (status: string) => {
    switch (status) {
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'approved': return 'bg-green-100 text-green-800';
      case 'denied': return 'bg-red-100 text-red-800';
      case 'expired': return 'bg-gray-100 text-gray-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const statusLabel = (status: string) => {
    switch (status) {
      case 'pending': return t.pending;
      case 'approved': return t.approved;
      case 'denied': return t.denied;
      case 'expired': return t.expired;
      default: return status;
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      {/* New request form */}
      <form onSubmit={handleCreate} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <input
            type="text"
            placeholder={t.selectDevice}
            value={formDeviceId}
            onChange={e => setFormDeviceId(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="text"
            placeholder={t.path}
            value={formPath}
            onChange={e => setFormPath(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <select
            value={formAction}
            onChange={e => setFormAction(e.target.value as any)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="browse">{t.browse}</option>
            <option value="preview">{t.preview}</option>
            <option value="download">{t.download}</option>
          </select>
          <button
            type="submit"
            disabled={creating}
            className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
          >
            {t.requestAccess}
          </button>
        </div>
      </form>

      {/* Status filter */}
      <div className="flex gap-2">
        {['', 'pending', 'approved', 'denied', 'expired'].map(s => (
          <button
            key={s}
            onClick={() => setStatusFilter(s)}
            className={`px-3 py-1 rounded-full text-xs font-medium border transition-colors ${
              statusFilter === s
                ? 'bg-indigo-600 text-white border-indigo-600'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'
            }`}
          >
            {s === '' ? 'All' : statusLabel(s)}
          </button>
        ))}
      </div>

      {/* Requests table */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : requests.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noFiles}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.selectDevice}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.path}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.action}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.status}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {requests.map(req => (
                <tr key={req.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{req.device_id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{req.path}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{req.action}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColor(req.status)}`}>
                      {statusLabel(req.status)}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                    {req.status === 'pending' && (
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => handleAction(req.id, 'approve')}
                          className="text-green-600 hover:text-green-900 text-xs font-medium"
                        >
                          {t.approve}
                        </button>
                        <button
                          onClick={() => handleAction(req.id, 'deny')}
                          className="text-red-600 hover:text-red-900 text-xs font-medium"
                        >
                          {t.deny}
                        </button>
                      </div>
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
