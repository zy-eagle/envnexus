"use client";

import { useState, useEffect, useCallback, useMemo } from 'react';
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

interface Device {
  id: string;
  device_name: string;
  hostname: string | null;
  platform: string;
  arch: string;
  status: string;
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
  try { return JSON.parse(resultStr); } catch { return null; }
}

// Detect Windows drive root like "C:" or "D:"
function isDriveRoot(p: string): boolean {
  return /^[A-Za-z]:$/.test(p);
}

function joinPath(base: string, child: string): string {
  // Clicking a drive entry (e.g. "C:") from the root "/" view
  if ((base === '/' || base === '') && isDriveRoot(child)) return child + '/';
  if (base === '/' || base === '') return '/' + child;
  // Inside a Windows drive like "C:/" or "C:/Users"
  return base.replace(/\/+$/, '') + '/' + child;
}

function splitPath(path: string): string[] {
  // "C:/Users/foo" → ["C:", "Users", "foo"]
  const match = path.match(/^([A-Za-z]:)([\\/].*)?$/);
  if (match) {
    const rest = (match[2] || '').split(/[\\/]/).filter(Boolean);
    return [match[1], ...rest];
  }
  return path.split('/').filter(Boolean);
}

function parentPath(segs: string[]): string {
  if (segs.length <= 1) return '/';
  const parent = segs.slice(0, -1);
  if (parent.length === 1 && isDriveRoot(parent[0])) return parent[0] + '/';
  if (isDriveRoot(parent[0])) return parent[0] + '/' + parent.slice(1).join('/');
  return '/' + parent.join('/');
}

function segmentsToPath(segs: string[], upTo: number): string {
  const sub = segs.slice(0, upTo + 1);
  if (sub.length === 1 && isDriveRoot(sub[0])) return sub[0] + '/';
  if (isDriveRoot(sub[0])) return sub[0] + '/' + sub.slice(1).join('/');
  return '/' + sub.join('/');
}

