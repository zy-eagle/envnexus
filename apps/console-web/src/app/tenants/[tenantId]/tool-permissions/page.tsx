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

  const fetchPerms = async () => {
    setLoading(true);
    try {
      const data = await api.get<{ items: ToolPermission[] }>(`/tenants/${params.tenantId}/tool-permissions`);
      setPerms(Array.isArray(data) ? data : (data as any)?.items || []);
    } catch {
      setPerms([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchPerms(); }, [params.tenantId]);

  const handleDelete = async (id: string) => {
    if (!confirm(ct.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/tool-permissions/${id}`);
      fetchPerms();
    } catch (error) {
      console.error('Failed to delete permission:', error);
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

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
                    <button onClick={() => handleDelete(p.id)} className="text-red-600 hover:text-red-900 text-xs font-medium">{ct.delete}</button>
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
