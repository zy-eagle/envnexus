"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Model Profiles",
    addProfile: "Add Profile",
    noProfiles: "No model profiles configured for this tenant.",
    name: "Name",
    provider: "Provider",
    modelName: "Model Name",
    status: "Status",
    actions: "Actions",
    createTitle: "Create Model Profile",
    baseURL: "Base URL",
    paramsJSON: "Params (JSON)",
    secretMode: "Secret Mode",
    cancel: "Cancel",
    create: "Create",
  },
  zh: {
    title: "模型配置",
    addProfile: "添加配置",
    noProfiles: "该租户暂无模型配置。",
    name: "名称",
    provider: "提供商",
    modelName: "模型名称",
    status: "状态",
    actions: "操作",
    createTitle: "创建模型配置",
    baseURL: "基础 URL",
    paramsJSON: "参数 (JSON)",
    secretMode: "密钥模式",
    cancel: "取消",
    create: "创建",
  }
};

interface ModelProfile {
  id: string;
  name: string;
  provider: string;
  base_url: string;
  model_name: string;
  status: string;
}

export default function ModelProfilesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [profiles, setProfiles] = useState<ModelProfile[]>([]);
  const [loading, setLoading] = useState(true);

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    model_name: 'gpt-4',
    params_json: '{}',
    secret_mode: 'env'
  });

  const fetchProfiles = async () => {
    try {
      const token = localStorage.getItem('token');
      const res = await fetch(`/api/v1/tenants/${params.tenantId}/model-profiles`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
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

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const token = localStorage.getItem('token');
      const res = await fetch(`/api/v1/tenants/${params.tenantId}/model-profiles`, {
        method: 'POST',
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
        alert('Failed to create profile');
      }
    } catch (error) {
      console.error('Error creating profile:', error);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.addProfile}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{t.createTitle}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
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
                  onChange={e => setFormData({...formData, provider: e.target.value})}
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
                <input 
                  type="text" 
                  required
                  value={formData.model_name}
                  onChange={e => setFormData({...formData, model_name: e.target.value})}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500" 
                />
              </div>
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
                  {t.create}
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
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.provider}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.modelName}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.status}</th>
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
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
