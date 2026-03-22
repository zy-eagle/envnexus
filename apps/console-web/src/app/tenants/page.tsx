"use client";

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: { title: "Tenants", createBtn: "Create Tenant", loading: "Loading tenants...", noTenants: "No tenants found. Create one to get started.", name: "Name", slug: "Slug", status: "Status", createdAt: "Created At", actions: "Actions", manage: "Manage", edit: "Edit", delete: "Delete", modalTitle: "Create New Tenant", editModalTitle: "Edit Tenant", tenantName: "Tenant Name", tenantSlug: "Tenant Slug", cancel: "Cancel", create: "Create", save: "Save", creating: "Creating...", confirmDelete: "Are you sure you want to delete this tenant?" },
  zh: { title: "租户管理", createBtn: "创建租户", loading: "加载中...", noTenants: "未找到租户，请创建一个。", name: "名称", slug: "标识 (Slug)", status: "状态", createdAt: "创建时间", actions: "操作", manage: "管理", edit: "编辑", delete: "删除", modalTitle: "创建新租户", editModalTitle: "编辑租户", tenantName: "租户名称", tenantSlug: "租户标识 (Slug)", cancel: "取消", create: "创建", save: "保存", creating: "创建中...", confirmDelete: "确定要删除此租户吗？" }
};

export default function TenantsPage() {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [tenants, setTenants] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [newTenantName, setNewTenantName] = useState('');
  const [newTenantSlug, setNewTenantSlug] = useState('');
  const [newTenantStatus, setNewTenantStatus] = useState('active');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const openCreateModal = () => {
    setEditingId(null);
    setNewTenantName('');
    setNewTenantSlug('');
    setNewTenantStatus('active');
    setIsModalOpen(true);
  };

  const openEditModal = (tenant: any) => {
    setEditingId(tenant.id);
    setNewTenantName(tenant.name);
    setNewTenantSlug(tenant.slug);
    setNewTenantStatus(tenant.status || 'active');
    setIsModalOpen(true);
  };

  const fetchTenants = () => {
    setLoading(true);
    const token = localStorage.getItem('token');
    fetch('/api/v1/tenants', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
      .then((res) => res.json())
      .then((data) => {
        if (data.data) {
          setTenants(data.data);
        } else {
          setTenants([]);
        }
        setLoading(false);
      })
      .catch((err) => {
        console.error('Failed to fetch tenants', err);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchTenants();
  }, []);

  const handleSaveTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const token = localStorage.getItem('token');
      const url = editingId ? `/api/v1/tenants/${editingId}` : '/api/v1/tenants';
      const method = editingId ? 'PUT' : 'POST';
      const body = editingId 
        ? { name: newTenantName, status: newTenantStatus }
        : { name: newTenantName, slug: newTenantSlug };

      const res = await fetch(url, {
        method,
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        setIsModalOpen(false);
        setNewTenantName('');
        setNewTenantSlug('');
        fetchTenants();
      } else {
        const errorData = await res.json();
        alert(`Failed to save tenant: ${errorData.error || 'Unknown error'}`);
      }
    } catch (err) {
      console.error(err);
      alert('Network error while saving tenant');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteTenant = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      const token = localStorage.getItem('token');
      const res = await fetch(`/api/v1/tenants/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      if (res.ok) {
        fetchTenants();
      } else {
        alert('Failed to delete tenant');
      }
    } catch (err) {
      console.error(err);
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
          {t.createBtn}
        </button>
      </div>

      {/* Create/Edit Tenant Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{editingId ? t.editModalTitle : t.modalTitle}</h2>
            <form onSubmit={handleSaveTenant}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">{t.tenantName}</label>
                  <input 
                    type="text" 
                    required
                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                    value={newTenantName}
                    onChange={(e) => setNewTenantName(e.target.value)}
                    placeholder="e.g. Acme Corp"
                  />
                </div>
                {!editingId && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700">{t.tenantSlug}</label>
                    <input 
                      type="text" 
                      required
                      pattern="[a-z0-9-]+"
                      title="Only lowercase letters, numbers, and hyphens are allowed"
                      className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      value={newTenantSlug}
                      onChange={(e) => setNewTenantSlug(e.target.value)}
                      placeholder="e.g. acme-corp"
                    />
                  </div>
                )}
                {editingId && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700">{t.status}</label>
                    <select 
                      className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                      value={newTenantStatus}
                      onChange={(e) => setNewTenantStatus(e.target.value)}
                    >
                      <option value="active">Active</option>
                      <option value="suspended">Suspended</option>
                      <option value="archived">Archived</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="mt-6 flex justify-end space-x-3">
                <button 
                  type="button" 
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                  {t.cancel}
                </button>
                <button 
                  type="submit" 
                  disabled={isSubmitting}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                >
                  {isSubmitting ? t.creating : (editingId ? t.save : t.create)}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4"></div>
            <p>{t.loading}</p>
          </div>
        ) : tenants.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            <p>{t.noTenants}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.name}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.slug}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.status}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.createdAt}
                  </th>
                  <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.actions}
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {tenants.map((tenant: any) => (
                  <tr key={tenant.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {tenant.name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {tenant.slug}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        tenant.status === 'active' 
                          ? 'bg-green-100 text-green-800' 
                          : 'bg-gray-100 text-gray-800'
                      }`}>
                        {tenant.status || 'active'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(tenant.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button 
                        onClick={() => openEditModal(tenant)}
                        className="text-blue-600 hover:text-blue-900 mr-4"
                      >
                        {t.edit}
                      </button>
                      <button 
                        onClick={() => handleDeleteTenant(tenant.id)}
                        className="text-red-600 hover:text-red-900 mr-4"
                      >
                        {t.delete}
                      </button>
                      <Link href={`/tenants/${tenant.id}/devices`} className="text-gray-600 hover:text-gray-900">
                        {t.manage}
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
