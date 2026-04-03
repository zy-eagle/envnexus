"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";
import { api } from "@/lib/api/client";
import { useAuth } from "@/lib/auth/AuthContext";

interface CommandTask {
  id: string;
  tenant_id: string;
  created_by: string;
  created_by_name?: string;
  approver_id?: string;
  approved_by?: string;
  title: string;
  command_type: string;
  command_payload: string;
  device_ids: string;
  risk_level: string;
  effective_risk: string;
  status: string;
  target_env: string;
  change_ticket: string;
  note: string;
  emergency: boolean;
  bypass_reason: string;
  approval_note: string;
  expires_at: string;
  approved_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

interface CommandExecution {
  id: string;
  task_id: string;
  device_id: string;
  device_name?: string;
  status: string;
  output?: string;
  error?: string;
  exit_code?: number;
  duration_ms?: number;
  sent_at?: string;
  started_at?: string;
  finished_at?: string;
}

interface TaskDetail extends CommandTask {
  executions?: CommandExecution[];
}

interface Device {
  id: string;
  device_name: string;
  hostname: string;
  status: string;
}

type StatusFilter =
  | ""
  | "pending_approval"
  | "approved"
  | "denied"
  | "executing"
  | "completed"
  | "expired"
  | "cancelled";

const STATUS_FILTERS: StatusFilter[] = [
  "",
  "pending_approval",
  "approved",
  "denied",
  "executing",
  "completed",
  "expired",
  "cancelled",
];

const RISK_COLORS: Record<string, string> = {
  L1: "bg-green-100 text-green-800",
  L2: "bg-yellow-100 text-yellow-800",
  L3: "bg-red-100 text-red-800",
};

const STATUS_COLORS: Record<string, string> = {
  pending_approval: "bg-yellow-100 text-yellow-800",
  approved: "bg-blue-100 text-blue-800",
  denied: "bg-red-100 text-red-800",
  executing: "bg-indigo-100 text-indigo-800",
  partial_done: "bg-orange-100 text-orange-800",
  completed: "bg-green-100 text-green-800",
  failed: "bg-red-100 text-red-800",
  expired: "bg-gray-100 text-gray-800",
  cancelled: "bg-gray-100 text-gray-500",
};

const EXEC_STATUS_ICONS: Record<string, string> = {
  pending: "⏳",
  sent: "📤",
  running: "🔄",
  succeeded: "✅",
  failed: "❌",
  timeout: "⏰",
  skipped: "⏭️",
};

const POLL_INTERVAL_MS = 15_000;

function CommandTasksContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = useDict("commandTasks", lang);
  const ct = useDict("common", lang);
  const { user } = useAuth();

  const [tasks, setTasks] = useState<CommandTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");

