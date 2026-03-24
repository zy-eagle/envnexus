"use client";

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface SessionDetail {
  id: string;
  tenant_id: string;
  device_id: string;
  transport: string;
  status: string;
  initiator_type: string;
  started_at: string;
  ended_at: string | null;
}

interface AuditEvent {
  ID: string;
  EventType: string;
  DeviceID: string | null;
  EventPayloadJSON: string;
  CreatedAt: string;
}

export default function SessionDetailPage({ params }: { params: { tenantId: string; sessionId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('sessions', lang);
  const ct = useDict('common', lang);
  const [session, setSession] = useState<SessionDetail | null>(null);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const [sessionData, auditData] = await Promise.all([
          api.get<SessionDetail>(`/tenants/${params.tenantId}/sessions/${params.sessionId}`),
          api.get<{ items: AuditEvent[] }>(`/tenants/${params.tenantId}/audit-events?session_id=${params.sessionId}`),
        ]);
        setSession(sessionData);
        setEvents(auditData.items || []);
      } catch (error) {
        console.error('Failed to load session detail:', error);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [params.tenantId, params.sessionId]);

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

  const formatDuration = (start: string, end: string | null) => {
    const startMs = new Date(start).getTime();
    const endMs = end ? new Date(end).getTime() : Date.now();
    const diffSec = Math.floor((endMs - startMs) / 1000);
    if (diffSec < 60) return `${diffSec}s`;
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ${diffSec % 60}s`;
    return `${Math.floor(diffSec / 3600)}h ${Math.floor((diffSec % 3600) / 60)}m`;
  };

  if (loading) {
    return (
      <div className="p-8 text-center text-gray-500">{ct.loading}</div>
    );
  }

  if (!session) {
    return (
      <div className="p-8 text-center text-gray-500">{ct.error}</div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center space-x-4">
        <Link
          href={`/tenants/${params.tenantId}/sessions`}
          className="text-blue-600 hover:text-blue-800 text-sm"
        >
          &larr; {t.backToSessions}
        </Link>
      </div>

      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.detail}</h1>
        <span className={`px-3 py-1 inline-flex text-sm leading-5 font-semibold rounded-full ${statusColor(session.status)}`}>
          {session.status}
        </span>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <dl className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.sessionId}</dt>
            <dd className="mt-1 text-sm text-gray-900 font-mono">{session.id}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.deviceId}</dt>
            <dd className="mt-1 text-sm text-gray-900 font-mono">{session.device_id}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.transport}</dt>
            <dd className="mt-1 text-sm text-gray-900">{session.transport}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.initiator}</dt>
            <dd className="mt-1 text-sm text-gray-900">{session.initiator_type}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.startedAt}</dt>
            <dd className="mt-1 text-sm text-gray-900">{new Date(session.started_at).toLocaleString()}</dd>
          </div>
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase">{t.duration}</dt>
            <dd className="mt-1 text-sm text-gray-900">{formatDuration(session.started_at, session.ended_at)}</dd>
          </div>
        </dl>
      </div>

      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">{t.timeline}</h2>
        {events.length === 0 ? (
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
            {t.noTimeline}
          </div>
        ) : (
          <div className="space-y-3">
            {events.map((evt, idx) => (
              <div key={evt.ID} className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
                <div
                  className="flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-gray-50"
                  onClick={() => setExpandedId(expandedId === evt.ID ? null : evt.ID)}
                >
                  <div className="flex items-center space-x-4">
                    <div className="flex-shrink-0 w-8 h-8 bg-blue-100 text-blue-700 rounded-full flex items-center justify-center text-xs font-bold">
                      {idx + 1}
                    </div>
                    <div>
                      <span className="px-2 py-0.5 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">
                        {evt.EventType}
                      </span>
                    </div>
                  </div>
                  <span className="text-xs text-gray-500">
                    {new Date(evt.CreatedAt).toLocaleString()}
                  </span>
                </div>
                {expandedId === evt.ID && evt.EventPayloadJSON && (
                  <div className="px-4 py-3 bg-gray-50 border-t border-gray-200">
                    <pre className="text-xs text-gray-700 overflow-x-auto whitespace-pre-wrap">
                      {(() => {
                        try {
                          return JSON.stringify(JSON.parse(evt.EventPayloadJSON), null, 2);
                        } catch {
                          return evt.EventPayloadJSON;
                        }
                      })()}
                    </pre>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
