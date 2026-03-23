"use client";

import { useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { api, APIError } from '@/lib/api/client';
import { useRouter } from 'next/navigation';

const dict = {
  en: {
    title: "Sign in to EnvNexus",
    emailPlaceholder: "Email address",
    passwordPlaceholder: "Password",
    signInBtn: "Sign in",
    loginFailed: "Login failed",
  },
  zh: {
    title: "登录 EnvNexus",
    emailPlaceholder: "邮箱地址",
    passwordPlaceholder: "密码",
    signInBtn: "登录",
    loginFailed: "登录失败",
  }
};

export default function LoginPage() {
  const router = useRouter();
  const { lang } = useLanguage();
  const t = dict[lang];
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
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
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8 bg-white p-8 rounded-lg shadow">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            {t.title}
          </h2>
        </div>
        <form className="mt-8 space-y-6" onSubmit={handleLogin}>
          {error && (
            <div className="text-red-500 text-sm text-center bg-red-50 p-2 rounded">
              {error}
            </div>
          )}
          <div className="rounded-md shadow-sm -space-y-px">
            <div>
              <input
                id="email-address"
                name="email"
                type="email"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder={t.emailPlaceholder}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <input
                id="password"
                name="password"
                type="password"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder={t.passwordPlaceholder}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>

          <div>
            <button
              type="submit"
              disabled={submitting}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
            >
              {submitting ? '...' : t.signInBtn}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
