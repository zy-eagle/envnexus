"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

interface RoleItem {
  id: string;
  tenant_id: string;
  name: string;
  permissions: string[];
  status: string;
  created_at: string;
  updated_at: string;
}

function permLabelKey(perm: string): string {
  return "perm_" + perm.replace(/:/g, "_").replace(/-/g, "_");
}

const PERM_CATEGORY_ORDER = ["tenants", "users", "roles", "profiles", "devices", "sessions", "approvals", "audit", "packages", "webhooks", "metrics", "licenses", "command", "file", "other"];

function groupPermissions(perms: string[]): Record<string, string[]> {
  const m: Record<string, string[]> = {};
  for (const p of perms) {
    const prefix = p.includes(":") ? p.slice(0, p.indexOf(":")) : "other";
    if (!m[prefix]) m[prefix] = [];
    m[prefix].push(p);
  }
  for (const k of Object.keys(m)) {
    m[k].sort((a, b) => a.localeCompare(b));
  }
  return m;
}

function orderedGroupKeys(grouped: Record<string, string[]>): string[] {
  const keys = Object.keys(grouped);
  const ordered = PERM_CATEGORY_ORDER.filter((k) => keys.includes(k));
  const rest = keys.filter((k) => !PERM_CATEGORY_ORDER.includes(k)).sort();
  return [...ordered, ...rest];
}

