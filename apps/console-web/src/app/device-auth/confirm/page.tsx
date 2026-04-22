"use client";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { api, APIError } from "@/lib/api/client";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";

function normalizeUserCode(raw: string | null): string {
  if (!raw) return "";
  let s = raw.toUpperCase().trim().replace(/\s+/g, "");
  s = s.replace(/-/g, "");
  if (s.length === 8) {
    return `${s.slice(0, 4)}-${s.slice(4)}`;
  }
  return raw.trim();
}

function ConfirmDeviceAuthInner() {
  const searchParams = useSearchParams();
  const { lang } = useLanguage();
  const t = useDict("deviceAuth", lang);
  const ct = useDict("common", lang);

  const codeParam = searchParams.get("code");
  const userCode = useMemo(() => normalizeUserCode(codeParam), [codeParam]);

  const [hasToken, setHasToken] = useState<boolean | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<"ok_approve" | "ok_deny" | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setHasToken(!!(typeof window !== "undefined" && localStorage.getItem("token")));
  }, []);

  const submit = useCallback(
    async (approve: boolean) => {
      if (!userCode) return;
      setError(null);
      setSubmitting(true);
      try {
        await api.post("/device-auth/confirm", { user_code: userCode, approve });
        setResult(approve ? "ok_approve" : "ok_deny");
      } catch (e) {
        if (e instanceof APIError) {
          setError(e.message);
        } else {
          setError(ct.error);
        }
      } finally {
        setSubmitting(false);
      }
    },
    [userCode, ct.error]
  );

  if (!codeParam || !userCode) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 px-4">
        <div className="max-w-md w-full rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
          <h1 className="text-lg font-semibold text-slate-900">{t.title}</h1>
          <p className="mt-2 text-sm text-slate-600">{t.missingCode}</p>
        </div>
      </div>
    );
  }

  if (hasToken === false) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 px-4">
        <div className="max-w-md w-full rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
          <h1 className="text-lg font-semibold text-slate-900">{t.title}</h1>
          <p className="mt-2 text-sm text-slate-600">{t.needLogin}</p>
          <p className="mt-4 rounded-lg bg-slate-50 px-3 py-2 font-mono text-sm text-slate-800">{userCode}</p>
          <p className="mt-4 text-sm text-slate-500">{t.keepCodeHint}</p>
          <Link
            href="/login"
            className="mt-6 inline-flex items-center justify-center rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t.signIn}
          </Link>
        </div>
      </div>
    );
  }

  if (hasToken === null) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50">
        <div className="text-sm text-slate-500">{ct.loading}</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 px-4">
      <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 p-4">
        <div
          className="relative z-50 w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-elevated"
          role="dialog"
          aria-modal="true"
          aria-labelledby="device-auth-title"
        >
          <h1 id="device-auth-title" className="text-lg font-semibold text-slate-900">
            {t.title}
          </h1>
          <p className="mt-2 text-sm text-slate-600">{t.description}</p>

          <div className="mt-6 rounded-xl border border-indigo-100 bg-indigo-50/60 px-4 py-3 text-center">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-indigo-600">{t.userCodeLabel}</div>
            <div className="mt-1 font-mono text-2xl font-semibold tracking-widest text-slate-900">{userCode}</div>
          </div>

          {error && (
            <div className="mt-4 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
          )}

          {result === "ok_approve" && (
            <p className="mt-4 text-sm font-medium text-emerald-700">{t.approvedSuccess}</p>
          )}
          {result === "ok_deny" && (
            <p className="mt-4 text-sm font-medium text-slate-700">{t.deniedSuccess}</p>
          )}

          {result === null && (
            <div className="mt-8 flex flex-col gap-2 sm:flex-row sm:justify-end">
              <button
                type="button"
                disabled={submitting}
                onClick={() => void submit(false)}
                className="order-2 rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 sm:order-1"
              >
                {t.deny}
              </button>
              <button
                type="button"
                disabled={submitting}
                onClick={() => void submit(true)}
                className="order-1 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 sm:order-2"
              >
                {submitting ? ct.loading : t.approve}
              </button>
            </div>
          )}

          {result !== null && (
            <div className="mt-6">
              <Link href="/overview" className="text-sm font-medium text-indigo-600 hover:text-indigo-800">
                {t.backToConsole}
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function DeviceAuthConfirmPage() {
  const { lang } = useLanguage();
  const ct = useDict("common", lang);
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center bg-slate-50">
          <p className="text-sm text-slate-500">{ct.loading}</p>
        </div>
      }
    >
      <ConfirmDeviceAuthInner />
    </Suspense>
  );
}
