"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface GovernanceRule {
  id: string;
  name: string;
  rule_type: string;
  severity: string;
  enabled: boolean;
  created_at: string;
}

export default function GovernanceRulesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('governanceRules', lang);
  const ct = useDict('common', lang);
  const [rules, setRules] = useState<GovernanceRule[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchRules = async () => {
    setLoading(true);
    try {
      const data = await api.get<{ items: GovernanceRule[] }>(`/tenants/${params.tenantId}/governance-rules`);
      setRules(Array.isArray(data) ? data : (data as any)?.items || []);
    } catch {
      setRules([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchRules(); }, [params.tenantId]);

  const handleDelete = async (id: string) => {
    if (!confirm(ct.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/governance-rules/${id}`);
      fetchRules();
    } catch (error) {
      console.error('Failed to delete rule:', error);
    }
  };

  const severityColor = (s: string) => {
    switch (s) {
      case 'critical': return 'bg-red-100 text-red-800';
      case 'warning': return 'bg-yellow-100 text-yellow-800';
      case 'info': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : rules.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noRules}</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{ct.name}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.ruleType}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.severity}</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.enabled}</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {rules.map(rule => (
                <tr key={rule.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{rule.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{rule.rule_type}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${severityColor(rule.severity)}`}>{rule.severity}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{rule.enabled ? 'Yes' : 'No'}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                    <button onClick={() => handleDelete(rule.id)} className="text-red-600 hover:text-red-900 text-xs font-medium">{ct.delete}</button>
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
