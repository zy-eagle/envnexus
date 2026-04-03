"use client";

import { useEffect, useMemo, useState } from "react";
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

function parsePermissions(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
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

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<RoleItem | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ name: "", permissionsText: "" });

  const title = useMemo(() => (editing ? t.editTitle : t.createTitle), [editing, t]);

  const fetchRoles = async (q: string) => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<{ items: RoleItem[] }>(`/tenants/${tenantId}/roles?q=${encodeURIComponent(q)}&limit=100`);
      setItems(Array.isArray(data?.items) ? data.items : []);
    } catch (e) {
      console.error("Failed to fetch roles:", e);
      setError(ct.error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const tmr = setTimeout(() => fetchRoles(query), 250);
    return () => clearTimeout(tmr);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, query]);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: "", permissionsText: "" });
    setModalOpen(true);
  };

  const openEdit = (r: RoleItem) => {
    setEditing(r);
    setForm({ name: r.name, permissionsText: (r.permissions || []).join("\n") });
    setModalOpen(true);
  };

  const save = async () => {
    if (!tenantId) return;
    setSaving(true);
    setError(null);
    try {
      const permissions = parsePermissions(form.permissionsText);
      if (editing) {
        await api.put(`/tenants/${tenantId}/roles/${editing.id}`, { permissions });
      } else {
        await api.post(`/tenants/${tenantId}/roles`, { name: form.name, permissions });
      }
      setModalOpen(false);
      await fetchRoles(query);
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
      await fetchRoles(query);
    } catch (e) {
      console.error("Failed to delete role:", e);
      setError(t.deleteFailed);
    }
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
        )}
      </div>

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
            <h2 className="text-xl font-semibold mb-5">{title}</h2>
            <div className="space-y-4">
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
                <div className="flex items-baseline justify-between">
                  <label className="block text-sm font-medium text-slate-700 mb-1">{t.permissions}</label>
                  <span className="text-xs text-slate-400">{t.permissionsHint}</span>
                </div>
                <textarea
                  rows={8}
                  value={form.permissionsText}
                  onChange={(e) => setForm({ ...form, permissionsText: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500 font-mono"
                />
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

