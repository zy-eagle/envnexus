"use client";

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';


function SessionsContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = useDict('sessions', lang);
  const ct = useDict('common', lang);
  const [sessions, setSessions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchSessions = async () => {
    try {
      const data = await api.get<{ items: any[] }>(`/tenants/${tenantId}/sessions`);
      console.log('Fetched sessions data:', data);
      if (Array.isArray(data.items)) {
        data.items.forEach((session, index) => {
          console.log(`Session ${index} started_at:`, session.started_at, typeof session.started_at);
        });
      }
      setSessions(Array.isArray(data.items) ? data.items : []);
    } catch (error) {
      console.error('Failed to fetch sessions:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAbort = async (sessionId: string) => {
    if (!confirm(t.confirmAbort)) return;
    try {
      await api.post(`/sessions/${sessionId}/abort`, { reason: "User requested" });
      fetchSessions();
    } catch (error) {
      console.error('Failed to abort session:', error);
    }
  };

  useEffect(() => { fetchSessions(); }, [tenantId]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'created': case 'attached': return 'bg-blue-100 text-blue-800';
      case 'diagnosing': case 'executing': return 'bg-yellow-100 text-yellow-800';
      case 'awaiting_approval': return 'bg-orange-100 text-orange-800';
      case 'completed': return 'bg-green-100 text-green-800';
      case 'aborted': case 'expired': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const isActive = (status: string) => !['completed', 'aborted', 'expired'].includes(status);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : sessions.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noSessions}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.transport}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.status}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.initiator}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.startedAt}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {sessions.map((session: any) => (
                <tr key={session.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 font-mono text-xs">{session.device_id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{session.transport}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColor(session.status)}`}>
                      {session.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{session.initiator_type}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {session.started_at ? (() => {
                      const date = new Date(session.started_at);
                      return isNaN(date.getTime()) ? 'Invalid Date' : date.toLocaleString();
                    })() : 'No Date'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <Link
                      href={`/tenants/${tenantId}/sessions/${session.id}`}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      {t.view}
                    </Link>
                    {isActive(session.status) && (
                      <button onClick={() => handleAbort(session.id)} className="text-red-600 hover:text-red-900">{t.abort}</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

export default function SessionsPage({ params }: { params: { tenantId: string } }) {
  return <SessionsContent tenantId={params.tenantId} />;
}
