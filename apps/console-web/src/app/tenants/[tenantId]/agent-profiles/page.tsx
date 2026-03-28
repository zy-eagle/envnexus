"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

export default function AgentProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('agentProfiles', lang);
  const ct = useDict('common', lang);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [profiles, setProfiles] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const [formData, setFormData] = useState({
    name: '',
    model_profile_id: '',
    policy_profile_id: '',
    capabilities_json: '{\n  "local_ui": true,\n  "repair_enabled": true\n}',
    update_channel: 'stable',
    status: 'active'
  });

  const fetchProfiles = async () => {
    try {
      const data = await api.get<{ items: any[] }>(`/tenants/${params.tenantId}/agent-profiles`);
      setProfiles(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch profiles:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProfiles();
  }, [params.tenantId]);

  const openCreateModal = () => {
    setEditingId(null);
    setFormData({
      name: '',
      model_profile_id: '',
      policy_profile_id: '',
      capabilities_json: '{\n  "local_ui": true,\n  "repair_enabled": true\n}',
      update_channel: 'stable',
      status: 'active'
    });
    setIsModalOpen(true);
  };

  const openEditModal = (profile: any) => {
    setEditingId(profile.id);
    setFormData({
      name: profile.name,
      model_profile_id: profile.model_profile_id,
      policy_profile_id: profile.policy_profile_id,
      capabilities_json: profile.capabilities_json || '{}',
      update_channel: profile.update_channel || 'stable',
      status: profile.status || 'active'
    });
    setIsModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingId) {
        await api.put(`/tenants/${params.tenantId}/agent-profiles/${editingId}`, formData);
      } else {
        await api.post(`/tenants/${params.tenantId}/agent-profiles`, formData);
      }
      setIsModalOpen(false);
      fetchProfiles();
    } catch (error) {
      console.error('Error saving profile:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/agent-profiles/${id}`);
      fetchProfiles();
    } catch (error) {
      console.error('Error deleting profile:', error);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button 
          onClick={openCreateModal}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.addProfile}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6">
            <h2 className="text-xl font-semibold mb-4">{editingId ? t.editTitle : t.createTitle}</h2>
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.name}</label>
                <input 
                  type="text" 
                  required
                  value={formData.name}
                  onChange={e => setFormData({...formData, name: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t.modelProfile}</label>
                  <input 
                    type="text" 
                    required
                    value={formData.model_profile_id}
                    onChange={e => setFormData({...formData, model_profile_id: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t.policyProfile}</label>
                  <input 
                    type="text" 
                    required
                    value={formData.policy_profile_id}
                    onChange={e => setFormData({...formData, policy_profile_id: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.channel}</label>
                <select 
                  value={formData.update_channel}
                  onChange={e => setFormData({...formData, update_channel: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="stable">Stable</option>
                  <option value="beta">Beta</option>
                  <option value="dev">Dev</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.capabilities}</label>
                <textarea 
                  required
                  rows={4}
                  value={formData.capabilities_json}
                  onChange={e => setFormData({...formData, capabilities_json: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
              {editingId && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{ct.status}</label>
                  <select 
                    value={formData.status}
                    onChange={e => setFormData({...formData, status: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="active">Active</option>
                    <option value="archived">Archived</option>
                  </select>
                </div>
              )}
              <div className="flex justify-end space-x-3 mt-6">
                <button 
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
                >
                  {ct.cancel}
                </button>
                <button 
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                >
                  {editingId ? ct.save : ct.create}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : profiles.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noProfiles}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.name}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.channel}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.status}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {profiles.map((profile: any) => (
                <tr key={profile.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{profile.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{profile.update_channel}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                      {profile.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button 
                      onClick={() => openEditModal(profile)}
                      className="text-blue-600 hover:text-blue-900 mr-4"
                    >
                      {ct.edit}
                    </button>
                    <button 
                      onClick={() => handleDelete(profile.id)}
                      className="text-red-600 hover:text-red-900"
                    >
                      {ct.delete}
                    </button>
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