  const [selectedTask, setSelectedTask] = useState<TaskDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const [showNewModal, setShowNewModal] = useState(false);
  const [devices, setDevices] = useState<Device[]>([]);
  const [devicesLoading, setDevicesLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const [formTitle, setFormTitle] = useState("");
  const [formCommandType, setFormCommandType] = useState("shell");
  const [formCommandPayload, setFormCommandPayload] = useState("");
  const [formDeviceIds, setFormDeviceIds] = useState<string[]>([]);
  const [formRiskLevel, setFormRiskLevel] = useState("L1");
  const [formTargetEnv, setFormTargetEnv] = useState("");
  const [formNote, setFormNote] = useState("");
  const [formEmergency, setFormEmergency] = useState(false);
  const [formBypassReason, setFormBypassReason] = useState("");

  const [approvalNote, setApprovalNote] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  const fetchTasks = useCallback(async () => {
    try {
      const endpoint = statusFilter
        ? `/tenants/${tenantId}/command-tasks?status=${statusFilter}`
        : `/tenants/${tenantId}/command-tasks`;
      const data = await api.get<any>(endpoint);
      setTasks(Array.isArray(data) ? data : data?.items ?? []);
    } catch (error) {
      console.error("Failed to fetch command tasks:", error);
    } finally {
      setLoading(false);
    }
  }, [tenantId, statusFilter]);

  useEffect(() => {
    setLoading(true);
    fetchTasks();
    const interval = setInterval(fetchTasks, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchTasks]);

  const fetchDetail = async (id: string) => {
    setDetailLoading(true);
    try {
      const data = await api.get<TaskDetail>(
        `/tenants/${tenantId}/command-tasks/${id}`
      );
      setSelectedTask(data);
    } catch (error) {
      console.error("Failed to fetch task detail:", error);
    } finally {
      setDetailLoading(false);
    }
  };

  const openNewModal = async () => {
    setShowNewModal(true);
    setDevicesLoading(true);
    try {
      const data = await api.get<any>(`/tenants/${tenantId}/devices`);
      setDevices(Array.isArray(data) ? data : data?.items ?? []);
    } catch (error) {
      console.error("Failed to fetch devices:", error);
    } finally {
      setDevicesLoading(false);
    }
  };

  const resetForm = () => {
    setFormTitle("");
    setFormCommandType("shell");
    setFormCommandPayload("");
    setFormDeviceIds([]);
    setFormRiskLevel("L1");
    setFormTargetEnv("");
    setFormNote("");
    setFormEmergency(false);
    setFormBypassReason("");
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await api.post(`/tenants/${tenantId}/command-tasks`, {
        title: formTitle,
        command_type: formCommandType,
        command_payload: formCommandPayload,
        device_ids: formDeviceIds,
        risk_level: formRiskLevel,
        target_env: formTargetEnv,
        note: formNote,
        emergency: formEmergency,
        bypass_reason: formEmergency ? formBypassReason : "",
      });
      setShowNewModal(false);
      resetForm();
      fetchTasks();
    } catch (error) {
      console.error("Failed to create task:", error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleApprove = async (taskId: string) => {
    setActionLoading(true);
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${taskId}/approve`, {
        note: approvalNote,
      });
      setApprovalNote("");
      if (selectedTask?.id === taskId) await fetchDetail(taskId);
      fetchTasks();
    } catch (error) {
      console.error("Failed to approve task:", error);
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeny = async (taskId: string) => {
    setActionLoading(true);
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${taskId}/deny`, {
        note: approvalNote,
      });
      setApprovalNote("");
      if (selectedTask?.id === taskId) await fetchDetail(taskId);
      fetchTasks();
    } catch (error) {
      console.error("Failed to deny task:", error);
    } finally {
      setActionLoading(false);
    }
  };

  const handleCancel = async (taskId: string) => {
    setActionLoading(true);
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${taskId}/cancel`);
      if (selectedTask?.id === taskId) await fetchDetail(taskId);
      fetchTasks();
    } catch (error) {
      console.error("Failed to cancel task:", error);
    } finally {
      setActionLoading(false);
    }
  };

  const toggleDevice = (deviceId: string) => {
    setFormDeviceIds((prev) =>
      prev.includes(deviceId)
        ? prev.filter((id) => id !== deviceId)
        : [...prev, deviceId]
    );
  };

  const parseDeviceIds = (raw: string): string[] => {
    try {
      return JSON.parse(raw);
    } catch {
      return [];
    }
  };

  const formatTime = (iso: string) => {
    try {
      return new Date(iso).toLocaleString();
    } catch {
      return iso;
    }
  };

  // ── Detail View ──
  if (selectedTask) {
    const task = selectedTask;
    const deviceIds = parseDeviceIds(task.device_ids);
    const isPending = task.status === "pending_approval";
    const isCreator = user?.id === task.created_by;

    return (
      <div className="space-y-6">
        <button
          onClick={() => setSelectedTask(null)}
          className="text-sm text-blue-600 hover:text-blue-800 flex items-center gap-1"
        >
          ← {ct.back}
        </button>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 space-y-6">
          <div className="flex items-start justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-gray-900">
                {task.title}
              </h1>
              <div className="mt-2 flex items-center gap-3 flex-wrap">
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${RISK_COLORS[task.effective_risk] || RISK_COLORS[task.risk_level] || "bg-gray-100 text-gray-800"}`}
                >
                  {task.effective_risk || task.risk_level}
                </span>
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[task.status] || "bg-gray-100 text-gray-800"}`}
                >
                  {(t as any)[`status_${task.status}`] || task.status}
                </span>
                {task.emergency && (
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-600 text-white">
                    {(t as any).emergency}
                  </span>
                )}
              </div>
            </div>

            {isPending && (
              <div className="flex items-center gap-2">
                {isCreator && (
                  <button
                    onClick={() => handleCancel(task.id)}
                    disabled={actionLoading}
                    className="px-3 py-1.5 text-sm border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 disabled:opacity-50"
                  >
                    {ct.cancel}
                  </button>
                )}
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-500">{(t as any).commandType}:</span>
              <span className="ml-2 font-medium text-gray-900">
                {task.command_type}
              </span>
            </div>
            <div>
              <span className="text-gray-500">{(t as any).createdBy}:</span>
              <span className="ml-2 font-medium text-gray-900">
                {task.created_by_name || task.created_by}
              </span>
            </div>
            <div>
              <span className="text-gray-500">{(t as any).createdAt}:</span>
              <span className="ml-2 text-gray-900">
                {formatTime(task.created_at)}
              </span>
            </div>
            <div>
              <span className="text-gray-500">{(t as any).deviceCount}:</span>
              <span className="ml-2 text-gray-900">{deviceIds.length}</span>
            </div>
            {task.target_env && (
              <div>
                <span className="text-gray-500">
                  {(t as any).targetEnv}:
                </span>
                <span className="ml-2 text-gray-900">{task.target_env}</span>
              </div>
            )}
            {task.change_ticket && (
              <div>
                <span className="text-gray-500">
                  {(t as any).changeTicket}:
                </span>
                <span className="ml-2 text-gray-900">
                  {task.change_ticket}
                </span>
              </div>
            )}
          </div>

          <div>
            <span className="text-sm text-gray-500">
              {(t as any).commandContent}:
            </span>
            <pre className="mt-1 bg-gray-900 text-green-400 text-sm rounded-md p-4 overflow-x-auto">
              {task.command_payload}
            </pre>
          </div>

          {task.note && (
            <div>
              <span className="text-sm text-gray-500">{(t as any).note}:</span>
              <p className="mt-1 text-sm text-gray-700">{task.note}</p>
            </div>
          )}

          {task.approval_note && (
            <div>
              <span className="text-sm text-gray-500">
                {(t as any).approvalNote}:
              </span>
              <p className="mt-1 text-sm text-gray-700">
                {task.approval_note}
              </p>
            </div>
          )}

          {/* Approve / Deny section for pending tasks */}
          {isPending && !isCreator && (
            <div className="border-t pt-4 space-y-3">
              <h3 className="text-sm font-medium text-gray-900">
                {(t as any).approvalAction}
              </h3>
              <textarea
                value={approvalNote}
                onChange={(e) => setApprovalNote(e.target.value)}
                placeholder={(t as any).approvalNotePlaceholder}
                rows={2}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
              <div className="flex gap-3">
                <button
                  onClick={() => handleApprove(task.id)}
                  disabled={actionLoading}
                  className="px-4 py-2 bg-green-600 text-white rounded-md text-sm font-medium hover:bg-green-700 disabled:opacity-50"
                >
                  {(t as any).approve}
                </button>
                <button
                  onClick={() => handleDeny(task.id)}
                  disabled={actionLoading}
                  className="px-4 py-2 bg-red-600 text-white rounded-md text-sm font-medium hover:bg-red-700 disabled:opacity-50"
                >
                  {(t as any).deny}
                </button>
              </div>
            </div>
          )}

          {/* Execution Results */}
          <div className="border-t pt-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-3">
              {(t as any).executionResults}
            </h3>
            {detailLoading ? (
              <div className="text-center py-4 text-gray-500">
                <div className="inline-block animate-spin rounded-full h-6 w-6 border-4 border-gray-200 border-t-blue-600 mb-2" />
                <p>{ct.loading}</p>
              </div>
            ) : !task.executions || task.executions.length === 0 ? (
              <p className="text-sm text-gray-500">
                {(t as any).noExecutions}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                        {(t as any).device}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                        {ct.status}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                        {(t as any).output}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                        {(t as any).exitCode}
                      </th>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                        {(t as any).duration}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {task.executions.map((exec) => (
                      <tr key={exec.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 text-sm font-medium text-gray-900">
                          {exec.device_name || exec.device_id}
                        </td>
                        <td className="px-4 py-3 text-sm">
                          <span className="whitespace-nowrap">
                            {EXEC_STATUS_ICONS[exec.status] || "•"}{" "}
                            {(t as any)[`exec_${exec.status}`] || exec.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-700 max-w-xs">
                          {exec.output ? (
                            <pre className="whitespace-pre-wrap text-xs bg-gray-50 rounded p-2 max-h-32 overflow-y-auto">
                              {exec.output}
                            </pre>
                          ) : exec.error ? (
                            <pre className="whitespace-pre-wrap text-xs text-red-600 bg-red-50 rounded p-2 max-h-32 overflow-y-auto">
                              {exec.error}
                            </pre>
                          ) : (
                            <span className="text-gray-400">—</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500">
                          {exec.exit_code != null ? exec.exit_code : "—"}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500">
                          {exec.duration_ms != null
                            ? `${(exec.duration_ms / 1000).toFixed(1)}s`
                            : "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // ── List View ──
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">
          {(t as any).title}
        </h1>
        <button
          onClick={openNewModal}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700"
        >
          {(t as any).newTask}
        </button>
      </div>

      {/* Status filter tabs */}
      <div className="flex flex-wrap gap-1 border-b border-gray-200">
        {STATUS_FILTERS.map((sf) => (
          <button
            key={sf}
            onClick={() => setStatusFilter(sf)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              statusFilter === sf
                ? "border-blue-600 text-blue-600"
                : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
            }`}
          >
            {sf === ""
              ? (t as any).filterAll
              : ((t as any)[`status_${sf}`] || sf)}
          </button>
        ))}
      </div>

      {/* Task list */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4" />
            <p>{ct.loading}</p>
          </div>
        ) : tasks.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            {(t as any).noTasks}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).taskTitle}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).command}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).riskLevel}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {ct.status}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).deviceCount}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).createdBy}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).createdAt}
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                    {ct.actions}
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {tasks.map((task) => {
                  const deviceIds = parseDeviceIds(task.device_ids);
                  const isPending = task.status === "pending_approval";
                  const isCreator = user?.id === task.created_by;

                  return (
                    <tr
                      key={task.id}
                      className="hover:bg-gray-50 cursor-pointer"
                      onClick={() => fetchDetail(task.id)}
                    >
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        <div className="flex items-center gap-2">
                          {task.title}
                          {task.emergency && (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-red-600 text-white">
                              {(t as any).emergency}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500 max-w-[200px]">
                        <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded truncate block overflow-hidden">
                          {task.command_payload}
                        </code>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${RISK_COLORS[task.risk_level] || "bg-gray-100 text-gray-800"}`}
                        >
                          {task.risk_level}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[task.status] || "bg-gray-100 text-gray-800"}`}
                        >
                          {(t as any)[`status_${task.status}`] || task.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {deviceIds.length}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {task.created_by_name || task.created_by}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatTime(task.created_at)}
                      </td>
                      <td
                        className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {isPending && isCreator && (
                          <button
                            onClick={() => handleCancel(task.id)}
                            disabled={actionLoading}
                            className="text-red-600 hover:text-red-900 disabled:opacity-50"
                          >
                            {ct.cancel}
                          </button>
                        )}
                        {isPending && !isCreator && (
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => handleApprove(task.id)}
                              disabled={actionLoading}
                              className="text-green-600 hover:text-green-900 disabled:opacity-50"
                            >
                              {(t as any).approve}
                            </button>
                            <button
                              onClick={() => handleDeny(task.id)}
                              disabled={actionLoading}
                              className="text-red-600 hover:text-red-900 disabled:opacity-50"
                            >
                              {(t as any).deny}
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* New Task Modal */}
      {showNewModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6">
            <h2 className="text-xl font-semibold mb-4">{(t as any).newTask}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              {/* Title */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).taskTitle}
                </label>
                <input
                  type="text"
                  required
                  value={formTitle}
                  onChange={(e) => setFormTitle(e.target.value)}
                  placeholder={(t as any).taskTitlePlaceholder}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>

              {/* Command Type */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).commandType}
                </label>
                <div className="flex gap-4">
                  {["shell", "tool"].map((ct_val) => (
                    <label key={ct_val} className="flex items-center gap-2 text-sm">
                      <input
                        type="radio"
                        name="commandType"
                        value={ct_val}
                        checked={formCommandType === ct_val}
                        onChange={() => setFormCommandType(ct_val)}
                        className="text-blue-600"
                      />
                      {ct_val === "shell"
                        ? (t as any).typeShell
                        : (t as any).typeTool}
                    </label>
                  ))}
                </div>
              </div>

              {/* Command Content */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).commandContent}
                </label>
                <textarea
                  required
                  value={formCommandPayload}
                  onChange={(e) => setFormCommandPayload(e.target.value)}
                  placeholder={(t as any).commandContentPlaceholder}
                  rows={3}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
                />
              </div>

              {/* Target Devices */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).targetDevices} ({formDeviceIds.length}{" "}
                  {(t as any).selected})
                </label>
                <div className="border border-gray-300 rounded-md max-h-40 overflow-y-auto p-2">
                  {devicesLoading ? (
                    <p className="text-sm text-gray-500 text-center py-2">
                      {ct.loading}
                    </p>
                  ) : devices.length === 0 ? (
                    <p className="text-sm text-gray-500 text-center py-2">
                      {(t as any).noDevicesAvailable}
                    </p>
                  ) : (
                    devices.map((d) => (
                      <label
                        key={d.id}
                        className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-gray-50 cursor-pointer text-sm"
                      >
                        <input
                          type="checkbox"
                          checked={formDeviceIds.includes(d.id)}
                          onChange={() => toggleDevice(d.id)}
                          className="rounded text-blue-600"
                        />
                        <span className="font-medium text-gray-900">
                          {d.device_name}
                        </span>
                        <span className="text-gray-400 text-xs">
                          {d.hostname}
                        </span>
                      </label>
                    ))
                  )}
                </div>
              </div>

              {/* Risk Level */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).riskLevel}
                </label>
                <div className="flex gap-3">
                  {(["L1", "L2", "L3"] as const).map((level) => (
                    <label
                      key={level}
                      className={`flex items-center gap-2 px-3 py-2 rounded-md border cursor-pointer text-sm transition-colors ${
                        formRiskLevel === level
                          ? level === "L1"
                            ? "border-green-500 bg-green-50"
                            : level === "L2"
                              ? "border-yellow-500 bg-yellow-50"
                              : "border-red-500 bg-red-50"
                          : "border-gray-200 hover:border-gray-400"
                      }`}
                    >
                      <input
                        type="radio"
                        name="riskLevel"
                        value={level}
                        checked={formRiskLevel === level}
                        onChange={() => setFormRiskLevel(level)}
                        className="sr-only"
                      />
                      <span
                        className={`w-2 h-2 rounded-full ${level === "L1" ? "bg-green-500" : level === "L2" ? "bg-yellow-500" : "bg-red-500"}`}
                      />
                      {level} — {(t as any)[`risk_${level}`]}
                    </label>
                  ))}
                </div>
              </div>

              {/* Target Env */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).targetEnv}
                </label>
                <select
                  value={formTargetEnv}
                  onChange={(e) => setFormTargetEnv(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  <option value="">{(t as any).selectEnv}</option>
                  <option value="production">{(t as any).envProduction}</option>
                  <option value="staging">{(t as any).envStaging}</option>
                  <option value="testing">{(t as any).envTesting}</option>
                  <option value="dr">{(t as any).envDR}</option>
                </select>
              </div>

              {/* Note */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).note}
                </label>
                <textarea
                  value={formNote}
                  onChange={(e) => setFormNote(e.target.value)}
                  placeholder={(t as any).notePlaceholder}
                  rows={2}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>

              {/* Emergency */}
              <div className="space-y-2">
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={formEmergency}
                    onChange={(e) => setFormEmergency(e.target.checked)}
                    className="rounded text-red-600"
                  />
                  <span className="font-medium text-red-700">
                    {(t as any).emergencyMode}
                  </span>
                </label>
                {formEmergency && (
                  <div>
                    <label className="block text-sm font-medium text-red-700 mb-1">
                      {(t as any).emergencyReason}
                    </label>
                    <textarea
                      required
                      value={formBypassReason}
                      onChange={(e) => setFormBypassReason(e.target.value)}
                      placeholder={(t as any).emergencyReasonPlaceholder}
                      rows={2}
                      className="w-full border border-red-300 rounded-md px-3 py-2 text-sm bg-red-50"
                    />
                  </div>
                )}
              </div>

              {/* Actions */}
              <div className="flex justify-end space-x-3 pt-4 border-t">
                <button
                  type="button"
                  onClick={() => {
                    setShowNewModal(false);
                    resetForm();
                  }}
                  className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm"
                >
                  {ct.cancel}
                </button>
                <button
                  type="submit"
                  disabled={submitting || formDeviceIds.length === 0}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {submitting ? (t as any).submitting : (t as any).submit}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default function CommandTasksPage() {
  const params = useParams();
  const tenantId = params.tenantId as string;
  return <CommandTasksContent tenantId={tenantId} />;
}
