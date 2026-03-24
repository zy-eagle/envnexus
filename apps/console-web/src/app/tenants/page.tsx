"use client";

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

export default function TenantsPage() {
  const { lang } = useLanguage();
  const t = useDict('tenants', lang);
  const ct = useDict('common', lang);
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

  const fetchTenants = async () => {
    setLoading(true);
    try {
      const data = await api.get<{ items: any[] }>('/tenants');
      setTenants(data.items || []);
    } catch (err) {
      console.error('Failed to fetch tenants', err);
      setTenants([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTenants();
  }, []);

  const handleSaveTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const body = editingId 
        ? { name: newTenantName, status: newTenantStatus }
        : { name: newTenantName, slug: newTenantSlug };

      if (editingId) {
        await api.put(`/tenants/${editingId}`, body);
      } else {
        await api.post('/tenants', body);
      }
      setIsModalOpen(false);
      setNewTenantName('');
      setNewTenantSlug('');
      fetchTenants();
    } catch (err: any) {
      console.error(err);
      alert(`Failed to save tenant: ${err.message || 'Unknown error'}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteTenant = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${id}`);
      fetchTenants();
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
                    <label className="block text-sm font-medium text-gray-700">{ct.status}</label>
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
                  {ct.cancel}
                </button>
                <button 
                  type="submit" 
                  disabled={isSubmitting}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                >
                  {isSubmitting ? t.creating : (editingId ? ct.save : ct.create)}
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
            <p>{ct.loading}</p>
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
                    {ct.status}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.createdAt}
                  </th>
                  <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {ct.actions}
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
                        {ct.edit}
                      </button>
                      <button 
                        onClick={() => handleDeleteTenant(tenant.id)}
                        className="text-red-600 hover:text-red-900 mr-4"
                      >
                        {ct.delete}
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
