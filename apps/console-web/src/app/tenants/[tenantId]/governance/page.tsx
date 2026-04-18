"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface Summary {
  total_baselines: number;
  total_drifts: number;
  unresolved_drifts: number;
}

interface Baseline {
  id: string;
  device_id: string;
  tenant_id: string;
  snapshot_json: string;
  captured_at: string;
}

interface Drift {
  id: string;
  device_id: string;
  tenant_id: string;
  baseline_id: string | null;
  drift_type: string;
  key_name: string;
  expected_value: string | null;
  actual_value: string | null;
  severity: string;
  detected_at: string;
  resolved_at: string | null;
}

const severityColors: Record<string, string> = {
  low: 'bg-blue-100 text-blue-800',
  medium: 'bg-yellow-100 text-yellow-800',
  high: 'bg-orange-100 text-orange-800',
  critical: 'bg-red-100 text-red-800',
};

export default function GovernancePage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('governance', lang);
  const [tab, setTab] = useState<'drifts' | 'baselines'>('drifts');
  const [summary, setSummary] = useState<Summary | null>(null);
  const [baselines, setBaselines] = useState<Baseline[]>([]);
  const [drifts, setDrifts] = useState<Drift[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [deviceFilter, setDeviceFilter] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [unresolvedOnly, setUnresolvedOnly] = useState(false);
  
  // Pagination state for drifts
  const [driftPagination, setDriftPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });
  
  // Pagination state for baselines
  const [baselinePagination, setBaselinePagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const fetchSummary = useCallback(async () => {
    try {
      const data = await api.get<Summary>(`/tenants/${params.tenantId}/governance/summary`);
      setSummary(data);
    } catch {
      setSummary(null);
    }
  }, [params.tenantId]);

  const fetchBaselines = useCallback(async (page: number = 1, pageSize: number = 10) => {
    try {
      const parts: string[] = [];
      if (deviceFilter) parts.push(`device_id=${deviceFilter}`);
      parts.push(`page=${page}`);
      parts.push(`page_size=${pageSize}`);
      const qs = parts.length > 0 ? `?${parts.join('&')}` : '';
      const data = await api.get<{ items: Baseline[]; total: number } | Baseline[]>(
        `/tenants/${params.tenantId}/governance/baselines${qs}`
      );
      
      if (Array.isArray(data)) {
        setBaselines(data);
        setBaselinePagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setBaselines(data.items);
        setBaselinePagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setBaselines([]);
        setBaselinePagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch {
      setBaselines([]);
      setBaselinePagination(prev => ({
        ...prev,
        total: 0
      }));
    }
  }, [params.tenantId, deviceFilter]);

  const fetchDrifts = useCallback(async (page: number = 1, pageSize: number = 10) => {
    const parts: string[] = [];
    if (deviceFilter) parts.push(`device_id=${deviceFilter}`);
    if (severityFilter) parts.push(`severity=${severityFilter}`);
    if (unresolvedOnly) parts.push('unresolved=true');
    parts.push(`page=${page}`);
    parts.push(`page_size=${pageSize}`);
    const qs = parts.length > 0 ? `?${parts.join('&')}` : '';
    try {
      const data = await api.get<{ items: Drift[]; total: number } | Drift[]>(
        `/tenants/${params.tenantId}/governance/drifts${qs}`
      );
      
      if (Array.isArray(data)) {
        setDrifts(data);
        setDriftPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setDrifts(data.items);
        setDriftPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setDrifts([]);
        setDriftPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch {
      setDrifts([]);
      setDriftPagination(prev => ({
        ...prev,
        total: 0
      }));
    }
  }, [params.tenantId, deviceFilter, severityFilter, unresolvedOnly]);

  useEffect(() => {
    setLoading(true);
    Promise.all([fetchSummary(), tab === 'baselines' ? fetchBaselines(baselinePagination.page, baselinePagination.pageSize) : fetchDrifts(driftPagination.page, driftPagination.pageSize)])
      .finally(() => setLoading(false));
  }, [fetchSummary, fetchBaselines, fetchDrifts, tab, baselinePagination.page, baselinePagination.pageSize, driftPagination.page, driftPagination.pageSize]);

  const handleResolve = async (driftId: string) => {
    try {
      await api.post(`/tenants/${params.tenantId}/governance/drifts/${driftId}/resolve`);
      fetchDrifts();
      fetchSummary();
    } catch (err) {
      console.error('Failed to resolve drift:', err);
    }
  };

  const handleFilter = (e: React.FormEvent) => {
    e.preventDefault();
    if (tab === 'baselines') fetchBaselines(baselinePagination.page, baselinePagination.pageSize);
    else fetchDrifts(driftPagination.page, driftPagination.pageSize);
  };

  const severityLabel = (s: string) => {
    const map: Record<string, string> = { low: t.severityLow, medium: t.severityMedium, high: t.severityHigh, critical: t.severityCritical };
    return map[s] || s;
  };

  // Pagination handlers for drifts
  const handleDriftPageChange = (newPage: number) => {
    setDriftPagination(prev => ({ ...prev, page: newPage }));
  };

  const handleDriftPageSizeChange = (newPageSize: number) => {
    setDriftPagination(prev => ({ ...prev, page: 1, pageSize: newPageSize }));
  };

  // Pagination handlers for baselines
  const handleBaselinePageChange = (newPage: number) => {
    setBaselinePagination(prev => ({ ...prev, page: newPage }));
  };

  const handleBaselinePageSizeChange = (newPageSize: number) => {
    setBaselinePagination(prev => ({ ...prev, page: 1, pageSize: newPageSize }));
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      {/* Summary Cards */}
      {summary && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5">
            <div className="text-sm font-medium text-gray-500">{t.totalBaselines}</div>
            <div className="mt-1 text-2xl font-semibold text-gray-900">{summary.total_baselines}</div>
          </div>
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5">
            <div className="text-sm font-medium text-gray-500">{t.totalDrifts}</div>
            <div className="mt-1 text-2xl font-semibold text-gray-900">{summary.total_drifts}</div>
          </div>
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5">
            <div className="text-sm font-medium text-gray-500">{t.unresolvedDrifts}</div>
            <div className={`mt-1 text-2xl font-semibold ${summary.unresolved_drifts > 0 ? 'text-red-600' : 'text-green-600'}`}>
              {summary.unresolved_drifts}
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setTab('drifts')}
            className={`py-3 px-1 border-b-2 text-sm font-medium transition-colors ${
              tab === 'drifts'
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            {t.driftsTab}
            {summary && summary.unresolved_drifts > 0 && (
              <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-red-100 text-red-700">{summary.unresolved_drifts}</span>
            )}
          </button>
          <button
            onClick={() => setTab('baselines')}
            className={`py-3 px-1 border-b-2 text-sm font-medium transition-colors ${
              tab === 'baselines'
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            {t.baselinesTab}
          </button>
        </nav>
      </div>

      {/* Filters */}
      <form onSubmit={handleFilter} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <input
            type="text"
            placeholder={t.filterDevice}
            value={deviceFilter}
            onChange={e => setDeviceFilter(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500 w-64"
          />
          {tab === 'drifts' && (
            <>
              <select
                value={severityFilter}
                onChange={e => setSeverityFilter(e.target.value)}
                className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">{t.filterSeverity}</option>
                <option value="low">{t.severityLow}</option>
                <option value="medium">{t.severityMedium}</option>
                <option value="high">{t.severityHigh}</option>
                <option value="critical">{t.severityCritical}</option>
              </select>
              <label className="flex items-center space-x-2 text-sm text-gray-600">
                <input
                  type="checkbox"
                  checked={unresolvedOnly}
                  onChange={e => setUnresolvedOnly(e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span>{t.showUnresolved}</span>
              </label>
            </>
          )}
          <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700">
            Filter
          </button>
        </div>
      </form>

      {/* Content */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{t.loading}</div>
        ) : tab === 'drifts' ? (
          drifts.length === 0 ? (
            <div className="p-12 text-center">
              <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-green-100 text-green-600 mb-3">
                <svg className="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
              </div>
              <p className="text-gray-500">{t.noDrifts}</p>
            </div>
          ) : (
            <>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.deviceId}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.driftType}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.keyName}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.expectedValue}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.actualValue}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.severity}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.status}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.detectedAt}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {drifts.map(d => (
                  <tr key={d.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-xs font-mono text-gray-500">{d.device_id.slice(0, 12)}...</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{d.drift_type}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-700">{d.key_name}</td>
                    <td className="px-6 py-4 text-sm text-gray-500 max-w-[160px] truncate" title={d.expected_value || '—'}>{d.expected_value || '—'}</td>
                    <td className="px-6 py-4 text-sm text-gray-500 max-w-[160px] truncate" title={d.actual_value || '—'}>{d.actual_value || '—'}</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-0.5 text-xs font-semibold rounded-full ${severityColors[d.severity] || 'bg-gray-100 text-gray-800'}`}>
                        {severityLabel(d.severity)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {d.resolved_at ? (
                        <span className="px-2 py-0.5 text-xs font-semibold rounded-full bg-green-100 text-green-800">{t.resolved}</span>
                      ) : (
                        <span className="px-2 py-0.5 text-xs font-semibold rounded-full bg-red-100 text-red-800">{t.unresolved}</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(d.detected_at).toLocaleString()}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                      {!d.resolved_at && (
                        <button
                          onClick={() => handleResolve(d.id)}
                          className="text-blue-600 hover:text-blue-900 font-medium"
                        >
                          {t.resolve}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {drifts.length > 0 && (
              <div className="flex justify-between items-center px-6 py-4 border-t border-gray-200">
                <div className="flex items-center space-x-4">
                  <div className="text-sm text-gray-500">
                    共 {driftPagination.total} 条记录
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-sm text-gray-500">每页显示：</span>
                    <select 
                      value={driftPagination.pageSize} 
                      onChange={(e) => {
                        handleDriftPageSizeChange(parseInt(e.target.value));
                      }}
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
                    onClick={() => {
                      handleDriftPageChange(driftPagination.page - 1);
                    }}
                    disabled={driftPagination.page === 1}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    上一页
                  </button>
                  <span className="text-sm">{driftPagination.page}</span>
                  <button 
                    onClick={() => {
                      handleDriftPageChange(driftPagination.page + 1);
                    }}
                    disabled={driftPagination.page * driftPagination.pageSize >= driftPagination.total}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
            </>
          )
        ) : (
          baselines.length === 0 ? (
            <div className="p-12 text-center">
              <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-blue-100 text-blue-600 mb-3">
                <svg className="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25Z" /></svg>
              </div>
              <p className="text-gray-500">{t.noBaselines}</p>
            </div>
          ) : (
            <>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.deviceId}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.capturedAt}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{t.snapshot}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {baselines.map(b => (
                  <>
                    <tr key={b.id}>
                      <td className="px-6 py-4 whitespace-nowrap text-xs font-mono text-gray-500">{b.device_id.slice(0, 12)}...</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(b.captured_at).toLocaleString()}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <button
                          onClick={() => setExpandedId(expandedId === b.id ? null : b.id)}
                          className="text-blue-600 hover:text-blue-900 font-medium"
                        >
                          {expandedId === b.id ? 'Hide' : t.viewSnapshot}
                        </button>
                      </td>
                    </tr>
                    {expandedId === b.id && (
                      <tr key={`${b.id}-snap`}>
                        <td colSpan={3} className="px-6 py-4 bg-gray-50">
                          <pre className="text-xs text-gray-700 overflow-x-auto whitespace-pre-wrap max-h-80 overflow-y-auto">
                            {(() => {
                              try { return JSON.stringify(JSON.parse(b.snapshot_json), null, 2); }
                              catch { return b.snapshot_json; }
                            })()}
                          </pre>
                        </td>
                      </tr>
                    )}
                  </>
                ))}
              </tbody>
            </table>
            {baselines.length > 0 && (
              <div className="flex justify-between items-center px-6 py-4 border-t border-gray-200">
                <div className="flex items-center space-x-4">
                  <div className="text-sm text-gray-500">
                    共 {baselinePagination.total} 条记录
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-sm text-gray-500">每页显示：</span>
                    <select 
                      value={baselinePagination.pageSize} 
                      onChange={(e) => {
                        handleBaselinePageSizeChange(parseInt(e.target.value));
                      }}
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
                    onClick={() => {
                      handleBaselinePageChange(baselinePagination.page - 1);
                    }}
                    disabled={baselinePagination.page === 1}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    上一页
                  </button>
                  <span className="text-sm">{baselinePagination.page}</span>
                  <button 
                    onClick={() => {
                      handleBaselinePageChange(baselinePagination.page + 1);
                    }}
                    disabled={baselinePagination.page * baselinePagination.pageSize >= baselinePagination.total}
                    className="px-3 py-1 border rounded-md text-sm disabled:opacity-50"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
            </>
          )
        )}
      </div>
    </div>
  );
}
