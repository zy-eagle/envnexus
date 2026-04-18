"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface BatchTask {
  id: string;
  device_group_id: string;
  command_task_id: string;
  strategy: string;
  total_devices: number;
  completed: number;
  failed: number;
  status: string;
  created_at: string;
}

export default function BatchTasksPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('batchTasks', lang);
  const ct = useDict('common', lang);
  const [tasks, setTasks] = useState<BatchTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const fetchTasks = async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const currentPage = page || pagination.page;
      const currentPageSize = pageSize || pagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/batch-tasks?page=${currentPage}&page_size=${currentPageSize}`);
      setTasks(Array.isArray(data) ? data : (data?.items ?? []));
      setPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch {
      setTasks([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchTasks(); }, [params.tenantId]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'pending': return 'bg-gray-100 text-gray-800';
      case 'executing': return 'bg-yellow-100 text-yellow-800';
      case 'completed': return 'bg-green-100 text-green-800';
      case 'partial_done': return 'bg-orange-100 text-orange-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const handlePageChange = (newPage: number) => {
    fetchTasks(newPage, pagination.pageSize);
  };

  const handlePageSizeChange = (newPageSize: number) => {
    fetchTasks(1, newPageSize);
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : tasks.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noTasks}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.groupId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.strategy}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.progress}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.createdAt}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {tasks.map(task => (
                <tr key={task.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{task.device_group_id.substring(0, 8)}...</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{task.strategy}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">
                    <div className="flex items-center gap-2">
                      <div className="w-24 bg-gray-200 rounded-full h-2">
                        <div
                          className="bg-indigo-600 h-2 rounded-full"
                          style={{ width: `${task.total_devices > 0 ? ((task.completed + task.failed) / task.total_devices * 100) : 0}%` }}
                        />
                      </div>
                      <span className="text-xs">{task.completed + task.failed}/{task.total_devices}</span>
                      {task.failed > 0 && <span className="text-xs text-red-500">({task.failed} failed)</span>}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColor(task.status)}`}>
                      {task.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(task.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!loading && tasks.length > 0 && (
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
        )}
      </div>
    </div>
  );
}
