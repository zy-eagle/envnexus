"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';

interface ModelProfile {
  id: string;
  name: string;
  provider: string;
  base_url: string;
  model_name: string;
  api_key?: string;
  status: string;
  params_json?: string;
  secret_mode?: string;
}

const PROVIDER_DEFAULTS: Record<string, { base_url: string; model_name: string }> = {
  openai:    { base_url: 'https://api.openai.com/v1',       model_name: 'gpt-4o' },
  anthropic: { base_url: 'https://api.anthropic.com/v1',    model_name: 'claude-3-5-sonnet-20240620' },
  deepseek:  { base_url: 'https://api.deepseek.com/v1',    model_name: 'deepseek-chat' },
  local:     { base_url: 'http://localhost:11434/v1',       model_name: 'llama3' },
};

export default function ModelProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('modelProfiles', lang);
  const ct = useDict('common', lang);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [profiles, setProfiles] = useState<ModelProfile[]>([]);
  const [loading, setLoading] = useState(true);
  
  // Pagination state
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const [formError, setFormError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    name: '',
    provider: 'openai',
    base_url: PROVIDER_DEFAULTS.openai.base_url,
    model_name: PROVIDER_DEFAULTS.openai.model_name,
    api_key: '',
    params_json: '{}',
    secret_mode: 'env'
  });

  const openCreateModal = () => {
    setEditingId(null);
    setFormError(null);
    setFormData({
      name: '',
      provider: 'openai',
      base_url: PROVIDER_DEFAULTS.openai.base_url,
      model_name: PROVIDER_DEFAULTS.openai.model_name,
      api_key: '',
      params_json: '{}',
      secret_mode: 'env'
    });
    setIsModalOpen(true);
  };

  const openEditModal = (profile: ModelProfile) => {
    setEditingId(profile.id);
    setFormError(null);
    setFormData({
      name: profile.name,
      provider: profile.provider,
      base_url: profile.base_url,
      model_name: profile.model_name,
      api_key: profile.api_key || '',
      params_json: profile.params_json || '{}',
      secret_mode: profile.secret_mode || 'env'
    });
    setIsModalOpen(true);
  };

  const fetchProfiles = async (page: number = 1, pageSize: number = 10) => {
    setLoading(true);
    try {
      const data = await api.get<{ items: ModelProfile[]; total: number } | ModelProfile[]>(
        `/tenants/${params.tenantId}/model-profiles?page=${page}&page_size=${pageSize}`
      );
      
      if (Array.isArray(data)) {
        setProfiles(data);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setProfiles(data.items);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setProfiles([]);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch (error) {
      console.error('Failed to fetch profiles:', error);
      setProfiles([]);
      setPagination(prev => ({
        ...prev,
        total: 0
      }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProfiles(pagination.page, pagination.pageSize);
  }, [params.tenantId, pagination.page, pagination.pageSize]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    try {
      if (editingId) {
        await api.put(`/tenants/${params.tenantId}/model-profiles/${editingId}`, formData);
      } else {
        await api.post(`/tenants/${params.tenantId}/model-profiles`, formData);
      }
      setIsModalOpen(false);
      fetchProfiles(pagination.page, pagination.pageSize);
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
      await api.delete(`/tenants/${params.tenantId}/model-profiles/${id}`);
      fetchProfiles(pagination.page, pagination.pageSize);
    } catch (error) {
      console.error('Error deleting profile:', error);
    }
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
          onClick={openCreateModal}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.addProfile}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{editingId ? t.editTitle : t.createTitle}</h2>
            <form onSubmit={handleSave} className="space-y-4">
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
                  value={formData.name}
                  onChange={e => setFormData({...formData, name: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.provider}</label>
                <select 
                  value={formData.provider}
                  onChange={e => {
                    const newProvider = e.target.value;
                    const defaults = PROVIDER_DEFAULTS[newProvider] || PROVIDER_DEFAULTS.openai;
                    setFormData({...formData, provider: newProvider, base_url: defaults.base_url, model_name: defaults.model_name});
                  }}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="openai">OpenAI</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="deepseek">DeepSeek</option>
                  <option value="local">Local</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.baseURL}</label>
                <input 
                  type="text" 
                  required
                  value={formData.base_url}
                  onChange={e => setFormData({...formData, base_url: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.modelName}</label>
                <select 
                  required
                  value={formData.model_name}
                  onChange={e => setFormData({...formData, model_name: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                >
                  {formData.provider === 'openai' && (
                    <>
                      <option value="gpt-4o">gpt-4o</option>
                      <option value="gpt-4-turbo">gpt-4-turbo</option>
                      <option value="gpt-4">gpt-4</option>
                      <option value="gpt-3.5-turbo">gpt-3.5-turbo</option>
                    </>
                  )}
                  {formData.provider === 'anthropic' && (
                    <>
                      <option value="claude-3-5-sonnet-20240620">claude-3-5-sonnet</option>
                      <option value="claude-3-opus-20240229">claude-3-opus</option>
                      <option value="claude-3-sonnet-20240229">claude-3-sonnet</option>
                      <option value="claude-3-haiku-20240307">claude-3-haiku</option>
                    </>
                  )}
                  {formData.provider === 'deepseek' && (
                    <>
                      <option value="deepseek-chat">deepseek-chat</option>
                      <option value="deepseek-reasoner">deepseek-reasoner</option>
                    </>
                  )}
                  {formData.provider === 'local' && (
                    <>
                      <option value="llama3">llama3</option>
                      <option value="qwen2">qwen2</option>
                      <option value="mistral">mistral</option>
                    </>
                  )}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.apiKey}</label>
                <input 
                  type="password" 
                  value={formData.api_key}
                  onChange={e => setFormData({...formData, api_key: e.target.value})}
                  placeholder={t.apiKeyPlaceholder}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
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
          <>
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.name}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.provider}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.modelName}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.status}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {profiles.map((profile) => (
                <tr key={profile.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{profile.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{profile.provider}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{profile.model_name}</td>
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
          {profiles.length > 0 && (
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
          </>
        )}
      </div>
    </div>
  );
}
