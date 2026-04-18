"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

interface RoleOption {
  id: string;
  name: string;
}

export default function TenantUsersPage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = params?.tenantId;
  const { lang } = useLanguage();
  const t = useDict("users", lang);
  const ct = useDict("common", lang);

  const [error, setError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ email: "", display_name: "", password: "", status: "active" });
  const [roleIds, setRoleIds] = useState<string[]>([]);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [roleOptions, setRoleOptions] = useState<RoleOption[]>([]);
  
  // User list state
  const [users, setUsers] = useState<any[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  
  // Pagination state
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const fetchRoleOptions = useCallback(async () => {
    if (!tenantId) return;
    setRolesLoading(true);
    try {
      const data = await api.get<{ items: RoleOption[] }>(`/tenants/${tenantId}/roles`);
      const items = Array.isArray(data?.items) ? data.items : [];
      setRoleOptions(items.map((r) => ({ id: r.id, name: r.name })));
    } catch (e) {
      console.error("Failed to fetch roles:", e);
      setRoleOptions([]);
    } finally {
      setRolesLoading(false);
    }
  }, [tenantId]);

  const fetchUsers = useCallback(async (page: number = 1, pageSize: number = 10) => {
    if (!tenantId) return;
    setUsersLoading(true);
    try {
      const data = await api.get<{ items: any[]; total: number } | any[]>(
        `/tenants/${tenantId}/users?page=${page}&page_size=${pageSize}`
      );
      
      if (Array.isArray(data)) {
        setUsers(data);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setUsers(data.items);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setUsers([]);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch (e) {
      console.error("Failed to fetch users:", e);
      setUsers([]);
      setPagination(prev => ({
        ...prev,
        total: 0
      }));
    } finally {
      setUsersLoading(false);
    }
  }, [tenantId]);

  const openCreate = () => {
    setForm({ email: "", display_name: "", password: "", status: "active" });
    setRoleIds([]);
    setError(null);
    setModalOpen(true);
    void fetchRoleOptions();
  };

  const title = useMemo(() => t.createTitle, [t]);

  // Fetch users on component load and when pagination changes
  useEffect(() => {
    fetchUsers(pagination.page, pagination.pageSize);
  }, [fetchUsers, pagination.page, pagination.pageSize]);

  const toggleRole = (id: string) => {
    setRoleIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  };

  const save = async () => {
    if (!tenantId) return;
    setSaving(true);
    setError(null);
    try {
      await api.post(`/tenants/${tenantId}/users`, {
        email: form.email,
        display_name: form.display_name,
        password: form.password,
        status: form.status,
        role_ids: roleIds,
      });
      setModalOpen(false);
      fetchUsers(pagination.page, pagination.pageSize);
    } catch (e) {
      console.error("Failed to save user:", e);
      if (e instanceof APIError) {
        setError(e.message || t.saveFailed);
      } else {
        setError(t.saveFailed);
      }
    } finally {
      setSaving(false);
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
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">{t.title}</h1>
          <p className="text-sm text-slate-500 mt-1">{t.pageHint}</p>
        </div>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700"
        >
          {t.addUser}
        </button>
      </div>

      {error && !modalOpen && (
        <div className="mb-4 text-sm text-red-700 bg-red-50 border border-red-100 rounded-lg px-4 py-3">{error}</div>
      )}

      {/* User List */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {usersLoading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : users.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noUsers}</div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.email}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.displayName}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.status}</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.rolesField}</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {users.map((user) => (
                    <tr key={user.id}>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{user.email}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{user.display_name || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${user.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                          {user.status === 'active' ? t.statusActive : t.statusDisabled}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {user.roles && user.roles.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {user.roles.map((role: any) => (
                              <span key={role.id} className="px-2 py-0.5 bg-gray-100 text-gray-800 rounded text-xs">
                                {role.name}
                              </span>
                            ))}
                          </div>
                        ) : (
                          '-'
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {users.length > 0 && (
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

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
            <h2 className="text-xl font-semibold mb-5">{title}</h2>
            {error && <div className="mb-4 text-sm text-red-700 bg-red-50 border border-red-100 rounded-md px-3 py-2">{error}</div>}
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.email}</label>
                <input
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.displayName}</label>
                <input
                  value={form.display_name}
                  onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.password}</label>
                <input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.status}</label>
                <select
                  value={form.status}
                  onChange={(e) => setForm({ ...form, status: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="active">{t.statusActive}</option>
                  <option value="disabled">{t.statusDisabled}</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.rolesField}</label>
                <p className="text-xs text-slate-500 mb-2">{t.rolesHint}</p>
                <div className="border border-slate-200 rounded-md max-h-40 overflow-y-auto p-2">
                  {rolesLoading ? (
                    <p className="text-sm text-slate-500 text-center py-2">{ct.loading}</p>
                  ) : roleOptions.length === 0 ? (
                    <p className="text-sm text-slate-500 text-center py-2">{t.noRolesInTenant}</p>
                  ) : (
                    roleOptions.map((r) => (
                      <label
                        key={r.id}
                        className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-slate-50 cursor-pointer text-sm"
                      >
                        <input
                          type="checkbox"
                          checked={roleIds.includes(r.id)}
                          onChange={() => toggleRole(r.id)}
                          className="rounded text-indigo-600"
                        />
                        <span className="text-slate-800">{r.name}</span>
                      </label>
                    ))
                  )}
                </div>
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-6">
              <button
                onClick={() => setModalOpen(false)}
                className="px-4 py-2 border border-slate-200 text-slate-700 rounded-md text-sm font-medium hover:bg-slate-50"
                disabled={saving}
              >
                {ct.cancel}
              </button>
              <button
                onClick={save}
                className="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
                disabled={saving}
              >
                {saving ? ct.loading : ct.save}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