export default function TenantRolesPage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = params?.tenantId;
  const { lang } = useLanguage();
  const t = useDict("roles", lang);
  const ct = useDict("common", lang);

  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<RoleItem[]>([]);
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);
  
  // Pagination state
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const [permissionCatalog, setPermissionCatalog] = useState<string[]>([]);
  const [catalogLoaded, setCatalogLoaded] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<RoleItem | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ name: "" });
  const [selectedPerms, setSelectedPerms] = useState<string[]>([]);

  const title = useMemo(() => (editing ? t.editTitle : t.createTitle), [editing, t]);

  const catalogSet = useMemo(() => new Set(permissionCatalog), [permissionCatalog]);
  const groupedCatalog = useMemo(() => groupPermissions(permissionCatalog), [permissionCatalog]);
  const catalogPrefixOrder = useMemo(() => orderedGroupKeys(groupedCatalog), [groupedCatalog]);
  const extraPerms = useMemo(
    () => selectedPerms.filter((p) => !catalogSet.has(p)).sort((a, b) => a.localeCompare(b)),
    [selectedPerms, catalogSet],
  );

  const fetchPermissionCatalog = useCallback(async () => {
    if (!tenantId) return;
    setCatalogError(null);
    try {
      const data = await api.get<{ permissions?: string[] }>(`/tenants/${tenantId}/permission-catalog`);
      const list = Array.isArray(data?.permissions) ? data.permissions : [];
      setPermissionCatalog(list);
      setCatalogLoaded(true);
    } catch (e) {
      console.error("Failed to load permission catalog:", e);
      setCatalogError(t.catalogLoadFailed);
      setCatalogLoaded(true);
      setPermissionCatalog([]);
    }
  }, [tenantId, t.catalogLoadFailed]);

  const fetchRoles = async (q: string, page: number = 1, pageSize: number = 10) => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<{ items: RoleItem[]; total: number } | RoleItem[]>(
        `/tenants/${tenantId}/roles?q=${encodeURIComponent(q)}&page=${page}&page_size=${pageSize}`
      );
      
      if (Array.isArray(data)) {
        setItems(data);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.length
        }));
      } else if (data && 'items' in data) {
        setItems(data.items);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: data.total
        }));
      } else {
        setItems([]);
        setPagination(prev => ({
          ...prev,
          page,
          pageSize,
          total: 0
        }));
      }
    } catch (e) {
      console.error("Failed to fetch roles:", e);
      setError(ct.error);
      setItems([]);
      setPagination(prev => ({
        ...prev,
        total: 0
      }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const tmr = setTimeout(() => fetchRoles(query, pagination.page, pagination.pageSize), 250);
    return () => clearTimeout(tmr);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, query, pagination.page, pagination.pageSize]);

  useEffect(() => {
    if (modalOpen && tenantId) {
      fetchPermissionCatalog();
    }
  }, [modalOpen, tenantId, fetchPermissionCatalog]);

  const permDescription = (perm: string): string => {
    const key = permLabelKey(perm) as keyof typeof t;
    const label = (t as Record<string, string | undefined>)[key];
    return label && label !== key ? label : perm;
  };

  const togglePerm = (perm: string) => {
    setSelectedPerms((prev) =>
      prev.includes(perm) ? prev.filter((p) => p !== perm) : [...prev, perm].sort((a, b) => a.localeCompare(b)),
    );
  };

  const selectAllCatalog = () => {
    const s = new Set(selectedPerms);
    permissionCatalog.forEach((p) => s.add(p));
    const merged: string[] = [];
    s.forEach((p) => merged.push(p));
    merged.sort((a, b) => a.localeCompare(b));
    setSelectedPerms(merged);
  };

  const clearAllSelected = () => setSelectedPerms([]);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: "" });
    setSelectedPerms([]);
    setModalOpen(true);
  };

  const openEdit = (r: RoleItem) => {
    setEditing(r);
    setForm({ name: r.name });
    setSelectedPerms([...(r.permissions || [])].sort((a, b) => a.localeCompare(b)));
    setModalOpen(true);
  };

  const save = async () => {
    if (!tenantId) return;
    setSaving(true);
    setError(null);
    try {
      const permissions = [...selectedPerms];
      if (editing) {
        await api.put(`/tenants/${tenantId}/roles/${editing.id}`, { permissions });
      } else {
        await api.post(`/tenants/${tenantId}/roles`, { name: form.name, permissions });
      }
      setModalOpen(false);
      await fetchRoles(query, pagination.page, pagination.pageSize);
    } catch (e) {
      console.error("Failed to save role:", e);
      if (e instanceof APIError) {
        setError(e.message || t.saveFailed);
      } else {
        setError(t.saveFailed);
      }
    } finally {
      setSaving(false);
    }
  };

  const del = async (r: RoleItem) => {
    if (!tenantId) return;
    if (!window.confirm(t.confirmDelete)) return;
    setError(null);
    try {
      await api.delete(`/tenants/${tenantId}/roles/${r.id}`);
      await fetchRoles(query, pagination.page, pagination.pageSize);
    } catch (e) {
      console.error("Failed to delete role:", e);
      setError(t.deleteFailed);
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
          <p className="text-sm text-slate-500 mt-1">{tenantId}</p>
        </div>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700"
        >
          {t.addRole}
        </button>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm">
        <div className="p-4 border-b border-slate-100">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t.searchPlaceholder}
            className="w-full max-w-md border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>

        {error && <div className="p-4 text-sm text-red-700 bg-red-50 border-b border-red-100">{error}</div>}

        {loading ? (
          <div className="p-6 text-sm text-slate-500">{ct.loading}</div>
        ) : items.length === 0 ? (
          <div className="p-6 text-sm text-slate-500">{ct.noData}</div>
        ) : (
          <>
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.name}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.permissions}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.status}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">{ct.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-slate-100">
                {items.map((r) => (
                  <tr key={r.id} className="hover:bg-slate-50/50">
                    <td className="px-6 py-4 text-sm text-slate-700">{r.name}</td>
                    <td className="px-6 py-4 text-sm text-slate-600">
                      {(r.permissions || []).slice(0, 6).join(", ")}
                      {(r.permissions || []).length > 6 ? " ..." : ""}
                    </td>
                    <td className="px-6 py-4 text-sm text-slate-600">{r.status}</td>
                    <td className="px-6 py-4 text-right text-sm">
                      <button onClick={() => openEdit(r)} className="text-indigo-600 hover:text-indigo-800 mr-4">
                        {ct.edit}
                      </button>
                      <button onClick={() => del(r)} className="text-red-600 hover:text-red-800">
                        {ct.delete}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {items.length > 0 && (
            <div className="flex justify-between items-center px-6 py-4 border-t border-slate-100">
              <div className="flex items-center space-x-4">
                <div className="text-sm text-slate-500">
                  共 {pagination.total} 条记录
                </div>
                <div className="flex items-center space-x-2">
                  <span className="text-sm text-slate-500">每页显示：</span>
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
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
            <div className="p-6 border-b border-slate-100 shrink-0">
              <h2 className="text-xl font-semibold">{title}</h2>
            </div>
            <div className="p-6 overflow-y-auto flex-1 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.name}</label>
                <input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  disabled={!!editing}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500 disabled:bg-slate-50"
                />
              </div>

              <div>
                <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
                  <label className="block text-sm font-medium text-slate-700">{t.permissions}</label>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={selectAllCatalog}
                      className="text-xs text-indigo-600 hover:text-indigo-800 disabled:opacity-50"
                      disabled={!catalogLoaded || permissionCatalog.length === 0}
                    >
                      {t.selectAllPermissions}
                    </button>
                    <span className="text-slate-300">|</span>
                    <button type="button" onClick={clearAllSelected} className="text-xs text-slate-600 hover:text-slate-900">
                      {t.clearPermissions}
                    </button>
                  </div>
                </div>
                <p className="text-xs text-slate-500 mb-3">{t.permissionPickerHint}</p>

                {catalogError && <div className="mb-3 text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-md px-3 py-2">{catalogError}</div>}

                {!catalogLoaded ? (
                  <div className="text-sm text-slate-500 py-4">{ct.loading}</div>
                ) : permissionCatalog.length === 0 && !catalogError ? (
                  <div className="text-sm text-slate-500 py-4">{t.catalogLoadFailed}</div>
                ) : (
                  <div className="border border-slate-200 rounded-md divide-y divide-slate-100 max-h-64 overflow-y-auto">
                    {catalogPrefixOrder.map((prefix) => (
                        <div key={prefix} className="p-3">
                          <div className="text-[11px] font-semibold text-slate-500 uppercase tracking-wide mb-2">{prefix}</div>
                          <ul className="space-y-2">
                            {groupedCatalog[prefix].map((perm) => (
                              <li key={perm} className="flex items-start gap-2">
                                <input
                                  type="checkbox"
                                  id={`perm-${perm}`}
                                  checked={selectedPerms.includes(perm)}
                                  onChange={() => togglePerm(perm)}
                                  className="mt-1 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                                />
                                <label htmlFor={`perm-${perm}`} className="text-sm text-slate-800 cursor-pointer select-none">
                                  <span className="font-mono text-xs text-slate-600">{perm}</span>
                                  <span className="block text-slate-700">{permDescription(perm)}</span>
                                </label>
                              </li>
                            ))}
                          </ul>
                        </div>
                      ))}
                  </div>
                )}

                {extraPerms.length > 0 && (
                  <div className="mt-4">
                    <div className="text-xs font-semibold text-amber-800 mb-2">{t.extraPermissions}</div>
                    <ul className="space-y-2 border border-amber-200 rounded-md p-3 bg-amber-50/50">
                      {extraPerms.map((perm) => (
                        <li key={perm} className="flex items-start gap-2">
                          <input
                            type="checkbox"
                            id={`perm-extra-${perm}`}
                            checked={selectedPerms.includes(perm)}
                            onChange={() => togglePerm(perm)}
                            className="mt-1 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                          />
                          <label htmlFor={`perm-extra-${perm}`} className="text-sm cursor-pointer font-mono break-all">
                            {perm}
                          </label>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>

            <div className="flex justify-end gap-3 p-6 border-t border-slate-100 shrink-0">
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
                disabled={saving || (!editing && !form.name.trim())}
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
