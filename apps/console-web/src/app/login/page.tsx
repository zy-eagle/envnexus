"use client";

import { useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';
import { useRouter } from 'next/navigation';

export default function LoginPage() {
  const router = useRouter();
  const { lang } = useLanguage();
  const t = useDict('login', lang);
  const [email, setEmail] = useState('admin@gmail.com');
  const [password, setPassword] = useState('admin123');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      const resp = await api.post<{
        access_token: string;
        expires_in: number;
        user: { id: string; tenant_id: string; email: string; display_name: string };
      }>('/auth/login', { email, password });

      localStorage.setItem('token', resp.access_token);
      localStorage.setItem('user', JSON.stringify(resp.user));
      router.push('/overview');
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError(t.loginFailed);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-slate-50">
      {/* Left: Branding Panel */}
      <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-indigo-600 via-indigo-700 to-indigo-900 relative overflow-hidden">
        <div className="absolute inset-0 opacity-10">
          <svg className="w-full h-full" viewBox="0 0 800 800" fill="none">
            <circle cx="400" cy="400" r="300" stroke="white" strokeWidth="0.5" />
            <circle cx="400" cy="400" r="200" stroke="white" strokeWidth="0.5" />
            <circle cx="400" cy="400" r="100" stroke="white" strokeWidth="0.5" />
            <line x1="100" y1="400" x2="700" y2="400" stroke="white" strokeWidth="0.5" />
            <line x1="400" y1="100" x2="400" y2="700" stroke="white" strokeWidth="0.5" />
          </svg>
        </div>
        <div className="relative z-10 flex flex-col justify-center px-16">
          <div className="flex items-center gap-3 mb-8">
            <div className="w-12 h-12 rounded-xl bg-white/20 backdrop-blur flex items-center justify-center">
              <span className="text-white text-xl font-bold">E</span>
            </div>
            <span className="text-2xl font-bold text-white tracking-tight">EnvNexus</span>
          </div>
          <h2 className="text-4xl font-bold text-white leading-tight mb-4">
            AI-Native Environment<br />Governance Platform
          </h2>
          <p className="text-indigo-200 text-lg max-w-md">
            Secure, manage, and govern your AI development environments with enterprise-grade controls.
          </p>
        </div>
      </div>

      {/* Right: Login Form */}
      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-sm">
          <div className="lg:hidden flex items-center gap-2.5 mb-10">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-indigo-600 flex items-center justify-center shadow-lg">
              <span className="text-white text-lg font-bold">E</span>
            </div>
            <span className="text-xl font-bold text-slate-900 tracking-tight">EnvNexus</span>
          </div>

          <h1 className="text-2xl font-semibold text-slate-900 tracking-tight">{t.title}</h1>
          <p className="mt-2 text-sm text-slate-500 mb-8">Enter your credentials to access the console</p>

          <form onSubmit={handleLogin} className="space-y-4">
            {error && (
              <div className="flex items-center gap-2 p-3 rounded-lg bg-red-50 border border-red-100">
                <svg className="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" /></svg>
                <span className="text-sm text-red-700">{error}</span>
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">{t.emailPlaceholder}</label>
              <input
                type="email"
                required
                className="input-field"
                placeholder="admin@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1.5">{t.passwordPlaceholder}</label>
              <input
                type="password"
                required
                className="input-field"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>

            <button
              type="submit"
              disabled={submitting}
              className="btn-primary w-full py-2.5 mt-2"
            >
              {submitting ? (
                <svg className="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" /><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" /></svg>
              ) : t.signInBtn}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
