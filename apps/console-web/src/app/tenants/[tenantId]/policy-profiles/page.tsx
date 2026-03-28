"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';

interface PolicyFields {
  default_mode: string;
  allow_write_tools: boolean;
}

function parsePolicyJSON(json: string): PolicyFields {
  try {
    const obj = JSON.parse(json);
    return {
      default_mode: obj.default_mode || 'read_only',
      allow_write_tools: obj.allow_write_tools !== false,
    };
  } catch {
    return { default_mode: 'read_only', allow_write_tools: true };
  }
}

function toPolicyJSON(fields: PolicyFields): string {
  return JSON.stringify({
    default_mode: fields.default_mode,
    allow_write_tools: fields.allow_write_tools,
  });
}

export default function PolicyProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('policyProfiles', lang);
  const ct = useDict('common', lang);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [profiles, setProfiles] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const [formError, setFormError] = useState<string | null>(null);
  const [formName, setFormName] = useState('');
  const [formStatus, setFormStatus] = useState('active');
  const [policyFields, setPolicyFields] = useState<PolicyFields>({
    default_mode: 'read_only',
    allow_write_tools: true,
  });

  const fetchProfiles = async () => {
    try {
      const data = await api.get<{ items: any[] }>(`/tenants/${params.tenantId}/policy-profiles`);
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
    setFormError(null);
    setFormName('');
    setFormStatus('active');
    setPolicyFields({ default_mode: 'read_only', allow_write_tools: true });
    setIsModalOpen(true);
  };

  const openEditModal = (profile: any) => {
    setEditingId(profile.id);
    setFormError(null);
    setFormName(profile.name);
    setFormStatus(profile.status || 'active');
    setPolicyFields(parsePolicyJSON(profile.policy_json));
    setIsModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    const body = {
      name: formName,
      policy_json: toPolicyJSON(policyFields),
      status: formStatus,
    };
    try {
      if (editingId) {
        await api.put(`/tenants/${params.tenantId}/policy-profiles/${editingId}`, body);
      } else {
        await api.post(`/tenants/${params.tenantId}/policy-profiles`, body);
      }
      setIsModalOpen(false);
      fetchProfiles();
    } catch (error) {
      if (error instanceof APIError && error.status === 409) {
        setFormError(ct.duplicateName);
      } else {
        setFormError(t.saveFailed);
      }
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/policy-profiles/${id}`);
      fetchProfiles();
    } catch (error) {
      console.error('Error deleting profile:', error);
    }
  };

  const getPolicySummary = (json: string) => {
    const fields = parsePolicyJSON(json);
    return fields;
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
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
            <h2 className="text-xl font-semibold mb-5">{editingId ? t.editTitle : t.createTitle}</h2>
            <form onSubmit={handleSave} className="space-y-5">
              {formError && (
                <div className="flex items-center gap-2 rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
                  <svg className="h-4 w-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd"/></svg>
                  {formError}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.name}</label>
                <input 
                  type="text" 
                  required
                  value={formName}
                  onChange={e => setFormName(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.defaultMode}</label>
                <p className="text-xs text-gray-500 mb-3">{t.defaultModeDesc}</p>
                <div className="space-y-2">
                  <label
                    className={`flex items-start p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                      policyFields.default_mode === 'read_only'
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <input
                      type="radio"
                      name="default_mode"
                      value="read_only"
                      checked={policyFields.default_mode === 'read_only'}
                      onChange={() => setPolicyFields({ ...policyFields, default_mode: 'read_only' })}
                      className="mt-0.5 text-blue-600 focus:ring-blue-500"
                    />
                    <div className="ml-3">
                      <span className="block text-sm font-medium text-gray-900">{t.modeReadOnly}</span>
                      <span className="block text-xs text-gray-500 mt-0.5">{t.modeReadOnlyDesc}</span>
                    </div>
                  </label>
                  <label
                    className={`flex items-start p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                      policyFields.default_mode === 'full'
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <input
                      type="radio"
                      name="default_mode"
                      value="full"
                      checked={policyFields.default_mode === 'full'}
                      onChange={() => setPolicyFields({ ...policyFields, default_mode: 'full' })}
                      className="mt-0.5 text-blue-600 focus:ring-blue-500"
                    />
                    <div className="ml-3">
                      <span className="block text-sm font-medium text-gray-900">{t.modeFull}</span>
                      <span className="block text-xs text-gray-500 mt-0.5">{t.modeFullDesc}</span>
                    </div>
                  </label>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">{t.allowWriteTools}</label>
                    <p className="text-xs text-gray-500 mt-0.5">{t.allowWriteToolsDesc}</p>
                  </div>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={policyFields.allow_write_tools}
                    onClick={() => setPolicyFields({ ...policyFields, allow_write_tools: !policyFields.allow_write_tools })}
                    className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                      policyFields.allow_write_tools ? 'bg-blue-600' : 'bg-gray-200'
                    }`}
                  >
                    <span
                      className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                        policyFields.allow_write_tools ? 'translate-x-5' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>
              </div>

              {editingId && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{ct.status}</label>
                  <select 
                    value={formStatus}
                    onChange={e => setFormStatus(e.target.value)}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="active">{t.statusActive}</option>
                    <option value="archived">{t.statusArchived}</option>
                  </select>
                </div>
              )}

              <div className="flex justify-end space-x-3 pt-2">
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
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.summaryMode}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.summaryWrite}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.status}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {profiles.map((profile) => {
                const summary = getPolicySummary(profile.policy_json);
                return (
                  <tr key={profile.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{profile.name}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        summary.default_mode === 'read_only'
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-amber-100 text-amber-800'
                      }`}>
                        {summary.default_mode === 'read_only' ? t.defaultModeLabel : t.defaultModeFullLabel}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        summary.allow_write_tools
                          ? 'bg-green-100 text-green-800'
                          : 'bg-red-100 text-red-800'
                      }`}>
                        {summary.allow_write_tools ? t.enabled : t.disabled}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        profile.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {profile.status === 'active' ? t.statusActive : t.statusArchived}
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
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
