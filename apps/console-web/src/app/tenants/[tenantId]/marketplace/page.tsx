"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

interface MarketplaceItemRow {
  id: string;
  type: string;
  name: string;
  description?: string;
  version: string;
  author?: string;
  status: string;
}

interface SubscriptionRow {
  id: string;
  item_id: string;
  status: string;
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

async function downloadMarketplacePlugin(
  tenantId: string,
  itemId: string,
  fallbackFileName: string
): Promise<void> {
  const token = getToken();
  const url = `/api/v1/tenants/${tenantId}/marketplace/items/${itemId}/download`;
  const res = await fetch(url, {
    method: "GET",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  const ct = res.headers.get("content-type") || "";

  if (ct.includes("application/json")) {
    const json = (await res.json()) as {
      data?: { download_url?: string } | null;
      error?: { message?: string } | null;
    };
    if (!res.ok) {
      throw new Error(json.error?.message || "Download failed");
    }
    const downloadUrl =
      json.data && typeof json.data === "object" && "download_url" in json.data
        ? (json.data as { download_url: string }).download_url
        : undefined;
    if (downloadUrl) {
      window.open(downloadUrl, "_blank", "noopener,noreferrer");
      return;
    }
    throw new Error("Download failed");
  }

  if (!res.ok) {
    let msg = res.statusText;
    try {
      const j = (await res.json()) as { error?: { message?: string } | null };
      msg = j.error?.message || msg;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }

  const blob = await res.blob();
  const disp = res.headers.get("Content-Disposition");
  let fileName = fallbackFileName;
  if (disp) {
    const m = /filename\*?=(?:UTF-8'')?["']?([^"';]+)/i.exec(disp);
    if (m?.[1]) {
      fileName = decodeURIComponent(m[1].replace(/['"]/g, ""));
    }
  }
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = objectUrl;
  a.download = fileName;
  a.click();
  URL.revokeObjectURL(objectUrl);
}

export default function MarketplacePage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = params?.tenantId;
  const { lang } = useLanguage();
  const t = useDict("marketplace", lang);
  const ct = useDict("common", lang);

  const [items, setItems] = useState<MarketplaceItemRow[]>([]);
  const [subs, setSubs] = useState<SubscriptionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionId, setActionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);

  const subscribedActive = useMemo(() => {
    const set = new Set<string>();
    for (const s of subs) {
      if (s.status === "active") {
        set.add(s.item_id);
      }
    }
    return set;
  }, [subs]);

  const load = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    try {
      const [itemsRes, subsRes] = await Promise.all([
        api.get<{ items?: MarketplaceItemRow[] }>(
          `/tenants/${tenantId}/marketplace/items?status=published&page=1&page_size=100`
        ),
        api.get<{ items?: SubscriptionRow[] }>(
          `/tenants/${tenantId}/marketplace/subscriptions?page=1&page_size=100`
        ),
      ]);
      setItems(Array.isArray(itemsRes?.items) ? itemsRes.items! : []);
      setSubs(Array.isArray(subsRes?.items) ? subsRes.items! : []);
    } catch (e) {
      if (e instanceof APIError) {
        setError(e.message);
      } else {
        setError(ct.error);
      }
      setItems([]);
      setSubs([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId, ct.error]);

  useEffect(() => {
    void load();
  }, [load]);

  const toggleSubscribe = async (item: MarketplaceItemRow) => {
    if (!tenantId) return;
    setActionId(item.id);
    setError(null);
    const active = subscribedActive.has(item.id);
    try {
      if (active) {
        await api.delete(`/tenants/${tenantId}/marketplace/subscriptions/${item.id}`);
      } else {
        await api.post(`/tenants/${tenantId}/marketplace/subscriptions`, { item_id: item.id });
      }
      await load();
    } catch (e) {
      if (e instanceof APIError) {
        setError(e.message);
      } else {
        setError(ct.error);
      }
    } finally {
      setActionId(null);
    }
  };

  const onDownload = async (item: MarketplaceItemRow) => {
    if (!tenantId) return;
    setDownloadingId(item.id);
    setError(null);
    const safe = `${item.name.replace(/[^a-zA-Z0-9._-]+/g, "_")}.vsix`;
    try {
      await downloadMarketplacePlugin(tenantId, item.id, safe);
    } catch (e) {
      setError(e instanceof Error ? e.message : ct.error);
    } finally {
      setDownloadingId(null);
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
        ) : items.length === 0 ? (
          <div className="p-8 text-center text-sm text-slate-500">{t.noItems}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-100 bg-slate-50/80">
                  <th className="px-4 py-3 font-medium text-slate-600">{t.name}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.type}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.version}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.author}</th>
                  <th className="px-4 py-3 font-medium text-slate-600">{t.subscription}</th>
                  <th className="px-4 py-3 font-medium text-slate-600 w-48">{ct.actions}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => {
                  const isSub = subscribedActive.has(item.id);
                  const isPlugin = item.type === "plugin";
                  return (
                    <tr key={item.id} className="border-b border-slate-50 last:border-0">
                      <td className="px-4 py-3">
                        <div className="font-medium text-slate-900">{item.name}</div>
                        {item.description ? (
                          <div className="text-xs text-slate-500 line-clamp-2 mt-0.5">{item.description}</div>
                        ) : null}
                      </td>
                      <td className="px-4 py-3 text-slate-600">{item.type}</td>
                      <td className="px-4 py-3 text-slate-600">{item.version}</td>
                      <td className="px-4 py-3 text-slate-600">{item.author || "—"}</td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                            isSub ? "bg-emerald-50 text-emerald-800" : "bg-slate-100 text-slate-600"
                          }`}
                        >
                          {isSub ? t.subscribed : t.notSubscribed}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <button
                            type="button"
                            disabled={actionId === item.id}
                            onClick={() => void toggleSubscribe(item)}
                            className={`rounded-lg px-3 py-1.5 text-xs font-medium ${
                              isSub
                                ? "border border-slate-200 text-slate-700 hover:bg-slate-50"
                                : "bg-indigo-600 text-white hover:bg-indigo-700"
                            }`}
                          >
                            {actionId === item.id ? ct.loading : isSub ? t.unsubscribe : t.subscribe}
                          </button>
                          {isPlugin && (
                            <button
                              type="button"
                              disabled={!isSub || downloadingId === item.id}
                              onClick={() => void onDownload(item)}
                              className="rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-1.5 text-xs font-medium text-indigo-800 hover:bg-indigo-100 disabled:opacity-50"
                              title={!isSub ? t.downloadRequiresSubscription : undefined}
                            >
                              {downloadingId === item.id ? ct.loading : t.download}
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
