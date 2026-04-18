"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface ToolPermission {
  id: string;
  tool_name: string;
  role_id: string | null;
  allowed: boolean;
  max_risk: string;
}

export default function ToolPermissionsPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('toolPermissions', lang);
  const ct = useDict('common', lang);
  const [perms, setPerms] = useState<ToolPermission[]>([]);
  const [loading, setLoading] = useState(true);
  
  // Pagination state
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });
  
  // Modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [currentPermId, setCurrentPermId] = useState<string>('');
  const [newPerm, setNewPerm] = useState({
    tool_name: '',
    role_id: null as string | null,
    allowed: true,
    max_risk: 'L0'
  });

  const fetchPerms = async (page: number = 1, pageSize: number = 10) => {
    setLoading(true);
    try {
      const data = await api.get<{ items: ToolPermission[]; total: number } | ToolPermission[]>(
        `/tenants/${params.tenantId}/tool-permissions?page=${page}&page_size=${pageSize}`
      );
      
      if (Array.isArray(data)) {
        setPerms(data);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setPerms(data.items);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setPerms([]);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch {
      setPerms([]);
      setPagination(prev => ({
        ...prev,
        total: 0
      }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchPerms(pagination.page, pagination.pageSize); }, [params.tenantId, pagination.page, pagination.pageSize]);

  const handleDelete = async (id: string) => {
    if (!confirm(ct.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/tool-permissions/${id}`);
      fetchPerms(pagination.page, pagination.pageSize);
    } catch (error) {
      console.error('Failed to delete permission:', error);
    }
  };

  const handleAddPermission = async () => {
    setIsSubmitting(true);
    try {
      if (isEditMode) {
        await api.put(`/tenants/${params.tenantId}/tool-permissions/${currentPermId}`, newPerm);
      } else {
        await api.post(`/tenants/${params.tenantId}/tool-permissions`, newPerm);
      }
      setIsModalOpen(false);
      setIsEditMode(false);
      setCurrentPermId('');
      setNewPerm({
        tool_name: '',
        role_id: null,
        allowed: true,
        max_risk: 'L0'
      });
      fetchPerms(pagination.page, pagination.pageSize);
    } catch (error) {
      console.error('Failed to save permission:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleEditPermission = (perm: ToolPermission) => {
    setIsEditMode(true);
    setCurrentPermId(perm.id);
    setNewPerm({
      tool_name: perm.tool_name,
      role_id: perm.role_id,
      allowed: perm.allowed,
      max_risk: perm.max_risk
    });
    setIsModalOpen(true);
  };

  // Pagination handlers
  const handlePageChange = (newPage: number) => {
    setPagination(prev => ({ ...prev, page: newPage }));
  };

  const handlePageSizeChange = (newPageSize: number) => {
    setPagination(prev => ({ ...prev, page: 1, pageSize: newPageSize }));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm font-medium"
        >
          添加工具权限
        </button>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : perms.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noPermissions}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.toolName}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.roleId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.allowed}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.maxRisk}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {perms.map(p => (
                <tr key={p.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{p.tool_name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{p.role_id || 'All'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      p.allowed ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {p.allowed ? 'Yes' : 'No'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{p.max_risk || '-'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                    <button onClick={() => handleEditPermission(p)} className="text-blue-600 hover:text-blue-900 text-xs font-medium mr-3">{ct.edit}</button>
                    <button onClick={() => handleDelete(p.id)} className="text-red-600 hover:text-red-900 text-xs font-medium">{ct.delete}</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
            {perms.length > 0 && (
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

      {/* Add Permission Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-lg p-6 max-w-md w-full">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{isEditMode ? '编辑工具权限' : '添加工具权限'}</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">工具名称</label>
                <input 
                  type="text" 
                  value={newPerm.tool_name}
                  onChange={(e) => setNewPerm({ ...newPerm, tool_name: e.target.value })}
                  className="w-full border rounded-md px-3 py-2"
                  placeholder="例如：shell_exec, file_download"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">角色 ID (可选)</label>
                <input 
                  type="text" 
                  value={newPerm.role_id || ''}
                  onChange={(e) => setNewPerm({ ...newPerm, role_id: e.target.value || null })}
                  className="w-full border rounded-md px-3 py-2"
                  placeholder="留空表示所有角色"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">是否允许</label>
                <select 
                  value={newPerm.allowed}
                  onChange={(e) => setNewPerm({ ...newPerm, allowed: e.target.value === 'true' })}
                  className="w-full border rounded-md px-3 py-2"
                >
                  <option value="true">是</option>
                  <option value="false">否</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">最高风险等级</label>
                <select 
                  value={newPerm.max_risk}
                  onChange={(e) => setNewPerm({ ...newPerm, max_risk: e.target.value })}
                  className="w-full border rounded-md px-3 py-2"
                >
                  <option value="L0">L0 (只读，无副作用)</option>
                  <option value="L1">L1 (低风险写操作)</option>
                  <option value="L2">L2 (中风险操作或敏感只读访问)</option>
                  <option value="L3">L3 (高风险操作)</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end space-x-3 mt-6">
              <button 
                onClick={() => {
                  setIsModalOpen(false);
                  setNewPerm({
                    tool_name: '',
                    role_id: null,
                    allowed: true,
                    max_risk: 'L0'
                  });
                }}
                className="px-4 py-2 border rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                取消
              </button>
              <button 
                onClick={handleAddPermission}
                disabled={isSubmitting || !newPerm.tool_name}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm font-medium disabled:opacity-50"
              >
                {isSubmitting ? '添加中...' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
