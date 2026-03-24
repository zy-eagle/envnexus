"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface AuditEvent {
  ID: string;
  TenantID: string;
  DeviceID: string | null;
  SessionID: string | null;
  EventType: string;
  EventPayloadJSON: string;
  Archived: boolean;
  CreatedAt: string;
}

export default function AuditEventsPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('auditEvents', lang);
  const ct = useDict('common', lang);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({
    session_id: '',
    device_id: '',
    event_type: '',
    include_archived: false,
  });
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchEvents = async () => {
    setLoading(true);
    try {
      const queryParts: string[] = [];
      if (filters.session_id) queryParts.push(`session_id=${filters.session_id}`);
      if (filters.device_id) queryParts.push(`device_id=${filters.device_id}`);
      if (filters.event_type) queryParts.push(`event_type=${filters.event_type}`);
      if (filters.include_archived) queryParts.push(`include_archived=true`);
      const qs = queryParts.length > 0 ? `?${queryParts.join('&')}` : '';

      const data = await api.get<{ items: AuditEvent[] }>(`/tenants/${params.tenantId}/audit-events${qs}`);
      setEvents(data.items || []);
    } catch (error) {
      console.error('Failed to fetch audit events:', error);
      setEvents([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEvents();
  }, [params.tenantId]);

  const handleFilter = (e: React.FormEvent) => {
    e.preventDefault();
    fetchEvents();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
      </div>

      <form onSubmit={handleFilter} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <input
            type="text"
            placeholder={t.filterBySession}
            value={filters.session_id}
            onChange={e => setFilters({ ...filters, session_id: e.target.value })}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="text"
            placeholder={t.filterByDevice}
            value={filters.device_id}
            onChange={e => setFilters({ ...filters, device_id: e.target.value })}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <input
            type="text"
            placeholder={t.filterByType}
            value={filters.event_type}
            onChange={e => setFilters({ ...filters, event_type: e.target.value })}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
          />
          <div className="flex items-center space-x-3">
            <label className="flex items-center space-x-2 text-sm text-gray-600">
              <input
                type="checkbox"
                checked={filters.include_archived}
                onChange={e => setFilters({ ...filters, include_archived: e.target.checked })}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span>{t.includeArchived}</span>
            </label>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
            >
              Filter
            </button>
          </div>
        </div>
      </form>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : events.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noEvents}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.eventType}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.deviceId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.sessionId}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.createdAt}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {events.map((evt) => (
                <>
                  <tr key={evt.ID}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-blue-100 text-blue-800">
                        {evt.EventType}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono text-xs">
                      {evt.DeviceID || '-'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono text-xs">
                      {evt.SessionID || '-'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(evt.CreatedAt).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={() => setExpandedId(expandedId === evt.ID ? null : evt.ID)}
                        className="text-blue-600 hover:text-blue-900"
                      >
                        {expandedId === evt.ID ? 'Hide' : t.payload}
                      </button>
                    </td>
                  </tr>
                  {expandedId === evt.ID && evt.EventPayloadJSON && (
                    <tr key={`${evt.ID}-payload`}>
                      <td colSpan={5} className="px-6 py-4 bg-gray-50">
                        <pre className="text-xs text-gray-700 overflow-x-auto whitespace-pre-wrap">
                          {(() => {
                            try {
                              return JSON.stringify(JSON.parse(evt.EventPayloadJSON), null, 2);
                            } catch {
                              return evt.EventPayloadJSON;
                            }
                          })()}
                        </pre>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
