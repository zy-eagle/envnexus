"use client";

import { useState, useEffect, useCallback } from 'react';
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
  result: string;
  note: string;
  created_at: string;
}

interface BrowseEntry {
  name: string;
  is_dir: boolean;
  size_bytes: number;
  modified_at: string;
  mode: string;
}

interface FileResult {
  request_id?: string;
  status?: string;
  error?: string;
  summary?: string;
  download_url?: string;
  output?: {
    path?: string;
    entries?: BrowseEntry[];
    dir_count?: number;
    file_count?: number;
    total_size?: number;
    truncated?: boolean;
    content?: string;
    lines_read?: number;
    file_size?: number;
    file_size_bytes?: number;
    content_type?: string;
  };
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function parseResult(resultStr: string): FileResult | null {
  if (!resultStr) return null;
  try {
    return JSON.parse(resultStr);
  } catch {
    return null;
  }
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
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
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
  }, [params.tenantId, statusFilter]);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  // Auto-refresh for approved requests that may have pending results
  useEffect(() => {
    const hasWaiting = requests.some(r => r.status === 'approved' && !r.result);
    if (!hasWaiting) return;
    const timer = setInterval(fetchRequests, 5000);
    return () => clearInterval(timer);
  }, [requests, fetchRequests]);

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

  const actionLabel = (action: string) => {
    switch (action) {
      case 'browse': return t.browse;
      case 'preview': return t.preview;
      case 'download': return t.download;
      default: return action;
    }
  };

  const renderResultBadge = (req: FileAccessRequest) => {
    if (req.status !== 'approved') return null;
    const result = parseResult(req.result);
    if (!result) {
      return (
        <span className="inline-flex items-center gap-1 text-xs text-gray-400">
          <span className="animate-pulse inline-block w-2 h-2 bg-blue-400 rounded-full" />
          {t.noResult}
        </span>
      );
    }
    if (result.status === 'failed' || result.error) {
      return <span className="text-xs text-red-600 font-medium">{t.resultFailed}</span>;
    }
    return <span className="text-xs text-green-600 font-medium">{t.resultSuccess}</span>;
  };

  const renderResultDetail = (req: FileAccessRequest) => {
    const result = parseResult(req.result);
    if (!result) return null;

    if (result.error) {
      return (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-sm text-red-700 font-mono">{result.error}</p>
        </div>
      );
    }

    const output = result.output;
    if (!output) return <p className="text-sm text-gray-500">{result.summary}</p>;

    if (req.action === 'browse' && output.entries) {
      return (
        <div className="space-y-2">
          <p className="text-xs text-gray-500">
            {output.dir_count} dirs, {output.file_count} files
            {output.total_size != null && ` · ${formatBytes(output.total_size)}`}
            {output.truncated && ' (truncated)'}
          </p>
          <div className="max-h-80 overflow-auto border border-gray-200 rounded-lg">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50 sticky top-0">
                <tr>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t.fileName}</th>
                  <th className="px-3 py-2 text-right text-xs font-medium text-gray-500">{t.fileSize}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t.modTime}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {output.entries.map((entry, i) => (
                  <tr key={i} className="hover:bg-gray-50">
                    <td className="px-3 py-1.5 whitespace-nowrap">
                      <span className={entry.is_dir ? 'text-blue-600 font-medium' : 'text-gray-700'}>
                        {entry.is_dir ? '📁 ' : '📄 '}{entry.name}
                      </span>
                    </td>
                    <td className="px-3 py-1.5 whitespace-nowrap text-right text-gray-500 font-mono text-xs">
                      {entry.is_dir ? '-' : formatBytes(entry.size_bytes)}
                    </td>
                    <td className="px-3 py-1.5 whitespace-nowrap text-gray-500 text-xs">
                      {entry.modified_at ? new Date(entry.modified_at).toLocaleString() : '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      );
    }

    if (req.action === 'preview' && output.content != null) {
      return (
        <div className="space-y-2">
          <p className="text-xs text-gray-500">
            {output.lines_read} lines · {output.file_size != null && formatBytes(output.file_size)}
          </p>
          <pre className="bg-gray-900 text-green-300 p-4 rounded-lg text-xs font-mono overflow-auto max-h-96 whitespace-pre-wrap">
            {output.content}
          </pre>
        </div>
      );
    }

    if (req.action === 'download' && result.download_url) {
      return (
        <div className="space-y-2">
          <p className="text-sm text-gray-700">{result.summary}</p>
          <a
            href={result.download_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700"
          >
            {t.download}
          </a>
          {output.file_size != null && (
            <span className="text-xs text-gray-500 ml-2">{formatBytes(output.file_size)}</span>
          )}
        </div>
      );
    }

    return <pre className="text-xs bg-gray-50 p-3 rounded-lg overflow-auto max-h-60">{JSON.stringify(output, null, 2)}</pre>;
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

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
            {s === '' ? t.all : statusLabel(s)}
          </button>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : requests.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noRequests}</div>
        ) : (
          <div className="divide-y divide-gray-200">
            {requests.map(req => (
              <div key={req.id}>
                <div
                  className={`grid grid-cols-12 gap-4 items-center px-6 py-4 hover:bg-gray-50 transition-colors ${
                    req.result ? 'cursor-pointer' : ''
                  }`}
                  onClick={() => req.result && setExpandedId(expandedId === req.id ? null : req.id)}
                >
                  <div className="col-span-2 text-sm font-mono text-gray-900 truncate" title={req.device_id}>
                    {req.device_id.slice(0, 12)}...
                  </div>
                  <div className="col-span-3 text-sm text-gray-700 truncate font-mono" title={req.path}>
                    {req.path}
                  </div>
                  <div className="col-span-1 text-sm text-gray-500">
                    {actionLabel(req.action)}
                  </div>
                  <div className="col-span-2">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColor(req.status)}`}>
                      {statusLabel(req.status)}
                    </span>
                  </div>
                  <div className="col-span-2">
                    {renderResultBadge(req)}
                  </div>
                  <div className="col-span-2 flex justify-end gap-2">
                    {req.status === 'pending' && (
                      <>
                        <button
                          onClick={(e) => { e.stopPropagation(); handleAction(req.id, 'approve'); }}
                          className="text-green-600 hover:text-green-900 text-xs font-medium px-2 py-1 rounded hover:bg-green-50"
                        >
                          {t.approve}
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); handleAction(req.id, 'deny'); }}
                          className="text-red-600 hover:text-red-900 text-xs font-medium px-2 py-1 rounded hover:bg-red-50"
                        >
                          {t.deny}
                        </button>
                      </>
                    )}
                    {req.result && (
                      <button
                        onClick={(e) => { e.stopPropagation(); setExpandedId(expandedId === req.id ? null : req.id); }}
                        className="text-indigo-600 hover:text-indigo-900 text-xs font-medium px-2 py-1 rounded hover:bg-indigo-50"
                      >
                        {expandedId === req.id ? t.closePreview : t.viewResult}
                      </button>
                    )}
                  </div>
                </div>

                {expandedId === req.id && req.result && (
                  <div className="px-6 pb-4 bg-gray-50 border-t border-gray-100">
                    <div className="pt-3">
                      {renderResultDetail(req)}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
