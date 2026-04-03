"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

interface UserItem {
  id: string;
  tenant_id: string;
  email: string;
  display_name: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export default function TenantUsersPage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = params?.tenantId;
  const { lang } = useLanguage();
  const t = useDict("users", lang);
  const ct = useDict("common", lang);

  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<UserItem[]>([]);
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<UserItem | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ email: "", display_name: "", password: "", status: "active" });

  const title = useMemo(() => (editing ? t.editTitle : t.createTitle), [editing, t]);

  const fetchUsers = async (q: string) => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<{ items: UserItem[] }>(
        `/tenants/${tenantId}/users?q=${encodeURIComponent(q)}&limit=50`,
      );
      setItems(Array.isArray(data?.items) ? data.items : []);
    } catch (e) {
      console.error("Failed to fetch users:", e);
      setError(ct.error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const tmr = setTimeout(() => fetchUsers(query), 250);
    return () => clearTimeout(tmr);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, query]);

  const openCreate = () => {
    setEditing(null);
    setForm({ email: "", display_name: "", password: "", status: "active" });
    setModalOpen(true);
  };

  const openEdit = (u: UserItem) => {
    setEditing(u);
    setForm({ email: u.email, display_name: u.display_name, password: "", status: u.status || "active" });
    setModalOpen(true);
  };

  const save = async () => {
    if (!tenantId) return;
    setSaving(true);
    setError(null);
    try {
      if (editing) {
        await api.put(`/tenants/${tenantId}/users/${editing.id}`, {
          display_name: form.display_name,
          password: form.password || undefined,
          status: form.status,
        });
      } else {
        await api.post(`/tenants/${tenantId}/users`, {
          email: form.email,
          display_name: form.display_name,
          password: form.password,
          status: form.status,
        });
      }
      setModalOpen(false);
      await fetchUsers(query);
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

  const del = async (u: UserItem) => {
    if (!tenantId) return;
    if (!window.confirm(t.confirmDelete)) return;
    setError(null);
    try {
      await api.delete(`/tenants/${tenantId}/users/${u.id}`);
      await fetchUsers(query);
    } catch (e) {
      console.error("Failed to delete user:", e);
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
          {t.addUser}
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
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.email}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.displayName}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">{t.status}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">{ct.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-slate-100">
                {items.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-50/50">
                    <td className="px-6 py-4 text-sm text-slate-700">{u.email}</td>
                    <td className="px-6 py-4 text-sm text-slate-700">{u.display_name}</td>
                    <td className="px-6 py-4 text-sm text-slate-600">{u.status}</td>
                    <td className="px-6 py-4 text-right text-sm">
                      <button onClick={() => openEdit(u)} className="text-indigo-600 hover:text-indigo-800 mr-4">
                        {ct.edit}
                      </button>
                      <button onClick={() => del(u)} className="text-red-600 hover:text-red-800">
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
              {!editing && (
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">{t.email}</label>
                  <input
                    value={form.email}
                    onChange={(e) => setForm({ ...form, email: e.target.value })}
                    className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                  />
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t.displayName}</label>
                <input
                  value={form.display_name}
                  onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                  className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <div className="flex items-baseline justify-between">
                  <label className="block text-sm font-medium text-slate-700 mb-1">{t.password}</label>
                  {editing && <span className="text-xs text-slate-400">{t.passwordHint}</span>}
                </div>
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

