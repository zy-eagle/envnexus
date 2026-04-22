"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

interface IdeTokenRow {
  id: string;
  name?: string;
  user_id?: string;
  access_expires_at?: string;
  refresh_expires_at?: string;
  last_used_at?: string | null;
  created_at?: string;
}

export default function DeveloperSettingsPage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = params?.tenantId;
  const { lang } = useLanguage();
  const t = useDict("developerSettings", lang);
  const ct = useDict("common", lang);

  const [rows, setRows] = useState<IdeTokenRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [revokeId, setRevokeId] = useState<string | null>(null);
  const [revoking, setRevoking] = useState(false);

  const load = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<{ items?: IdeTokenRow[] } | IdeTokenRow[]>(
        `/tenants/${tenantId}/ide-tokens?page=1&page_size=100`
      );
      if (Array.isArray(data)) {
        setRows(data);
      } else {
        setRows(Array.isArray(data?.items) ? data.items! : []);
      }
    } catch (e) {
      if (e instanceof APIError) {
        setError(e.message);
      } else {
        setError(ct.error);
      }
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId, ct.error]);

  useEffect(() => {
    void load();
  }, [load]);

  const confirmRevoke = async () => {
    if (!tenantId || !revokeId) return;
    setRevoking(true);
    setError(null);
    try {
      await api.delete(`/tenants/${tenantId}/ide-tokens/${revokeId}`);
      setRevokeId(null);
      await load();
    } catch (e) {
      if (e instanceof APIError) {
        setError(e.message);
      } else {
        setError(ct.error);
      }
    } finally {
      setRevoking(false);
    }
  };

  const fmt = (iso?: string | null) => {
    if (!iso) return "—";
    try {
      return new Date(iso).toLocaleString(lang === "zh" ? "zh-CN" : "en-US");
    } catch {
      return iso;
    }
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-slate-900 tracking-tight">{t.title}</h1>
        <p className="mt-1 text-sm text-slate-500">{t.subtitle}</p>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
      )}

      <div className="rounded-xl border border-slate-200/80 bg-white overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-sm text-slate-500">{ct.loading}</div>
        ) : rows.length === 0 ? (
          <div className="p-8 text-center text-sm text-slate-500">{t.noTokens}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-100 bg-slate-50/80">
                  <th className="px-4 py-3 font-medium text-slate-600">{t.name}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.userId}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.lastUsed}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.accessExpires}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.createdAt}</th>
                  <th className="px-4 py-3 font-medium text-slate-600 w-28">{ct.actions}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id} className="border-b border-slate-50 last:border-0">
                    <td className="px-4 py-3 text-slate-800">{row.name || "—"}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-600">{row.user_id || "—"}</td>
                    <td className="px-4 py-3 text-slate-600">{fmt(row.last_used_at)}</td>
                    <td className="px-4 py-3 text-slate-600">{fmt(row.access_expires_at)}</td>
                    <td className="px-4 py-3 text-slate-600">{fmt(row.created_at)}</td>
                    <td className="px-4 py-3">
                      <button
                        type="button"
                        onClick={() => setRevokeId(row.id)}
                        className="text-sm font-medium text-red-600 hover:text-red-800"
                      >
                        {t.revoke}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {revokeId != null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
          <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-elevated">
            <h2 className="text-base font-semibold text-slate-900">{t.revokeTitle}</h2>
            <p className="mt-2 text-sm text-slate-600">{t.revokeBody}</p>
            <div className="mt-6 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setRevokeId(null)}
                className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
              >
                {ct.cancel}
              </button>
              <button
                type="button"
                disabled={revoking}
                onClick={() => void confirmRevoke()}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                {revoking ? ct.loading : ct.confirm}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
