"use client";

import { useState, useEffect } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';

interface ApprovalPolicy {
  id: string;
  tenant_id: string;
  name: string;
  risk_level: string;
  approver_user_id: string | null;
  approver_role_id: string | null;
  auto_approve: boolean;
  approval_rule: string;
  separation_of_duty: boolean;
  expires_minutes: number;
  status: string;
  priority: number;
  version: number;
  created_at: string;
  updated_at: string;
}

type ApprovalMethod = 'user' | 'role';

interface FormState {
  name: string;
  risk_level: string;
  approval_method: ApprovalMethod;
  approver_user_id: string;
  approver_role_id: string;
  auto_approve: boolean;
  approval_rule: string;
  separation_of_duty: boolean;
  expires_minutes: number;
  priority: number;
}

const INITIAL_FORM: FormState = {
  name: '',
  risk_level: 'L1',
  approval_method: 'user',
  approver_user_id: '',
  approver_role_id: '',
  auto_approve: false,
  approval_rule: 'single',
  separation_of_duty: false,
  expires_minutes: 30,
  priority: 0,
};

const RISK_LEVELS = ['L1', 'L2', 'L3', '*'] as const;
const APPROVAL_RULES = ['single', 'dual', 'sequential'] as const;

export default function ApprovalPoliciesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('approvalPolicies', lang);
  const ct = useDict('common', lang);

  const [policies, setPolicies] = useState<ApprovalPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(INITIAL_FORM);
  const [deleteTarget, setDeleteTarget] = useState<ApprovalPolicy | null>(null);

  const fetchPolicies = async () => {
    try {
      const data = await api.get<ApprovalPolicy[]>(`/tenants/${params.tenantId}/approval-policies`);
      setPolicies(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch approval policies:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPolicies();
  }, [params.tenantId]);

  const openCreateModal = () => {
    setEditingId(null);
    setFormError(null);
    setForm(INITIAL_FORM);
    setIsModalOpen(true);
  };

  const openEditModal = (policy: ApprovalPolicy) => {
    setEditingId(policy.id);
    setFormError(null);
    const method: ApprovalMethod = policy.approver_role_id ? 'role' : 'user';
    setForm({
      name: policy.name,
      risk_level: policy.risk_level,
      approval_method: method,
      approver_user_id: policy.approver_user_id || '',
      approver_role_id: policy.approver_role_id || '',
      auto_approve: policy.auto_approve,
      approval_rule: policy.approval_rule,
      separation_of_duty: policy.separation_of_duty,
      expires_minutes: policy.expires_minutes,
      priority: policy.priority,
    });
    setIsModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const body: Record<string, unknown> = {
      name: form.name,
      risk_level: form.risk_level,
      auto_approve: form.auto_approve,
      approval_rule: form.approval_rule,
      separation_of_duty: form.separation_of_duty,
      expires_minutes: form.expires_minutes,
      priority: form.priority,
    };

    if (!form.auto_approve) {
      if (form.approval_method === 'user') {
        body.approver_user_id = form.approver_user_id;
        body.approver_role_id = '';
      } else {
        body.approver_role_id = form.approver_role_id;
        body.approver_user_id = '';
      }
    } else {
      body.approver_user_id = '';
      body.approver_role_id = '';
    }

    try {
      if (editingId) {
        await api.put(`/tenants/${params.tenantId}/approval-policies/${editingId}`, body);
      } else {
        await api.post(`/tenants/${params.tenantId}/approval-policies`, body);
      }
      setIsModalOpen(false);
      fetchPolicies();
    } catch (error) {
      if (error instanceof APIError && error.status === 409) {
        setFormError(ct.duplicateName);
      } else {
        setFormError(t.saveFailed);
      }
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/approval-policies/${deleteTarget.id}`);
      setDeleteTarget(null);
      fetchPolicies();
    } catch (error) {
      console.error('Failed to delete approval policy:', error);
    }
  };

  const riskLevelBadge = (level: string) => {
    const colors: Record<string, string> = {
      L1: 'bg-green-100 text-green-800',
      L2: 'bg-yellow-100 text-yellow-800',
      L3: 'bg-red-100 text-red-800',
      '*': 'bg-gray-100 text-gray-800',
    };
    return colors[level] || 'bg-gray-100 text-gray-800';
  };

  const ruleLabel = (rule: string) => {
    const map: Record<string, string> = {
      single: t.ruleSingle,
      dual: t.ruleDual,
      sequential: t.ruleSequential,
    };
    return map[rule] || rule;
  };

  const approverDisplay = (policy: ApprovalPolicy) => {
    if (policy.auto_approve) return '-';
    if (policy.approver_user_id) return policy.approver_user_id;
    if (policy.approver_role_id) return policy.approver_role_id;
    return '-';
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button
          onClick={openCreateModal}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.addPolicy}
        </button>
      </div>

      {/* Data Table */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">{ct.loading}</div>
        ) : policies.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noPolicies}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.name}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.riskLevel}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.approver}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.autoApprove}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.approvalRule}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.expiresMinutes}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t.priority}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">{ct.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {policies.map((policy) => (
                  <tr key={policy.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{policy.name}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${riskLevelBadge(policy.risk_level)}`}>
                        {policy.risk_level}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono max-w-[160px] truncate">
                      {approverDisplay(policy)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        policy.auto_approve ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {policy.auto_approve ? t.autoApproveLabel : t.manualApproveLabel}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ruleLabel(policy.approval_rule)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{policy.expires_minutes}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{policy.priority}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={() => openEditModal(policy)}
                        className="text-blue-600 hover:text-blue-900 mr-4"
                      >
                        {ct.edit}
                      </button>
                      <button
                        onClick={() => setDeleteTarget(policy)}
                        className="text-red-600 hover:text-red-900"
                      >
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

      {/* Create / Edit Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
            <h2 className="text-xl font-semibold mb-5">{editingId ? t.editTitle : t.createTitle}</h2>
            <form onSubmit={handleSave} className="space-y-5">
              {formError && (
                <div className="flex items-center gap-2 rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
                  <svg className="h-4 w-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
                  </svg>
                  {formError}
                </div>
              )}

              {/* Policy Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.name}</label>
                <input
                  type="text"
                  required
                  value={form.name}
                  onChange={e => setForm({ ...form, name: e.target.value })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              {/* Risk Level */}
              {!editingId && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">{t.riskLevel}</label>
                  <select
                    value={form.risk_level}
                    onChange={e => setForm({ ...form, risk_level: e.target.value })}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                  >
                    {RISK_LEVELS.map(level => (
                      <option key={level} value={level}>
                        {level === '*' ? t.riskLevelAll : level}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Auto Approve Toggle */}
              <div>
                <div className="flex items-center justify-between">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">{t.autoApprove}</label>
                    <p className="text-xs text-gray-500 mt-0.5">{t.autoApproveDesc}</p>
                  </div>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={form.auto_approve}
                    onClick={() => setForm({ ...form, auto_approve: !form.auto_approve })}
                    className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                      form.auto_approve ? 'bg-blue-600' : 'bg-gray-200'
                    }`}
                  >
                    <span
                      className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                        form.auto_approve ? 'translate-x-5' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>
              </div>

              {/* Approver fields — hidden when auto_approve is on */}
              {!form.auto_approve && (
                <>
                  {/* Approval Method */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">{t.approvalMethod}</label>
                    <div className="flex gap-4">
                      <label
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg border-2 cursor-pointer transition-colors ${
                          form.approval_method === 'user'
                            ? 'border-blue-500 bg-blue-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <input
                          type="radio"
                          name="approval_method"
                          value="user"
                          checked={form.approval_method === 'user'}
                          onChange={() => setForm({ ...form, approval_method: 'user' })}
                          className="text-blue-600 focus:ring-blue-500"
                        />
                        <span className="text-sm font-medium text-gray-900">{t.methodUser}</span>
                      </label>
                      <label
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg border-2 cursor-pointer transition-colors ${
                          form.approval_method === 'role'
                            ? 'border-blue-500 bg-blue-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <input
                          type="radio"
                          name="approval_method"
                          value="role"
                          checked={form.approval_method === 'role'}
                          onChange={() => setForm({ ...form, approval_method: 'role' })}
                          className="text-blue-600 focus:ring-blue-500"
                        />
                        <span className="text-sm font-medium text-gray-900">{t.methodRole}</span>
                      </label>
                    </div>
                  </div>

                  {/* Approver User */}
                  {form.approval_method === 'user' && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">{t.approverUser}</label>
                      <input
                        type="text"
                        value={form.approver_user_id}
                        onChange={e => setForm({ ...form, approver_user_id: e.target.value })}
                        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                      />
                    </div>
                  )}

                  {/* Approver Role */}
                  {form.approval_method === 'role' && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">{t.approverRole}</label>
                      <input
                        type="text"
                        value={form.approver_role_id}
                        onChange={e => setForm({ ...form, approver_role_id: e.target.value })}
                        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                      />
                    </div>
                  )}
                </>
              )}

              {/* Approval Rule */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.approvalRule}</label>
                <select
                  value={form.approval_rule}
                  onChange={e => setForm({ ...form, approval_rule: e.target.value })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                >
                  {APPROVAL_RULES.map(rule => (
                    <option key={rule} value={rule}>{ruleLabel(rule)}</option>
                  ))}
                </select>
              </div>

              {/* Separation of Duty */}
              <div>
                <div className="flex items-center justify-between">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">{t.separationOfDuty}</label>
                    <p className="text-xs text-gray-500 mt-0.5">{t.separationOfDutyDesc}</p>
                  </div>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={form.separation_of_duty}
                    onClick={() => setForm({ ...form, separation_of_duty: !form.separation_of_duty })}
                    className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                      form.separation_of_duty ? 'bg-blue-600' : 'bg-gray-200'
                    }`}
                  >
                    <span
                      className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                        form.separation_of_duty ? 'translate-x-5' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>
              </div>

              {/* Expires Minutes */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.expiresMinutes}</label>
                <input
                  type="number"
                  min={1}
                  value={form.expires_minutes}
                  onChange={e => setForm({ ...form, expires_minutes: parseInt(e.target.value) || 30 })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              {/* Priority */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.priority}</label>
                <input
                  type="number"
                  min={0}
                  value={form.priority}
                  onChange={e => setForm({ ...form, priority: parseInt(e.target.value) || 0 })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              {/* Actions */}
              <div className="flex justify-end space-x-3 pt-2">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
                >
                  {ct.cancel}
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                >
                  {editingId ? ct.save : ct.create}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-sm p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">{ct.confirm}</h3>
            <p className="text-sm text-gray-600 mb-1">{t.confirmDelete}</p>
            <p className="text-sm font-medium text-gray-800 mb-5">
              {deleteTarget.name}
            </p>
            <div className="flex justify-end space-x-3">
              <button
                type="button"
                onClick={() => setDeleteTarget(null)}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
              >
                {ct.cancel}
              </button>
              <button
                type="button"
                onClick={handleDeleteConfirm}
                className="px-4 py-2 bg-red-600 text-white rounded-md text-sm font-medium hover:bg-red-700"
              >
                {ct.delete}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
