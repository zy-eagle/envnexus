"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface DeviceGroup {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
}

export default function DeviceGroupsPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('deviceGroups', lang);
  const ct = useDict('common', lang);
  const [groups, setGroups] = useState<DeviceGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState('');
  const [formDesc, setFormDesc] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const data = await api.get<{ items: DeviceGroup[] }>(`/tenants/${params.tenantId}/device-groups`);
      setGroups(Array.isArray(data) ? data : (data as any)?.items || []);
    } catch {
      setGroups([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchGroups(); }, [params.tenantId]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName) return;
    setCreating(true);
    try {
      await api.post(`/tenants/${params.tenantId}/device-groups`, { name: formName, description: formDesc });
      setFormName('');
      setFormDesc('');
      setShowForm(false);
      fetchGroups();
    } catch (error) {
      console.error('Failed to create device group:', error);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(ct.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/device-groups/${id}`);
      fetchGroups();
    } catch (error) {
      console.error('Failed to delete device group:', error);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700"
        >
          {ct.create}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 space-y-4">
          <input
            type="text"
            placeholder={ct.name}
            value={formName}
            onChange={e => setFormName(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="text"
            placeholder={t.description}
            value={formDesc}
            onChange={e => setFormDesc(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <div className="flex gap-2">
            <button type="submit" disabled={creating} className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">{ct.save}</button>
            <button type="button" onClick={() => setShowForm(false)} className="border border-gray-300 px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-50">{ct.cancel}</button>
          </div>
        </form>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : groups.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noGroups}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.name}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.description}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.createdAt}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {groups.map(g => (
                <tr key={g.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{g.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{g.description || '-'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(g.created_at).toLocaleString()}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                    <button onClick={() => handleDelete(g.id)} className="text-red-600 hover:text-red-900 text-xs font-medium">{ct.delete}</button>
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
