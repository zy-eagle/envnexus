"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api, APIError } from '@/lib/api/client';

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof APIError) return err.message || fallback;
  if (err instanceof Error) return err.message || fallback;
  return fallback;
}

interface CommandTask {
  id: string;
  tenant_id: string;
  created_by: string;
  approver_id: string | null;
  approved_by: string | null;
  title: string;
  command_type: string;
  command_payload: string;
  device_ids: string[];
  risk_level: string;
  effective_risk: string;
  bypass_approval: boolean;
  emergency: boolean;
  target_env: string;
  change_ticket: string;
  business_app: string;
  note: string;
  status: string;
  approval_note: string;
  expires_at: string;
  approved_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

interface PendingApprovalsResponse {
  tasks: CommandTask[];
  total: number;
}

interface FileApproval {
  id: string;
  tenant_id: string;
  device_id: string;
  requested_by: string;
  path: string;
  action: string;
  status: string;
  expires_at: string;
  created_at: string;
}

const POLL_INTERVAL_MS = 15_000;

function relativeTime(dateStr: string, lang: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffSec = Math.max(0, Math.floor((now - then) / 1000));

  if (lang === 'zh') {
    if (diffSec < 60) return `${diffSec} 秒前`;
    const m = Math.floor(diffSec / 60);
    if (m < 60) return `${m} 分钟前`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h} 小时前`;
    const d = Math.floor(h / 24);
    return `${d} 天前`;
  }
  if (diffSec < 60) return `${diffSec}s ago`;
  const m = Math.floor(diffSec / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

function riskBadgeClass(level: string): string {
  switch (level) {
    case 'L1': return 'bg-green-100 text-green-800';
    case 'L2': return 'bg-yellow-100 text-yellow-800';
    case 'L3': return 'bg-red-100 text-red-800';
    default:   return 'bg-gray-100 text-gray-800';
  }
}

function PendingApprovalsContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = useDict('pendingApprovals', lang);
  const ct = useDict('common', lang);

  const [activeTab, setActiveTab] = useState<'commands' | 'files'>('commands');
  const [tasks, setTasks] = useState<CommandTask[]>([]);
  const [fileApprovals, setFileApprovals] = useState<FileApproval[]>([]);
  const [loading, setLoading] = useState(true);

  const [approveModal, setApproveModal] = useState<CommandTask | null>(null);
  const [approveNote, setApproveNote] = useState('');
  const [approving, setApproving] = useState(false);

  const [denyModal, setDenyModal] = useState<CommandTask | null>(null);
  const [denyReason, setDenyReason] = useState('');
  const [denying, setDenying] = useState(false);
  const [approveError, setApproveError] = useState('');
  const [denyError, setDenyError] = useState('');

  const [detailModal, setDetailModal] = useState<CommandTask | null>(null);

  const fetchTasks = useCallback(async () => {
    try {
      const data = await api.get<PendingApprovalsResponse>(`/tenants/${tenantId}/pending-approvals`);
      if (data && Array.isArray(data.tasks)) {
        setTasks(data.tasks);
      } else if (Array.isArray(data)) {
        setTasks(data as unknown as CommandTask[]);
      } else {
        setTasks([]);
      }
    } catch (error) {
      console.error('Failed to fetch pending approvals:', error);
      setTasks([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  const fetchFileApprovals = useCallback(async () => {
    try {
      const data = await api.get<{ items: FileApproval[] }>(`/tenants/${tenantId}/pending-file-approvals`);
      setFileApprovals(Array.isArray(data) ? data : (data as any)?.items || []);
    } catch {
      setFileApprovals([]);
    }
  }, [tenantId]);

  useEffect(() => {
    fetchTasks();
    fetchFileApprovals();
    const interval = setInterval(() => { fetchTasks(); fetchFileApprovals(); }, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchTasks, fetchFileApprovals]);

  const handleFileApprove = async (id: string) => {
    try {
      await api.post(`/tenants/${tenantId}/file-access-requests/${id}/approve`);
      fetchFileApprovals();
    } catch (error) {
      console.error('Failed to approve file request:', error);
    }
  };

  const handleFileDeny = async (id: string) => {
    try {
      await api.post(`/tenants/${tenantId}/file-access-requests/${id}/deny`);
      fetchFileApprovals();
    } catch (error) {
      console.error('Failed to deny file request:', error);
    }
  };

  const handleApprove = async () => {
    if (!approveModal) return;
    setApproving(true);
    setApproveError('');
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${approveModal.id}/approve`, { note: approveNote });
      setApproveModal(null);
      setApproveNote('');
      fetchTasks();
    } catch (error) {
      console.error('Failed to approve task:', error);
      setApproveError(errorMessage(error, t.actionFailed));
    } finally {
      setApproving(false);
    }
  };

  const handleDeny = async () => {
    if (!denyModal) return;
    setDenying(true);
    setDenyError('');
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${denyModal.id}/deny`, { reason: denyReason });
      setDenyModal(null);
      setDenyReason('');
      fetchTasks();
    } catch (error) {
      console.error('Failed to deny task:', error);
      setDenyError(errorMessage(error, t.actionFailed));
    } finally {
      setDenying(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
          {!loading && (
            <span className="inline-flex items-center justify-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
              {tasks.length + fileApprovals.length}
            </span>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200">
        <button
          onClick={() => setActiveTab('commands')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'commands'
              ? 'border-indigo-600 text-indigo-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          {(t as any).tabCommands}
          {tasks.length > 0 && <span className="ml-1.5 px-1.5 py-0.5 rounded-full text-xs bg-blue-100 text-blue-700">{tasks.length}</span>}
        </button>
        <button
          onClick={() => setActiveTab('files')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'files'
              ? 'border-indigo-600 text-indigo-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          {(t as any).tabFileDownloads}
          {fileApprovals.length > 0 && <span className="ml-1.5 px-1.5 py-0.5 rounded-full text-xs bg-amber-100 text-amber-700">{fileApprovals.length}</span>}
        </button>
      </div>

      {/* File download approvals */}
      {activeTab === 'files' && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          {fileApprovals.length === 0 ? (
            <div className="p-12 text-center">
              <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <h3 className="mt-3 text-sm font-medium text-gray-900">{t.noApprovals}</h3>
            </div>
          ) : (
            <div className="divide-y divide-gray-200">
              {fileApprovals.map((fa) => (
                <div key={fa.id} className="p-5 hover:bg-gray-50 transition-colors">
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2 mb-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800">
                          {(t as any).fileDownload}
                        </span>
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">L1</span>
                      </div>
                      <code className="block text-sm bg-gray-900 text-green-400 rounded px-3 py-2 font-mono mb-3 overflow-x-auto max-w-2xl">
                        {fa.path}
                      </code>
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500">
                        <span>
                          <span className="font-medium text-gray-600">{t.requester}:</span>{' '}
                          {fa.requested_by}
                        </span>
                        <span>
                          <span className="font-medium text-gray-600">{(t as any).targetDevice}:</span>{' '}
                          {fa.device_id.slice(0, 12)}...
                        </span>
                        <span title={new Date(fa.created_at).toLocaleString()}>
                          {relativeTime(fa.created_at, lang)}
                        </span>
                      </div>
                    </div>
                    <div className="ml-4 flex-shrink-0 flex items-center space-x-2">
                      <button
                        onClick={() => handleFileApprove(fa.id)}
                        className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-green-600 hover:bg-green-700"
                      >
                        {t.approve}
                      </button>
                      <button
                        onClick={() => handleFileDeny(fa.id)}
                        className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700"
                      >
                        {t.deny}
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Command task list */}
      {activeTab === 'commands' && <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4"></div>
            <p>{ct.loading}</p>
          </div>
        ) : tasks.length === 0 ? (
          <div className="p-12 text-center">
            <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h3 className="mt-3 text-sm font-medium text-gray-900">{t.noApprovals}</h3>
            <p className="mt-1 text-sm text-gray-500">{t.noApprovalsDesc}</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {tasks.map((task) => (
              <div key={task.id} className="p-5 hover:bg-gray-50 transition-colors">
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    {/* Title row */}
                    <div className="flex items-center space-x-2 mb-2">
                      <h3 className="text-sm font-semibold text-gray-900 truncate">{task.title}</h3>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${riskBadgeClass(task.effective_risk)}`}>
                        {task.effective_risk}
                      </span>
                      {task.emergency && (
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-600 text-white">
                          {t.emergency}
                        </span>
                      )}
                    </div>

                    {/* Command payload */}
                    <div className="mb-3">
                      <code className="block text-xs bg-gray-900 text-green-400 rounded px-3 py-2 font-mono overflow-x-auto max-w-2xl whitespace-pre-wrap">
                        {task.command_payload}
                      </code>
                    </div>

                    {/* Meta row */}
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500">
                      <span>
                        <span className="font-medium text-gray-600">{t.requester}:</span>{' '}
                        {task.created_by}
                      </span>
                      <span>
                        <span className="font-medium text-gray-600">{t.devices}:</span>{' '}
                        {task.device_ids?.length ?? 0} {t.deviceUnit}
                      </span>
                      <span>
                        <span className="font-medium text-gray-600">{t.commandType}:</span>{' '}
                        {task.command_type === 'shell' ? 'Shell' : 'Tool'}
                      </span>
                      {task.target_env && (
                        <span>
                          <span className="font-medium text-gray-600">{t.targetEnv}:</span>{' '}
                          {task.target_env}
                        </span>
                      )}
                      <span title={new Date(task.created_at).toLocaleString()}>
                        {relativeTime(task.created_at, lang)}
                      </span>
                    </div>
                  </div>

                  {/* Action buttons */}
                  <div className="ml-4 flex-shrink-0 flex items-center space-x-2">
                    <button
                      onClick={() => { setApproveError(''); setDenyError(''); setApproveModal(task); }}
                      className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
                    >
                      {t.approve}
                    </button>
                    <button
                      onClick={() => { setApproveError(''); setDenyError(''); setDenyModal(task); }}
                      className="inline-flex items-center px-3 py-1.5 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                    >
                      {t.deny}
                    </button>
                    <button
                      onClick={() => setDetailModal(task)}
                      className="inline-flex items-center px-3 py-1.5 border border-gray-300 text-xs font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    >
                      {t.viewDetail}
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>}

      {/* Approve Modal */}
      {approveModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-1">{t.approveTitle}</h2>
            <p className="text-sm text-gray-500 mb-4">
              {t.approveDesc} <span className="font-medium text-gray-700">{approveModal.title}</span>
            </p>

            <div className="mb-2 p-2 rounded bg-gray-50 border border-gray-200">
              <code className="text-xs font-mono text-gray-800 whitespace-pre-wrap break-all">
                {approveModal.command_payload}
              </code>
            </div>

            <div className="flex items-center space-x-2 mb-4 text-xs text-gray-500">
              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${riskBadgeClass(approveModal.effective_risk)}`}>
                {approveModal.effective_risk}
              </span>
              <span>{approveModal.device_ids?.length ?? 0} {t.deviceUnit}</span>
            </div>

            <label className="block text-sm font-medium text-gray-700 mb-1">{t.noteOptional}</label>
            <textarea
              value={approveNote}
              onChange={(e) => setApproveNote(e.target.value)}
              rows={3}
              placeholder={t.notePlaceholder}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-green-500 focus:border-green-500"
            />

            {approveError && (
              <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
                {approveError}
              </div>
            )}

            <div className="flex justify-end space-x-3 mt-5">
              <button
                type="button"
                onClick={() => { setApproveModal(null); setApproveNote(''); setApproveError(''); }}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm hover:bg-gray-50"
              >
                {ct.cancel}
              </button>
              <button
                type="button"
                onClick={handleApprove}
                disabled={approving}
                className="px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700 disabled:opacity-50"
              >
                {approving ? t.processing : t.confirmApprove}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Deny Modal */}
      {denyModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-1">{t.denyTitle}</h2>
            <p className="text-sm text-gray-500 mb-4">
              {t.denyDesc} <span className="font-medium text-gray-700">{denyModal.title}</span>
            </p>

            <div className="mb-2 p-2 rounded bg-gray-50 border border-gray-200">
              <code className="text-xs font-mono text-gray-800 whitespace-pre-wrap break-all">
                {denyModal.command_payload}
              </code>
            </div>

            <div className="flex items-center space-x-2 mb-4 text-xs text-gray-500">
              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${riskBadgeClass(denyModal.effective_risk)}`}>
                {denyModal.effective_risk}
              </span>
              <span>{denyModal.device_ids?.length ?? 0} {t.deviceUnit}</span>
            </div>

            <label className="block text-sm font-medium text-gray-700 mb-1">{t.reasonOptional}</label>
            <textarea
              value={denyReason}
              onChange={(e) => setDenyReason(e.target.value)}
              rows={3}
              placeholder={t.reasonPlaceholder}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-red-500 focus:border-red-500"
            />

            {denyError && (
              <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
                {denyError}
              </div>
            )}

            <div className="flex justify-end space-x-3 mt-5">
              <button
                type="button"
                onClick={() => { setDenyModal(null); setDenyReason(''); setDenyError(''); }}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm hover:bg-gray-50"
              >
                {ct.cancel}
              </button>
              <button
                type="button"
                onClick={handleDeny}
                disabled={denying}
                className="px-4 py-2 bg-red-600 text-white rounded-md text-sm font-medium hover:bg-red-700 disabled:opacity-50"
              >
                {denying ? t.processing : t.confirmDeny}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Detail Modal */}
      {detailModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">{t.taskDetail}</h2>
              <button onClick={() => setDetailModal(null)} className="text-gray-400 hover:text-gray-600">
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <dl className="divide-y divide-gray-100">
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.taskTitle}</dt>
                <dd className="text-sm text-gray-900 col-span-2">{detailModal.title}</dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{ct.status}</dt>
                <dd className="text-sm text-gray-900 col-span-2">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
                    {detailModal.status}
                  </span>
                </dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.riskLevel}</dt>
                <dd className="text-sm col-span-2">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${riskBadgeClass(detailModal.effective_risk)}`}>
                    {detailModal.effective_risk}
                  </span>
                </dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.commandType}</dt>
                <dd className="text-sm text-gray-900 col-span-2">{detailModal.command_type === 'shell' ? 'Shell' : 'Tool'}</dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.commandContent}</dt>
                <dd className="col-span-2">
                  <code className="block text-xs bg-gray-900 text-green-400 rounded px-3 py-2 font-mono whitespace-pre-wrap break-all">
                    {detailModal.command_payload}
                  </code>
                </dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.requester}</dt>
                <dd className="text-sm text-gray-900 col-span-2">{detailModal.created_by}</dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.targetDevices}</dt>
                <dd className="text-sm text-gray-900 col-span-2">
                  {detailModal.device_ids?.map((id) => (
                    <span key={id} className="inline-block mr-1 mb-1 px-2 py-0.5 bg-gray-100 rounded text-xs font-mono">{id}</span>
                  ))}
                </dd>
              </div>
              {detailModal.target_env && (
                <div className="py-2 grid grid-cols-3 gap-4">
                  <dt className="text-sm font-medium text-gray-500">{t.targetEnv}</dt>
                  <dd className="text-sm text-gray-900 col-span-2">{detailModal.target_env}</dd>
                </div>
              )}
              {detailModal.change_ticket && (
                <div className="py-2 grid grid-cols-3 gap-4">
                  <dt className="text-sm font-medium text-gray-500">{t.changeTicket}</dt>
                  <dd className="text-sm text-gray-900 col-span-2">{detailModal.change_ticket}</dd>
                </div>
              )}
              {detailModal.business_app && (
                <div className="py-2 grid grid-cols-3 gap-4">
                  <dt className="text-sm font-medium text-gray-500">{t.businessApp}</dt>
                  <dd className="text-sm text-gray-900 col-span-2">{detailModal.business_app}</dd>
                </div>
              )}
              {detailModal.note && (
                <div className="py-2 grid grid-cols-3 gap-4">
                  <dt className="text-sm font-medium text-gray-500">{t.note}</dt>
                  <dd className="text-sm text-gray-900 col-span-2">{detailModal.note}</dd>
                </div>
              )}
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.submittedAt}</dt>
                <dd className="text-sm text-gray-900 col-span-2">{new Date(detailModal.created_at).toLocaleString()}</dd>
              </div>
              <div className="py-2 grid grid-cols-3 gap-4">
                <dt className="text-sm font-medium text-gray-500">{t.expiresAt}</dt>
                <dd className="text-sm text-gray-900 col-span-2">{new Date(detailModal.expires_at).toLocaleString()}</dd>
              </div>
            </dl>

            <div className="flex justify-end space-x-3 mt-5 pt-4 border-t border-gray-200">
              <button
                onClick={() => { setApproveError(''); setDenyError(''); setDetailModal(null); setApproveModal(detailModal); }}
                className="px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700"
              >
                {t.approve}
              </button>
              <button
                onClick={() => { setApproveError(''); setDenyError(''); setDetailModal(null); setDenyModal(detailModal); }}
                className="px-4 py-2 bg-red-600 text-white rounded-md text-sm font-medium hover:bg-red-700"
              >
                {t.deny}
              </button>
              <button
                onClick={() => setDetailModal(null)}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm hover:bg-gray-50"
              >
                {ct.close}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function PendingApprovalsPage({ params }: { params: { tenantId: string } }) {
  return <PendingApprovalsContent tenantId={params.tenantId} />;
}
