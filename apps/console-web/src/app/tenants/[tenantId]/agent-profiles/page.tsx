"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Agent Profiles",
    addProfile: "Add Profile",
    noProfiles: "No agent profiles configured for this tenant.",
    name: "Name",
    modelProfile: "Model Profile ID",
    policyProfile: "Policy Profile ID",
    capabilities: "Capabilities (JSON)",
    channel: "Update Channel",
    status: "Status",
    actions: "Actions",
    createTitle: "Create Agent Profile",
    editTitle: "Edit Agent Profile",
    cancel: "Cancel",
    create: "Create",
    save: "Save",
    edit: "Edit",
    delete: "Delete",
    confirmDelete: "Are you sure you want to delete this profile?",
  },
  zh: {
    title: "Agent 配置",
    addProfile: "添加配置",
    noProfiles: "该租户暂无 Agent 配置。",
    name: "名称",
    modelProfile: "模型配置 ID",
    policyProfile: "策略配置 ID",
    capabilities: "能力清单 (JSON)",
    channel: "更新通道",
    status: "状态",
    actions: "操作",
    createTitle: "创建 Agent 配置",
    editTitle: "编辑 Agent 配置",
    cancel: "取消",
    create: "创建",
    save: "保存",
    edit: "编辑",
    delete: "删除",
    confirmDelete: "确定要删除此配置吗？",
  }
};

export default function AgentProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];
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
      const token = localStorage.getItem('token');
      const res = await fetch(`/api/v1/tenants/${params.tenantId}/agent-profiles`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setProfiles(data.data || []);
      }
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
      const token = localStorage.getItem('token');
      const url = editingId 
        ? `/api/v1/tenants/${params.tenantId}/agent-profiles/${editingId}`
        : `/api/v1/tenants/${params.tenantId}/agent-profiles`;
      const method = editingId ? 'PUT' : 'POST';

      const res = await fetch(url, {
        method,
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData),
      });
      if (res.ok) {
        setIsModalOpen(false);
        fetchProfiles();
      } else {
        alert('Failed to save profile');
      }
    } catch (error) {
      console.error('Error saving profile:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      const token = localStorage.getItem('token');
      const res = await fetch(`/api/v1/tenants/${params.tenantId}/agent-profiles/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        fetchProfiles();
      } else {
        alert('Failed to delete profile');
      }
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
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t.status}</label>
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
                  {t.cancel}
                </button>
                <button 
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                >
                  {editingId ? t.save : t.create}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">Loading...</div>
        ) : profiles.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noProfiles}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.name}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.channel}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.status}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{t.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {profiles.map((profile) => (
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
                      {t.edit}
                    </button>
                    <button 
                      onClick={() => handleDelete(profile.id)}
                      className="text-red-600 hover:text-red-900"
                    >
                      {t.delete}
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
