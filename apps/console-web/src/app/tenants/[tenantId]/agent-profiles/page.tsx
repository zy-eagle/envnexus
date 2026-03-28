"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface CapabilityFields {
  local_ui: boolean;
}

function parseCapabilities(json: string): CapabilityFields {
  try {
    const obj = JSON.parse(json);
    return {
      local_ui: obj.local_ui ?? true,
    };
  } catch {
    return { local_ui: true };
  }
}

function toCapabilitiesJSON(fields: CapabilityFields): string {
  return JSON.stringify(fields);
}

function ToggleSwitch({ checked, onChange, label, description }: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
  description: string;
}) {
  return (
    <div className="flex items-center justify-between py-3">
      <div className="pr-4">
        <span className="block text-sm font-medium text-gray-700">{label}</span>
        <span className="block text-xs text-gray-500 mt-0.5">{description}</span>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
          checked ? 'bg-blue-600' : 'bg-gray-200'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
            checked ? 'translate-x-5' : 'translate-x-0'
          }`}
        />
      </button>
    </div>
  );
}

export default function AgentProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('agentProfiles', lang);
  const ct = useDict('common', lang);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [profiles, setProfiles] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [modelProfiles, setModelProfiles] = useState<any[]>([]);
  const [policyProfiles, setPolicyProfiles] = useState<any[]>([]);

  const [formData, setFormData] = useState({
    name: '',
    model_profile_id: '',
    policy_profile_id: '',
    capabilities_json: toCapabilitiesJSON({ local_ui: true }),
    update_channel: 'stable',
    status: 'active'
  });

  const [capFields, setCapFields] = useState<CapabilityFields>({ local_ui: true });

  const updateCapField = useCallback((field: keyof CapabilityFields, value: boolean) => {
    const next = { ...capFields, [field]: value };
    setCapFields(next);
    setFormData(prev => ({ ...prev, capabilities_json: toCapabilitiesJSON(next) }));
  }, [capFields]);

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

  const fetchRelatedProfiles = async () => {
    try {
      const [models, policies] = await Promise.all([
        api.get<any[]>(`/tenants/${params.tenantId}/model-profiles`),
        api.get<any[]>(`/tenants/${params.tenantId}/policy-profiles`),
      ]);
      setModelProfiles(Array.isArray(models) ? models : []);
      setPolicyProfiles(Array.isArray(policies) ? policies : []);
    } catch (error) {
      console.error('Failed to fetch related profiles:', error);
    }
  };

  useEffect(() => {
    fetchProfiles();
    fetchRelatedProfiles();
  }, [params.tenantId]);

  const openCreateModal = () => {
    setEditingId(null);
    const defaultCap: CapabilityFields = { local_ui: true };
    setCapFields(defaultCap);
    setFormData({
      name: '',
      model_profile_id: '',
      policy_profile_id: '',
      capabilities_json: toCapabilitiesJSON(defaultCap),
      update_channel: 'stable',
      status: 'active'
    });
    setIsModalOpen(true);
  };

  const openEditModal = (profile: any) => {
    setEditingId(profile.id);
    const cap = parseCapabilities(profile.capabilities_json || '{}');
    setCapFields(cap);
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
                  <select
                    required
                    value={formData.model_profile_id}
                    onChange={e => setFormData({...formData, model_profile_id: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="">{t.selectModel}</option>
                    {modelProfiles.map((mp: any) => (
                      <option key={mp.id} value={mp.id}>{mp.name} ({mp.provider})</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t.policyProfile}</label>
                  <select
                    required
                    value={formData.policy_profile_id}
                    onChange={e => setFormData({...formData, policy_profile_id: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="">{t.selectPolicy}</option>
                    {policyProfiles.map((pp: any) => (
                      <option key={pp.id} value={pp.id}>{pp.name}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.channel}</label>
                <select 
                  value={formData.update_channel}
                  onChange={e => setFormData({...formData, update_channel: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="stable">{t.channelStable}</option>
                  <option value="beta">{t.channelBeta}</option>
                  <option value="dev">{t.channelDev}</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">{t.capabilities}</label>
                <div className="border border-gray-200 rounded-lg px-4 divide-y divide-gray-100">
                  <ToggleSwitch
                    checked={capFields.local_ui}
                    onChange={v => updateCapField('local_ui', v)}
                    label={t.capLocalUI}
                    description={t.capLocalUIDesc}
                  />
                </div>
              </div>

              {editingId && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{ct.status}</label>
                  <select 
                    value={formData.status}
                    onChange={e => setFormData({...formData, status: e.target.value})}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="active">{t.statusActive}</option>
                    <option value="archived">{t.statusArchived}</option>
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
