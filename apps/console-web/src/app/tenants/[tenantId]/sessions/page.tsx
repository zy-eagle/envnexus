"use client";

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';


function SessionsContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = useDict('sessions', lang);
  const ct = useDict('common', lang);
  const [sessions, setSessions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSessions, setSelectedSessions] = useState<string[]>([]);
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const fetchSessions = async (page?: number, pageSize?: number) => {
    try {
      const currentPage = page || pagination.page;
      const currentPageSize = pageSize || pagination.pageSize;
      const data = await api.get<{ items: any[], total: number }>(`/tenants/${tenantId}/sessions?page=${currentPage}&page_size=${currentPageSize}`);
      console.log('Fetched sessions data:', data);
      if (Array.isArray(data.items)) {
        data.items.forEach((session, index) => {
          console.log(`Session ${index} started_at:`, session.started_at, typeof session.started_at);
        });
      }
      setSessions(Array.isArray(data.items) ? data.items : []);
      setPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data.total || 0
      }));
    } catch (error) {
      console.error('Failed to fetch sessions:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAbort = async (sessionId: string) => {
    if (!confirm(t.confirmAbort)) return;
    try {
      await api.post(`/sessions/${sessionId}/abort`, { reason: "User requested" });
      fetchSessions();
    } catch (error) {
      console.error('Failed to abort session:', error);
    }
  };

  const handleSessionSelect = (sessionId: string) => {
    setSelectedSessions(prev => {
      if (prev.includes(sessionId)) {
        return prev.filter(id => id !== sessionId);
      } else {
        return [...prev, sessionId];
      }
    });
  };

  const handleSelectAll = () => {
    if (selectedSessions.length === sessions.length) {
      setSelectedSessions([]);
    } else {
      setSelectedSessions(sessions.map(session => session.id));
    }
  };

  const handleBatchDelete = async () => {
    if (selectedSessions.length === 0) return;
    if (!confirm('确定要删除选中的会话吗？')) return;
    try {
      await api.post(`/tenants/${tenantId}/sessions/batch-delete`, { session_ids: selectedSessions });
      setSelectedSessions([]);
      fetchSessions();
    } catch (error) {
      console.error('Failed to delete sessions:', error);
    }
  };

  const handlePageChange = (newPage: number) => {
    fetchSessions(newPage, pagination.pageSize);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    fetchSessions(1, newPageSize);
  };

  useEffect(() => { fetchSessions(); }, [tenantId]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'created': case 'attached': return 'bg-blue-100 text-blue-800';
      case 'diagnosing': case 'executing': return 'bg-yellow-100 text-yellow-800';
      case 'awaiting_approval': return 'bg-orange-100 text-orange-800';
      case 'completed': return 'bg-green-100 text-green-800';
      case 'aborted': case 'expired': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const isActive = (status: string) => !['completed', 'aborted', 'expired'].includes(status);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        {selectedSessions.length > 0 && (
          <button 
            onClick={handleBatchDelete}
            className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700"
          >
            批量删除 ({selectedSessions.length})
          </button>
        )}
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : sessions.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noSessions}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  <input 
                    type="checkbox" 
                    checked={sessions.length > 0 && selectedSessions.length === sessions.length}
                    onChange={handleSelectAll}
                    className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.transport}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.initiator}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.startedAt}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {sessions.map((session: any) => (
                <tr key={session.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <input 
                      type="checkbox" 
                      checked={selectedSessions.includes(session.id)}
                      onChange={() => handleSessionSelect(session.id)}
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 font-mono text-xs">{session.device_id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{session.transport}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColor(session.status)}`}>
                      {session.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{session.initiator_type}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {session.started_at ? (() => {
                      const date = new Date(session.started_at);
                      return isNaN(date.getTime()) ? 'Invalid Date' : date.toLocaleString();
                    })() : 'No Date'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <Link
                      href={`/tenants/${tenantId}/sessions/${session.id}`}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      {t.view}
                    </Link>
                    {isActive(session.status) && (
                      <button onClick={() => handleAbort(session.id)} className="text-red-600 hover:text-red-900">{t.abort}</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      {!loading && sessions.length > 0 && (
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
  );
}

export default function SessionsPage({ params }: { params: { tenantId: string } }) {
  return <SessionsContent tenantId={params.tenantId} />;
}
