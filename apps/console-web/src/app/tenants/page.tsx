"use client";

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useAuth } from '@/lib/auth/AuthContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

export default function TenantsPage() {
  const { lang } = useLanguage();
  const { switchTenant, activeTenantId } = useAuth();
  const router = useRouter();
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
      const data = await api.get<any[]>('/tenants');
      setTenants(Array.isArray(data) ? data : []);
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
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 tracking-tight">{t.title}</h1>
          <p className="mt-1 text-sm text-slate-500">{lang === 'zh' ? '管理您的租户组织' : 'Manage your tenant organizations'}</p>
        </div>
        <button onClick={openCreateModal} className="btn-primary">
          <svg className="w-4 h-4 mr-1.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
          {t.createBtn}
        </button>
      </div>

      {/* Modal */}
      {isModalOpen && (
        <div className="modal-overlay">
          <div className="modal-content max-w-md">
            <h2 className="text-lg font-semibold text-slate-900 mb-5">{editingId ? t.editModalTitle : t.modalTitle}</h2>
            <form onSubmit={handleSaveTenant} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1.5">{t.tenantName}</label>
                <input 
                  type="text" 
                  required
                  className="input-field"
                  value={newTenantName}
                  onChange={(e) => setNewTenantName(e.target.value)}
                  placeholder="e.g. Acme Corp"
                />
              </div>
              {!editingId && (
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">{t.tenantSlug}</label>
                  <input 
                    type="text" 
                    required
                    pattern="[a-z0-9-]+"
                    title="Only lowercase letters, numbers, and hyphens are allowed"
                    className="input-field"
                    value={newTenantSlug}
                    onChange={(e) => setNewTenantSlug(e.target.value)}
                    placeholder="e.g. acme-corp"
                  />
                </div>
              )}
              {editingId && (
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">{ct.status}</label>
                  <select 
                    className="select-field"
                    value={newTenantStatus}
                    onChange={(e) => setNewTenantStatus(e.target.value)}
                  >
                    <option value="active">Active</option>
                    <option value="suspended">Suspended</option>
                    <option value="archived">Archived</option>
                  </select>
                </div>
              )}
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setIsModalOpen(false)} className="btn-secondary">
                  {ct.cancel}
                </button>
                <button type="submit" disabled={isSubmitting} className="btn-primary">
                  {isSubmitting ? '...' : (editingId ? ct.save : ct.create)}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      
      {/* Table */}
      <div className="card overflow-hidden">
        {loading ? (
          <div className="p-12 text-center">
            <div className="inline-block animate-spin rounded-full h-6 w-6 border-2 border-slate-200 border-t-indigo-600 mb-3"></div>
            <p className="text-sm text-slate-400">{ct.loading}</p>
          </div>
        ) : tenants.length === 0 ? (
          <div className="p-12 text-center">
            <svg className="w-10 h-10 text-slate-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" strokeWidth={1} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21M3 3h12m-.75 4.5H21m-3.75 3h.008v.008h-.008v-.008Zm0 3h.008v.008h-.008v-.008Zm0 3h.008v.008h-.008v-.008Z" /></svg>
            <p className="text-sm text-slate-400">{t.noTenants}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead>
                <tr className="border-b border-slate-100 bg-slate-50/50">
                  <th className="table-header">{t.name}</th>
                  <th className="table-header">{t.slug}</th>
                  <th className="table-header">{ct.status}</th>
                  <th className="table-header">{t.createdAt}</th>
                  <th className="table-header text-right">{ct.actions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {tenants.map((tenant: any) => (
                  <tr key={tenant.id} className="hover:bg-slate-50/50 transition-colors">
                    <td className="table-cell">
                      <div className="flex items-center gap-3">
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${
                          activeTenantId === tenant.id
                            ? 'bg-indigo-100 text-indigo-600'
                            : 'bg-slate-100 text-slate-500'
                        }`}>
                          <span className="text-xs font-bold">{tenant.name.charAt(0).toUpperCase()}</span>
                        </div>
                        <span className="font-medium text-slate-900">{tenant.name}</span>
                      </div>
                    </td>
                    <td className="table-cell">
                      <code className="text-xs bg-slate-100 text-slate-600 px-2 py-0.5 rounded-md font-mono">{tenant.slug}</code>
                    </td>
                    <td className="table-cell">
                      <span className={
                        tenant.status === 'active' ? 'badge-success' :
                        tenant.status === 'suspended' ? 'badge-warning' : 'badge-neutral'
                      }>
                        {tenant.status || 'active'}
                      </span>
                    </td>
                    <td className="table-cell text-slate-500">
                      {new Date(tenant.created_at).toLocaleDateString()}
                    </td>
                    <td className="table-cell text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button onClick={() => openEditModal(tenant)} className="btn-ghost text-xs">
                          {ct.edit}
                        </button>
                        <button onClick={() => handleDeleteTenant(tenant.id)} className="btn-danger text-xs">
                          {ct.delete}
                        </button>
                        <button
                          onClick={() => { switchTenant(tenant.id); router.push(`/tenants/${tenant.id}/health`); }}
                          className={`inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-150 ${
                            activeTenantId === tenant.id
                              ? 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/20'
                              : 'bg-indigo-50 text-indigo-700 hover:bg-indigo-100'
                          }`}
                        >
                          {activeTenantId === tenant.id ? (
                            <>
                              <div className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                              {t.current || 'Current'}
                            </>
                          ) : (
                            <>
                              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="m4.5 19.5 15-15m0 0H8.25m11.25 0v11.25" /></svg>
                              {t.manage}
                            </>
                          )}
                        </button>
                      </div>
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