export default function FileBrowserPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('fileBrowser', lang);
  const ct = useDict('common', lang);

  const [devices, setDevices] = useState<Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [currentPath, setCurrentPath] = useState('/');
  const [manualPath, setManualPath] = useState('/');
  const [requests, setRequests] = useState<FileAccessRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showHistory, setShowHistory] = useState(false);
  const [devicePagination, setDevicePagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });
  const [requestPagination, setRequestPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  // Active browse request tracking
  const [activeBrowseId, setActiveBrowseId] = useState<string | null>(null);
  const [browseEntries, setBrowseEntries] = useState<BrowseEntry[] | null>(null);
  const [browseError, setBrowseError] = useState<string | null>(null);
  const [browseMeta, setBrowseMeta] = useState<{ dir_count?: number; file_count?: number; total_size?: number; truncated?: boolean } | null>(null);

  // Preview/download state
  const [previewContent, setPreviewContent] = useState<{ content: string; lines: number; size: number } | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [previewAction, setPreviewAction] = useState<string | null>(null);
  const [pendingDownloadId, setPendingDownloadId] = useState<string | null>(null);

  const fetchDevices = useCallback(async (page?: number, pageSize?: number) => {
    try {
      const currentPage = page || devicePagination.page;
      const currentPageSize = pageSize || devicePagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/devices?page=${currentPage}&page_size=${currentPageSize}`);
      setDevices(Array.isArray(data) ? data : (data?.items ?? []));
      setDevicePagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch { setDevices([]); }
  }, [params.tenantId, devicePagination.page, devicePagination.pageSize]);

  const fetchRequests = useCallback(async (page?: number, pageSize?: number) => {
    try {
      const currentPage = page || requestPagination.page;
      const currentPageSize = pageSize || requestPagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/file-access-requests?page=${currentPage}&page_size=${currentPageSize}`);
      setRequests(Array.isArray(data) ? data : (data?.items ?? []));
      setRequestPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch { setRequests([]); }
  }, [params.tenantId, requestPagination.page, requestPagination.pageSize]);

  useEffect(() => { fetchDevices(); fetchRequests(); }, [fetchDevices, fetchRequests]);

  const getDeviceName = useCallback((deviceId: string) => {
    const dev = devices.find(d => d.id === deviceId);
    return dev ? (dev.device_name || dev.hostname || deviceId.slice(0, 12)) : deviceId.slice(0, 12) + '...';
  }, [devices]);

  const browsePath = useCallback(async (path: string) => {
    if (!selectedDeviceId) return;
    setLoading(true);
    setBrowseEntries(null);
    setBrowseError(null);
    setBrowseMeta(null);
    setPreviewContent(null);
    setDownloadUrl(null);
    setPreviewAction(null);
    setSearchQuery('');
    setCurrentPath(path);
    setManualPath(path);

    try {
      const resp = await api.post<FileAccessRequest>(`/tenants/${params.tenantId}/file-access-requests`, {
        device_id: selectedDeviceId,
        path,
        action: 'browse',
      });
      const reqId = (resp as any)?.id;
      if (reqId) {
        setActiveBrowseId(reqId);
      }
      fetchRequests();
    } catch (error) {
      console.error('Failed to create browse request:', error);
      setBrowseError('Failed to create request');
      setLoading(false);
    }
  }, [selectedDeviceId, params.tenantId, fetchRequests]);

  const requestPreview = useCallback(async (filePath: string) => {
    if (!selectedDeviceId) return;
    setPreviewContent(null);
    setPreviewAction('preview');
    setLoading(true);

    try {
      const resp = await api.post<FileAccessRequest>(`/tenants/${params.tenantId}/file-access-requests`, {
        device_id: selectedDeviceId,
        path: filePath,
        action: 'preview',
      });
      const reqId = (resp as any)?.id;
      if (reqId) setActiveBrowseId(reqId);
      fetchRequests();
    } catch (error) {
      console.error('Failed to create preview request:', error);
      setLoading(false);
    }
  }, [selectedDeviceId, params.tenantId, fetchRequests]);

  const requestDownload = useCallback(async (filePath: string) => {
    if (!selectedDeviceId) return;
    setDownloadUrl(null);
    setPreviewAction('download');
    setLoading(true);

    try {
      const resp = await api.post<FileAccessRequest>(`/tenants/${params.tenantId}/file-access-requests`, {
        device_id: selectedDeviceId,
        path: filePath,
        action: 'download',
      });
      const reqId = (resp as any)?.id;
      if (reqId) setActiveBrowseId(reqId);
      fetchRequests();
    } catch (error) {
      console.error('Failed to create download request:', error);
      setLoading(false);
    }
  }, [selectedDeviceId, params.tenantId, fetchRequests]);

  // Poll for active request result
  useEffect(() => {
    if (!activeBrowseId) return;
    let pendingDetected = false;
    const poll = setInterval(async () => {
      try {
        const updated = await api.get<FileAccessRequest>(`/tenants/${params.tenantId}/file-access-requests/${activeBrowseId}`);
        const req = updated as any;

        if (req.status === 'approved' && req.result) {
          const result = parseResult(req.result);
          if (result) {
            if (result.error) {
              setBrowseError(result.error);
            } else if (result.output?.entries) {
              setBrowseEntries(result.output.entries);
              setBrowseMeta({
                dir_count: result.output.dir_count,
                file_count: result.output.file_count,
                total_size: result.output.total_size,
                truncated: result.output.truncated,
              });
            } else if (result.output?.content != null) {
              setPreviewContent({
                content: result.output.content,
                lines: result.output.lines_read || 0,
                size: result.output.file_size || result.output.file_size_bytes || 0,
              });
            } else if (result.download_url) {
              setDownloadUrl(result.download_url);
            }
          }
          setPendingDownloadId(null);
          setActiveBrowseId(null);
          setLoading(false);
          fetchRequests();
          clearInterval(poll);
        } else if (req.status === 'pending' && req.action === 'download' && !pendingDetected) {
          pendingDetected = true;
          setPendingDownloadId(req.id);
          setLoading(false);
          fetchRequests();
        } else if (req.status === 'denied') {
          setBrowseError('Request denied');
          setPendingDownloadId(null);
          setActiveBrowseId(null);
          setLoading(false);
          clearInterval(poll);
        } else if (req.status === 'expired') {
          setBrowseError('Request expired');
          setPendingDownloadId(null);
          setActiveBrowseId(null);
          setLoading(false);
          clearInterval(poll);
        }
      } catch { /* keep polling */ }
    }, 2000);
    return () => clearInterval(poll);
  }, [activeBrowseId, params.tenantId, fetchRequests]);

  const filteredEntries = useMemo(() => {
    if (!browseEntries) return null;
    if (!searchQuery.trim()) return browseEntries;
    const q = searchQuery.toLowerCase();
    return browseEntries.filter(e => e.name.toLowerCase().includes(q));
  }, [browseEntries, searchQuery]);

  const handleDirClick = (entry: BrowseEntry) => {
    if (entry.is_dir) {
      browsePath(joinPath(currentPath, entry.name));
    }
  };

  const handleFileClick = (entry: BrowseEntry) => {
    if (!entry.is_dir) {
      requestPreview(joinPath(currentPath, entry.name));
    }
  };

  const handleManualBrowse = (e: React.FormEvent) => {
    e.preventDefault();
    if (manualPath.trim()) browsePath(manualPath.trim());
  };

  const handleApprove = async (id: string) => {
    try {
      await api.post(`/tenants/${params.tenantId}/file-access-requests/${id}/approve`);
      fetchRequests();
    } catch (error) {
      console.error('Failed to approve:', error);
    }
  };

  const handleDeny = async (id: string) => {
    try {
      await api.post(`/tenants/${params.tenantId}/file-access-requests/${id}/deny`);
      fetchRequests();
    } catch (error) {
      console.error('Failed to deny:', error);
    }
  };

  const handleDevicePageChange = (newPage: number) => {
    fetchDevices(newPage, devicePagination.pageSize);
  };

  const handleDevicePageSizeChange = (newPageSize: number) => {
    fetchDevices(1, newPageSize);
  };

  const handleRequestPageChange = (newPage: number) => {
    fetchRequests(newPage, requestPagination.pageSize);
  };

  const handleRequestPageSizeChange = (newPageSize: number) => {
    fetchRequests(1, newPageSize);
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

  const pathSegments = splitPath(currentPath);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button
          onClick={() => setShowHistory(!showHistory)}
          className={`text-sm font-medium px-3 py-1.5 rounded-md transition-colors ${
            showHistory ? 'bg-indigo-100 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'
          }`}
        >
          {(t as any).historyTitle}
        </button>
      </div>

      {/* Device selector + path input */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-12 gap-3">
          <div className="md:col-span-4 space-y-2">
            <select
              value={selectedDeviceId}
              onChange={e => {
                setSelectedDeviceId(e.target.value);
                setBrowseEntries(null);
                setBrowseError(null);
                setPreviewContent(null);
                setDownloadUrl(null);
                setCurrentPath('/');
                setManualPath('/');
              }}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="">{t.selectDevice}</option>
              {devices.map(d => (
                <option key={d.id} value={d.id}>
                  {d.device_name || d.hostname || d.id.substring(0, 12)} — {d.platform}/{d.arch}
                </option>
              ))}
            </select>
            <div className="flex justify-between items-center text-xs text-gray-500">
              <div>共 {devicePagination.total} 台设备</div>
              <div className="flex items-center space-x-2">
                <span>每页显示：</span>
                <select 
                  value={devicePagination.pageSize} 
                  onChange={(e) => handleDevicePageSizeChange(parseInt(e.target.value))}
                  className="border rounded-md px-1 py-0.5 text-xs"
                >
                  <option value="10">10条</option>
                  <option value="20">20条</option>
                  <option value="50">50条</option>
                  <option value="100">100条</option>
                </select>
                <button 
                  onClick={() => handleDevicePageChange(devicePagination.page - 1)}
                  disabled={devicePagination.page === 1}
                  className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                >
                  上一页
                </button>
                <span>{devicePagination.page}</span>
                <button 
                  onClick={() => handleDevicePageChange(devicePagination.page + 1)}
                  disabled={devicePagination.page * devicePagination.pageSize >= devicePagination.total}
                  className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                >
                  下一页
                </button>
              </div>
            </div>
          </div>
          <form onSubmit={handleManualBrowse} className="md:col-span-8 flex gap-2">
            <input
              type="text"
              value={manualPath}
              onChange={e => setManualPath(e.target.value)}
              placeholder={(t as any).manualPathPlaceholder}
              disabled={!selectedDeviceId}
              className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:ring-indigo-500 focus:border-indigo-500 disabled:bg-gray-50 disabled:text-gray-400"
            />
            <button
              type="submit"
              disabled={!selectedDeviceId || loading}
              className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 whitespace-nowrap"
            >
              {(t as any).browseThisPath}
            </button>
          </form>
        </div>

        {/* Breadcrumb */}
        {selectedDeviceId && (
          <div className="flex items-center gap-1 text-sm overflow-x-auto">
            <button
              onClick={() => browsePath('/')}
              className="text-indigo-600 hover:text-indigo-800 font-medium px-1.5 py-0.5 rounded hover:bg-indigo-50 flex-shrink-0"
            >
              {(t as any).root}
            </button>
            {pathSegments.map((seg, i) => (
              <span key={i} className="flex items-center gap-1 flex-shrink-0">
                <span className="text-gray-400">{i === 0 && isDriveRoot(seg) ? '' : '/'}</span>
                <button
                  onClick={() => browsePath(segmentsToPath(pathSegments, i))}
                  className={`px-1.5 py-0.5 rounded font-medium ${
                    i === pathSegments.length - 1
                      ? 'text-gray-900 bg-gray-100'
                      : 'text-indigo-600 hover:text-indigo-800 hover:bg-indigo-50'
                  }`}
                >
                  {seg}
                </button>
              </span>
            ))}
          </div>
        )}
      </div>

      {/* No device selected hint */}
      {!selectedDeviceId && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center text-gray-400">
          <svg className="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" strokeWidth={1} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25A2.25 2.25 0 0 1 5.25 3h13.5A2.25 2.25 0 0 1 21 5.25Z" />
          </svg>
          <p className="text-sm">{(t as any).noDeviceSelected}</p>
        </div>
      )}

      {/* Loading / waiting state */}
      {loading && selectedDeviceId && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-indigo-600 mb-3" />
          <p className="text-sm text-gray-500">
            {activeBrowseId ? (t as any).awaitingResult : ct.loading}
          </p>
        </div>
      )}

      {/* Pending download approval */}
      {pendingDownloadId && !loading && (
        <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="animate-pulse inline-block w-2.5 h-2.5 bg-amber-400 rounded-full" />
            <p className="text-sm text-amber-800 font-medium">{(t as any).awaitingApproval}</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={async () => {
                try {
                  await api.post(`/tenants/${params.tenantId}/file-access-requests/${pendingDownloadId}/approve`);
                  fetchRequests();
                } catch (error) {
                  console.error('Failed to approve:', error);
                }
              }}
              className="text-green-700 bg-green-100 hover:bg-green-200 text-xs font-medium px-3 py-1.5 rounded-md transition-colors"
            >
              {t.approve}
            </button>
            <button
              onClick={async () => {
                try {
                  await api.post(`/tenants/${params.tenantId}/file-access-requests/${pendingDownloadId}/deny`);
                  setPendingDownloadId(null);
                  setActiveBrowseId(null);
                  fetchRequests();
                } catch (error) {
                  console.error('Failed to deny:', error);
                }
              }}
              className="text-red-700 bg-red-100 hover:bg-red-200 text-xs font-medium px-3 py-1.5 rounded-md transition-colors"
            >
              {t.deny}
            </button>
          </div>
        </div>
      )}

      {/* Browse error */}
      {browseError && !loading && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-sm text-red-700 font-mono">{browseError}</p>
        </div>
      )}

      {/* Preview content */}
      {previewContent && !loading && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <div className="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
            <div className="text-sm text-gray-600">
              {previewContent.lines} lines {previewContent.size > 0 && `· ${formatBytes(previewContent.size)}`}
            </div>
            <button onClick={() => setPreviewContent(null)} className="text-xs text-gray-500 hover:text-gray-700">{t.closePreview}</button>
          </div>
          <pre className="bg-gray-900 text-green-300 p-4 text-xs font-mono overflow-auto max-h-[500px] whitespace-pre-wrap">
            {previewContent.content}
          </pre>
        </div>
      )}

      {/* Download link */}
      {downloadUrl && !loading && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 text-center space-y-3">
          <p className="text-sm text-gray-700">{t.resultSuccess}</p>
          <a
            href={downloadUrl}
            download
            className="inline-flex items-center gap-2 px-6 py-2.5 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
            </svg>
            {t.download}
          </a>
          <button onClick={() => setDownloadUrl(null)} className="block mx-auto text-xs text-gray-400 hover:text-gray-600 mt-2">{t.closePreview}</button>
        </div>
      )}

      {/* File listing */}
      {filteredEntries && !loading && !previewContent && !downloadUrl && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          {/* Search + meta */}
          <div className="px-4 py-3 border-b border-gray-200 flex flex-col sm:flex-row sm:items-center gap-3">
            <div className="relative flex-1">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
              </svg>
              <input
                type="text"
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                placeholder={(t as any).searchPlaceholder}
                className="w-full pl-9 pr-3 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-indigo-500 focus:border-indigo-500"
              />
            </div>
            <div className="text-xs text-gray-500 flex-shrink-0">
              {browseMeta?.dir_count ?? 0} dirs, {browseMeta?.file_count ?? 0} files
              {browseMeta?.total_size != null && ` · ${formatBytes(browseMeta.total_size)}`}
              {browseMeta?.truncated && ' (truncated)'}
            </div>
          </div>

          {/* Parent dir link */}
          {currentPath !== '/' && (
            <button
              onClick={() => browsePath(parentPath(pathSegments))}
              className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-indigo-600 hover:bg-indigo-50 border-b border-gray-100 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 15 3 9m0 0 6-6M3 9h12a6 6 0 0 1 0 12h-3" />
              </svg>
              ..
            </button>
          )}

          {/* Entries */}
          {filteredEntries.length === 0 ? (
            <div className="p-8 text-center text-gray-400 text-sm">{t.noFiles}</div>
          ) : (
            <div className="divide-y divide-gray-100">
              {/* Directories first, then files */}
              {filteredEntries
                .sort((a, b) => {
                  if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
                  return a.name.localeCompare(b.name);
                })
                .map((entry, i) => (
                  <div
                    key={i}
                    className="flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 transition-colors group"
                  >
                    {/* Icon + name (clickable) */}
                    <button
                      onClick={() => entry.is_dir ? handleDirClick(entry) : handleFileClick(entry)}
                      className="flex items-center gap-2 min-w-0 flex-1 text-left"
                    >
                      <span className="flex-shrink-0 text-base">
                        {entry.is_dir ? '📁' : '📄'}
                      </span>
                      <span className={`text-sm truncate ${
                        entry.is_dir ? 'text-indigo-600 font-medium' : 'text-gray-700'
                      }`}>
                        {entry.name}
                      </span>
                    </button>

                    {/* Size */}
                    <span className="text-xs text-gray-400 font-mono flex-shrink-0 w-20 text-right">
                      {entry.is_dir ? '-' : formatBytes(entry.size_bytes)}
                    </span>

                    {/* Modified */}
                    <span className="text-xs text-gray-400 flex-shrink-0 w-36 text-right hidden sm:block">
                      {entry.modified_at ? new Date(entry.modified_at).toLocaleString() : '-'}
                    </span>

                    {/* File actions */}
                    {!entry.is_dir && (
                      <div className="flex gap-1 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={() => handleFileClick(entry)}
                          className="text-xs text-indigo-600 hover:text-indigo-800 px-2 py-1 rounded hover:bg-indigo-50"
                        >
                          {t.preview}
                        </button>
                        <button
                          onClick={() => requestDownload(joinPath(currentPath, entry.name))}
                          className="text-xs text-indigo-600 hover:text-indigo-800 px-2 py-1 rounded hover:bg-indigo-50"
                        >
                          {t.download}
                        </button>
                      </div>
                    )}
                  </div>
                ))}
            </div>
          )}
        </div>
      )}

      {/* Initial state — no browse yet */}
      {selectedDeviceId && !loading && !browseEntries && !browseError && !previewContent && !downloadUrl && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center text-gray-400">
          <svg className="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" strokeWidth={1} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z" />
          </svg>
          <p className="text-sm">{(t as any).clickDirToBrowse}</p>
        </div>
      )}

      {/* Request history (collapsible) */}
      {showHistory && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <div className="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
            <h3 className="text-sm font-medium text-gray-700">{(t as any).historyTitle}</h3>
            {requests.length > 0 && (
              <button
                onClick={async () => {
                  if (!confirm((t as any).confirmClear)) return;
                  try {
                    await api.delete(`/tenants/${params.tenantId}/file-access-requests`);
                    fetchRequests();
                  } catch (error) {
                    console.error('Failed to clear history:', error);
                  }
                }}
                className="text-xs text-red-500 hover:text-red-700 font-medium px-2 py-1 rounded hover:bg-red-50 transition-colors"
              >
                {(t as any).clearHistory}
              </button>
            )}
          </div>
          {requests.length === 0 ? (
            <div className="p-6 text-center text-gray-400 text-sm">{t.noRequests}</div>
          ) : (
            <>
            <div className="divide-y divide-gray-200 max-h-80 overflow-y-auto">
              {requests.slice(0, 50).map(req => (
                <div key={req.id} className="flex items-center gap-3 px-4 py-3 text-sm hover:bg-gray-50">
                  <span className="text-gray-900 flex-shrink-0 w-28 truncate">{getDeviceName(req.device_id)}</span>
                  <span className="text-gray-600 font-mono truncate flex-1" title={req.path}>{req.path}</span>
                  <span className="text-gray-500 flex-shrink-0 w-16">{req.action}</span>
                  <span className={`px-2 py-0.5 text-xs font-semibold rounded-full flex-shrink-0 ${statusColor(req.status)}`}>
                    {req.status}
                  </span>
                  {req.status === 'pending' && (
                    <div className="flex gap-1 flex-shrink-0">
                      <button onClick={() => handleApprove(req.id)} className="text-green-600 hover:text-green-800 text-xs font-medium px-1.5 py-0.5 rounded hover:bg-green-50">{t.approve}</button>
                      <button onClick={() => handleDeny(req.id)} className="text-red-600 hover:text-red-800 text-xs font-medium px-1.5 py-0.5 rounded hover:bg-red-50">{t.deny}</button>
                    </div>
                  )}
                  <span className="text-xs text-gray-400 flex-shrink-0">{new Date(req.created_at).toLocaleTimeString()}</span>
                </div>
              ))}
            </div>
            {requests.length > 0 && (
              <div className="flex justify-between items-center px-4 py-3 border-t border-gray-200 text-xs text-gray-500">
                <div>共 {requestPagination.total} 条记录</div>
                <div className="flex items-center space-x-2">
                  <span>每页显示：</span>
                  <select 
                    value={requestPagination.pageSize} 
                    onChange={(e) => handleRequestPageSizeChange(parseInt(e.target.value))}
                    className="border rounded-md px-1 py-0.5 text-xs"
                  >
                    <option value="10">10条</option>
                    <option value="20">20条</option>
                    <option value="50">50条</option>
                    <option value="100">100条</option>
                  </select>
                  <button 
                    onClick={() => handleRequestPageChange(requestPagination.page - 1)}
                    disabled={requestPagination.page === 1}
                    className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                  >
                    上一页
                  </button>
                  <span>{requestPagination.page}</span>
                  <button 
                    onClick={() => handleRequestPageChange(requestPagination.page + 1)}
                    disabled={requestPagination.page * requestPagination.pageSize >= requestPagination.total}
                    className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
