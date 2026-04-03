"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useParams } from "next/navigation";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { useDict } from "@/lib/i18n/dictionary";
import { api, APIError } from "@/lib/api/client";
import { useAuth } from "@/lib/auth/AuthContext";

function apiErrMessage(err: unknown, fallback: string): string {
  if (err instanceof APIError) return err.message || fallback;
  if (err instanceof Error) return err.message || fallback;
  return fallback;
}

interface CommandTask {
  id: string;
  tenant_id: string;
  created_by: string;
  created_by_name?: string;
  approver_id?: string;
  approved_by?: string;
  policy_snapshot_id?: string;
  title: string;
  command_type: string;
  command_payload: string;
  device_ids: string[];
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
  const [approvalPolicyNameById, setApprovalPolicyNameById] = useState<Record<string, string>>({});

  const [selectedTask, setSelectedTask] = useState<TaskDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const [showNewModal, setShowNewModal] = useState(false);
  const [editingTaskId, setEditingTaskId] = useState<string | null>(null);
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

  const [nlInput, setNlInput] = useState("");
  const [nlGenerating, setNlGenerating] = useState(false);
  const [nlError, setNlError] = useState("");
  const [nlMustSucceed, setNlMustSucceed] = useState(false);
  const [nlElapsed, setNlElapsed] = useState(0);
  const nlTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [formToolName, setFormToolName] = useState("");
  const [formToolParams, setFormToolParams] = useState<Record<string, string>>({});

  const [approvalNote, setApprovalNote] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [approvalFlowError, setApprovalFlowError] = useState("");

  const TOOL_CATALOG: {
    name: string;
    label: string;
    description: string;
    riskLevel: string;
    params: { key: string; label: string; required: boolean; placeholder: string; enumValues?: string[] }[];
  }[] = [
    {
      name: "system_info",
      label: lang === "zh" ? "系统信息" : "System Info",
      description: lang === "zh" ? "获取系统信息（OS、CPU、内存）" : "Get system info (OS, CPU, memory)",
      riskLevel: "L1",
      params: [],
    },
    {
      name: "disk_usage",
      label: lang === "zh" ? "磁盘使用" : "Disk Usage",
      description: lang === "zh" ? "检查磁盘使用情况" : "Check disk usage",
      riskLevel: "L1",
      params: [{ key: "path", label: lang === "zh" ? "路径" : "Path", required: false, placeholder: "/" }],
    },
    {
      name: "process_list",
      label: lang === "zh" ? "进程列表" : "Process List",
      description: lang === "zh" ? "列出运行中的进程" : "List running processes",
      riskLevel: "L1",
      params: [
        { key: "sort_by", label: lang === "zh" ? "排序方式" : "Sort By", required: false, placeholder: "cpu", enumValues: ["cpu", "memory", "pid"] },
        { key: "limit", label: lang === "zh" ? "数量限制" : "Limit", required: false, placeholder: "20" },
      ],
    },
    {
      name: "check_runtime_deps",
      label: lang === "zh" ? "运行时依赖" : "Runtime Deps",
      description: lang === "zh" ? "检测已安装的运行时环境" : "Check installed runtime dependencies",
      riskLevel: "L1",
      params: [{ key: "filter", label: lang === "zh" ? "过滤" : "Filter", required: false, placeholder: "python" }],
    },
    {
      name: "env_vars",
      label: lang === "zh" ? "环境变量" : "Env Variables",
      description: lang === "zh" ? "获取环境变量信息" : "Get environment variables",
      riskLevel: "L1",
      params: [{ key: "filter", label: lang === "zh" ? "过滤" : "Filter", required: false, placeholder: "PATH" }],
    },
    {
      name: "dir_list",
      label: lang === "zh" ? "目录列表" : "Directory List",
      description: lang === "zh" ? "列出指定目录内容" : "List directory contents",
      riskLevel: "L1",
      params: [
        { key: "path", label: lang === "zh" ? "路径" : "Path", required: true, placeholder: "/var/log" },
        { key: "depth", label: lang === "zh" ? "深度" : "Depth", required: false, placeholder: "1" },
      ],
    },
    {
      name: "installed_apps",
      label: lang === "zh" ? "已安装应用" : "Installed Apps",
      description: lang === "zh" ? "列出已安装的应用程序" : "List installed applications",
      riskLevel: "L1",
      params: [{ key: "filter", label: lang === "zh" ? "过滤" : "Filter", required: false, placeholder: "nginx" }],
    },
    {
      name: "file_rename",
      label: lang === "zh" ? "文件重命名" : "File Rename",
      description: lang === "zh" ? "重命名或移动文件/文件夹" : "Rename or move a file/directory",
      riskLevel: "L2",
      params: [
        { key: "source", label: lang === "zh" ? "原路径" : "Source", required: true, placeholder: "D:\\old_name" },
        { key: "destination", label: lang === "zh" ? "新路径" : "Destination", required: true, placeholder: "D:\\new_name" },
      ],
    },
    {
      name: "shell_exec",
      label: lang === "zh" ? "Shell 执行" : "Shell Exec",
      description: lang === "zh" ? "在设备上执行 shell 命令" : "Execute a shell command on device",
      riskLevel: "L2",
      params: [
        { key: "command", label: lang === "zh" ? "命令" : "Command", required: true, placeholder: "systemctl status nginx" },
        { key: "timeout", label: lang === "zh" ? "超时(秒)" : "Timeout(s)", required: false, placeholder: "30" },
      ],
    },
    {
      name: "config_modify",
      label: lang === "zh" ? "配置修改" : "Config Modify",
      description: lang === "zh" ? "修改配置文件内容" : "Modify configuration file content",
      riskLevel: "L3",
      params: [
        { key: "file_path", label: lang === "zh" ? "文件路径" : "File Path", required: true, placeholder: "/etc/nginx/nginx.conf" },
        { key: "action", label: lang === "zh" ? "操作" : "Action", required: true, placeholder: "replace", enumValues: ["replace", "append", "prepend"] },
        { key: "content", label: lang === "zh" ? "内容" : "Content", required: true, placeholder: "" },
      ],
    },
    {
      name: "port_scan",
      label: lang === "zh" ? "端口扫描" : "Port Scan",
      description: lang === "zh" ? "扫描目标主机端口" : "Scan common ports on a host",
      riskLevel: "L1",
      params: [{ key: "host", label: lang === "zh" ? "主机" : "Host", required: true, placeholder: "localhost" }],
    },
    {
      name: "ping",
      label: "Ping",
      description: lang === "zh" ? "Ping 目标主机" : "Ping a target host",
      riskLevel: "L1",
      params: [
        { key: "host", label: lang === "zh" ? "主机" : "Host", required: true, placeholder: "8.8.8.8" },
        { key: "count", label: lang === "zh" ? "次数" : "Count", required: false, placeholder: "4" },
      ],
    },
    {
      name: "dns_lookup",
      label: "DNS Lookup",
      description: lang === "zh" ? "DNS 查询" : "Perform DNS lookup",
      riskLevel: "L1",
      params: [{ key: "domain", label: lang === "zh" ? "域名" : "Domain", required: true, placeholder: "example.com" }],
    },
    {
      name: "http_check",
      label: "HTTP Check",
      description: lang === "zh" ? "检测 HTTP 端点" : "Check HTTP endpoint",
      riskLevel: "L1",
      params: [
        { key: "url", label: "URL", required: true, placeholder: "https://example.com/healthz" },
        { key: "method", label: lang === "zh" ? "方法" : "Method", required: false, placeholder: "GET", enumValues: ["GET", "POST", "HEAD"] },
      ],
    },
    {
      name: "docker_inspect",
      label: "Docker Inspect",
      description: lang === "zh" ? "检查 Docker 容器状态" : "Inspect Docker containers",
      riskLevel: "L1",
      params: [{ key: "container", label: lang === "zh" ? "容器" : "Container", required: false, placeholder: "nginx" }],
    },
    {
      name: "docker_compose",
      label: "Docker Compose",
      description: lang === "zh" ? "Docker Compose 操作" : "Docker Compose operations",
      riskLevel: "L2",
      params: [
        { key: "action", label: lang === "zh" ? "操作" : "Action", required: true, placeholder: "ps", enumValues: ["ps", "up", "down", "restart", "logs"] },
        { key: "service", label: lang === "zh" ? "服务" : "Service", required: false, placeholder: "" },
      ],
    },
    {
      name: "kubectl_diagnose",
      label: "Kubectl Diagnose",
      description: lang === "zh" ? "Kubernetes 集群诊断" : "Diagnose Kubernetes cluster",
      riskLevel: "L1",
      params: [
        { key: "action", label: lang === "zh" ? "操作" : "Action", required: true, placeholder: "get-pods", enumValues: ["cluster-info", "get-nodes", "get-pods", "describe-pod", "logs", "get-events", "get-services"] },
        { key: "namespace", label: "Namespace", required: false, placeholder: "default" },
        { key: "pod", label: "Pod", required: false, placeholder: "" },
      ],
    },
    {
      name: "mysql_check",
      label: "MySQL Check",
      description: lang === "zh" ? "检查 MySQL 连接和状态" : "Check MySQL connection and status",
      riskLevel: "L1",
      params: [
        { key: "host", label: lang === "zh" ? "主机" : "Host", required: false, placeholder: "localhost" },
        { key: "port", label: lang === "zh" ? "端口" : "Port", required: false, placeholder: "3306" },
      ],
    },
    {
      name: "redis_check",
      label: "Redis Check",
      description: lang === "zh" ? "检查 Redis 连接和状态" : "Check Redis connection and status",
      riskLevel: "L1",
      params: [
        { key: "host", label: lang === "zh" ? "主机" : "Host", required: false, placeholder: "localhost" },
        { key: "port", label: lang === "zh" ? "端口" : "Port", required: false, placeholder: "6379" },
      ],
    },
  ];

  const selectedToolDef = TOOL_CATALOG.find((t) => t.name === formToolName);

  const handleNlGenerate = async () => {
    if (!nlInput.trim()) return;
    setNlGenerating(true);
    setNlError("");
    setNlMustSucceed(true);
    setNlElapsed(0);
    nlTimerRef.current = setInterval(() => setNlElapsed((s) => s + 1), 1000);
    try {
      const data = await api.post<{ command: string; risk_level?: string; title?: string }>(
        `/tenants/${tenantId}/command-tasks/generate`,
        { prompt: nlInput },
        { timeoutMs: 5 * 60 * 1000 }
      );
      if (data && data.command) {
        setFormCommandPayload(data.command);
        setNlError("");
      } else {
        // Do NOT fall back to NL text as a runnable command. Block submission until fixed.
        setNlError(lang === "zh" ? "AI 返回了空命令：请重试生成，或清空自然语言后手动填写可执行命令" : "AI returned empty command: retry generation, or clear NL and enter a runnable command manually");
      }
      if (data?.risk_level) setFormRiskLevel(data.risk_level);
      if (data?.title && !formTitle) setFormTitle(data.title);
    } catch (err: unknown) {
      console.error("[nl-gen] Generate failed:", err);
      if (err instanceof APIError) {
        setNlError(err.message);
        return;
      }
      let msg: string;
      if (err instanceof Error && err.name === "AbortError") {
        msg =
          lang === "zh"
            ? "等待模型响应超过 5 分钟已中止，请稍后重试或换用更快的模型"
            : "Aborted after waiting 5 minutes for the model; retry or use a faster endpoint";
      } else if (err instanceof TypeError) {
        msg = lang === "zh" ? "网络连接超时，请检查服务端是否正常运行" : "Network timeout, check if server is running";
      } else {
        msg = err instanceof Error ? err.message : lang === "zh" ? "生成失败" : "Generation failed";
      }
      setNlError(
        lang === "zh"
          ? `命令生成失败: ${msg}（请重试生成，或清空自然语言后手动填写可执行命令）`
          : `Command generation failed: ${msg} (retry generation, or clear NL and enter a runnable command manually)`
      );
    } finally {
      if (nlTimerRef.current) { clearInterval(nlTimerRef.current); nlTimerRef.current = null; }
      setNlGenerating(false);
    }
  };

  const buildToolPayload = (): string => {
    const payload: Record<string, string> = {};
    if (selectedToolDef) {
      for (const p of selectedToolDef.params) {
        if (formToolParams[p.key]) payload[p.key] = formToolParams[p.key];
      }
    }
    return JSON.stringify({ tool_name: formToolName, params: payload });
  };

  const fetchTasks = useCallback(async () => {
    try {
      const endpoint = statusFilter
        ? `/tenants/${tenantId}/command-tasks?status=${statusFilter}`
        : `/tenants/${tenantId}/command-tasks`;
      const data = await api.get<any>(endpoint);
      // API standard response unwraps to `data`.
      // platform-api ListTasks returns `{ tasks: [...], total: n }`.
      setTasks(
        Array.isArray(data)
          ? data
          : (data?.tasks ?? data?.items ?? [])
      );
    } catch (error) {
      console.error("Failed to fetch command tasks:", error);
    } finally {
      setLoading(false);
    }
  }, [tenantId, statusFilter]);

  const fetchApprovalPolicies = useCallback(async () => {
    try {
      const data = await api.get<any>(`/tenants/${tenantId}/approval-policies`);
      const items: any[] = Array.isArray(data) ? data : data?.items ?? data?.policies ?? [];
      const map: Record<string, string> = {};
      for (const p of items) {
        if (p && typeof p.id === "string" && typeof p.name === "string") {
          map[p.id] = p.name;
        }
      }
      setApprovalPolicyNameById(map);
    } catch (error) {
      console.error("Failed to fetch approval policies:", error);
    }
  }, [tenantId]);

  useEffect(() => {
    setLoading(true);
    fetchTasks();
    fetchApprovalPolicies();
    const interval = setInterval(fetchTasks, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchTasks, fetchApprovalPolicies]);

  const getApprovalPolicyDisplay = (task: CommandTask) => {
    if (!task.policy_snapshot_id) return (t as any).tenantAdminApproval;
    const name = approvalPolicyNameById[task.policy_snapshot_id];
    if (name) return name;
    return `${(t as any).policyIdFallbackPrefix} ${task.policy_snapshot_id}`;
  };

  const fetchDetail = async (id: string) => {
    setDetailLoading(true);
    setApprovalFlowError("");
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
    setEditingTaskId(null);
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
    setNlInput("");
    setNlError("");
    setNlMustSucceed(false);
    setFormToolName("");
    setFormToolParams({});
    setEditingTaskId(null);
  };

  const openEditModal = async (task: TaskDetail) => {
    setShowNewModal(true);
    setEditingTaskId(task.id);
    setDevicesLoading(true);
    try {
      const data = await api.get<any>(`/tenants/${tenantId}/devices`);
      setDevices(Array.isArray(data) ? data : data?.items ?? []);
    } catch (error) {
      console.error("Failed to fetch devices:", error);
    } finally {
      setDevicesLoading(false);
    }

    setFormTitle(task.title || "");
    setFormCommandType(task.command_type || "shell");
    setFormCommandPayload(task.command_payload || "");
    setFormDeviceIds(parseDeviceIds(task.device_ids));
    setFormRiskLevel(task.risk_level || "L1");
    setFormTargetEnv(task.target_env || "");
    setFormNote(task.note || "");
    setFormEmergency(!!task.emergency);
    setFormBypassReason(task.bypass_reason || "");
    setNlInput("");
    setNlError("");
    setNlMustSucceed(false);
    setFormToolName("");
    setFormToolParams({});
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    // If user used NL input, generation must succeed; do not allow NL text to be dispatched as a command.
    if (formCommandType === "shell" && nlMustSucceed && nlInput.trim() && formCommandPayload.trim() === nlInput.trim()) {
      setNlError(lang === "zh" ? "自然语言不能直接下发到设备。请先生成成功，或清空自然语言并手动填写可执行命令。" : "Natural language cannot be dispatched. Generate a command successfully, or clear NL and enter a runnable command.");
      return;
    }
    setSubmitting(true);
    const payload = formCommandType === "tool" ? buildToolPayload() : formCommandPayload;
    const riskForTool = formCommandType === "tool" && selectedToolDef ? selectedToolDef.riskLevel : formRiskLevel;
    try {
      const body = {
        title: formTitle,
        command_type: formCommandType,
        command_payload: payload,
        device_ids: formDeviceIds,
        risk_level: formCommandType === "tool" ? riskForTool : formRiskLevel,
        target_env: formTargetEnv,
        note: formNote,
        emergency: formEmergency,
        bypass_reason: formEmergency ? formBypassReason : "",
      };
      if (editingTaskId) {
        await api.put(`/tenants/${tenantId}/command-tasks/${editingTaskId}`, body);
        if (selectedTask?.id === editingTaskId) await fetchDetail(editingTaskId);
      } else {
        await api.post(`/tenants/${tenantId}/command-tasks`, body);
      }
      setShowNewModal(false);
      resetForm();
      fetchTasks();
    } catch (error) {
      console.error("Failed to create task:", error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (taskId: string) => {
    if (
      !window.confirm(
        lang === "zh"
          ? "确定删除/归档该任务？\n- 未执行的任务会被彻底删除\n- 已执行/已失败/已完成的任务会被归档并从列表隐藏"
          : "Delete/Archive this task?\n- Unexecuted tasks will be permanently deleted\n- Executed/failed/completed tasks will be archived and hidden from the default list"
      )
    ) {
      return;
    }
    setActionLoading(true);
    try {
      await api.delete(`/tenants/${tenantId}/command-tasks/${taskId}`);
      if (selectedTask?.id === taskId) setSelectedTask(null);
      fetchTasks();
    } catch (error) {
      console.error("Failed to delete task:", error);
    } finally {
      setActionLoading(false);
    }
  };

  const handleApprove = async (taskId: string) => {
    setActionLoading(true);
    setApprovalFlowError("");
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${taskId}/approve`, {
        note: approvalNote,
      });
      setApprovalNote("");
      if (selectedTask?.id === taskId) await fetchDetail(taskId);
      fetchTasks();
    } catch (error) {
      console.error("Failed to approve task:", error);
      setApprovalFlowError(
        apiErrMessage(error, lang === "zh" ? "审批失败" : "Approval failed")
      );
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeny = async (taskId: string) => {
    setActionLoading(true);
    setApprovalFlowError("");
    try {
      await api.post(`/tenants/${tenantId}/command-tasks/${taskId}/deny`, {
        reason: approvalNote,
      });
      setApprovalNote("");
      if (selectedTask?.id === taskId) await fetchDetail(taskId);
      fetchTasks();
    } catch (error) {
      console.error("Failed to deny task:", error);
      setApprovalFlowError(
        apiErrMessage(error, lang === "zh" ? "拒绝失败" : "Deny failed")
      );
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

  const parseDeviceIds = (raw: any): string[] => {
    if (Array.isArray(raw)) return raw.filter((x) => typeof x === "string");
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        return Array.isArray(parsed)
          ? parsed.filter((x) => typeof x === "string")
          : [];
      } catch {
        return [];
      }
    }
    return [];
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
          onClick={() => {
            setSelectedTask(null);
            setApprovalFlowError("");
          }}
          className="text-sm text-blue-600 hover:text-blue-800 flex items-center gap-1"
        >
          ← {ct.back}
        </button>

        {approvalFlowError && (
          <div className="flex items-start justify-between gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
            <span>{approvalFlowError}</span>
            <button
              type="button"
              onClick={() => setApprovalFlowError("")}
              className="shrink-0 text-red-600 hover:text-red-900"
              aria-label={ct.close}
            >
              ×
            </button>
          </div>
        )}

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

            {(isCreator || (!isCreator && isPending)) && (
              <div className="flex items-center gap-2">
                {isCreator && (
                  <button
                    onClick={() => openEditModal(task)}
                    disabled={actionLoading}
                    className="px-3 py-1.5 text-sm border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 disabled:opacity-50"
                  >
                    {ct.edit || (lang === "zh" ? "编辑" : "Edit")}
                  </button>
                )}
                {isCreator && isPending && (
                  <button
                    onClick={() => handleCancel(task.id)}
                    disabled={actionLoading}
                    className="px-3 py-1.5 text-sm border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 disabled:opacity-50"
                  >
                    {ct.cancel}
                  </button>
                )}
                {isCreator && (
                  <button
                    onClick={() => handleDelete(task.id)}
                    disabled={actionLoading}
                    className="px-3 py-1.5 text-sm border border-red-300 text-red-700 rounded-md hover:bg-red-50 disabled:opacity-50"
                  >
                    {lang === "zh" ? "删除/归档" : "Delete/Archive"}
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
              <span className="text-gray-500">{(t as any).approvalPolicy}:</span>
              <span className="ml-2 font-medium text-gray-900">
                {getApprovalPolicyDisplay(task)}
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
      {approvalFlowError && (
        <div className="flex items-start justify-between gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          <span>{approvalFlowError}</span>
          <button
            type="button"
            onClick={() => setApprovalFlowError("")}
            className="shrink-0 text-red-600 hover:text-red-900"
            aria-label={ct.close}
          >
            ×
          </button>
        </div>
      )}

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
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    {(t as any).approvalPolicy}
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
                        {task.command_type === "tool" ? (
                          (() => {
                            try {
                              const parsed = JSON.parse(task.command_payload);
                              return (
                                <span className="inline-flex items-center gap-1.5 text-xs">
                                  <span className="bg-indigo-100 text-indigo-700 px-1.5 py-0.5 rounded font-medium">{parsed.tool_name}</span>
                                </span>
                              );
                            } catch {
                              return <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded truncate block overflow-hidden">{task.command_payload}</code>;
                            }
                          })()
                        ) : (
                          <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded truncate block overflow-hidden">
                            {task.command_payload}
                          </code>
                        )}
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
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <span className="truncate block max-w-[220px]" title={getApprovalPolicyDisplay(task)}>
                          {getApprovalPolicyDisplay(task)}
                        </span>
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
                        {isCreator && !isPending && (
                          <button
                            onClick={() => handleDelete(task.id)}
                            disabled={actionLoading}
                            className="text-gray-600 hover:text-gray-900 disabled:opacity-50"
                          >
                            {lang === "zh" ? "归档" : "Archive"}
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

              {/* Command Type Tabs */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {(t as any).commandType}
                </label>
                <div className="flex rounded-lg border border-gray-200 overflow-hidden">
                  {(["shell", "tool"] as const).map((ct_val) => (
                    <button
                      key={ct_val}
                      type="button"
                      onClick={() => {
                        setFormCommandType(ct_val);
                        setFormCommandPayload("");
                        setFormToolName("");
                        setFormToolParams({});
                      }}
                      className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
                        formCommandType === ct_val
                          ? "bg-blue-600 text-white"
                          : "bg-white text-gray-600 hover:bg-gray-50"
                      }`}
                    >
                      {ct_val === "shell" ? (t as any).typeShell : (t as any).typeTool}
                    </button>
                  ))}
                </div>
              </div>

              {/* Shell mode: NL input + command textarea */}
              {formCommandType === "shell" && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      {(t as any).nlPrompt}
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={nlInput}
                        onChange={(e) => setNlInput(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") { e.preventDefault(); handleNlGenerate(); }
                        }}
                        placeholder={(t as any).nlPlaceholder}
                        className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm"
                      />
                      <button
                        type="button"
                        onClick={handleNlGenerate}
                        disabled={nlGenerating || !nlInput.trim()}
                        className="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap flex items-center gap-1.5"
                      >
                        {nlGenerating ? (
                          <span className="inline-block animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full" />
                        ) : (
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
                        )}
                        {(t as any).nlGenerate}
                      </button>
                    </div>
                    {nlMustSucceed && nlInput.trim() && formCommandPayload.trim() === nlInput.trim() && (
                      <p className="mt-1 text-xs text-amber-600">
                        {lang === "zh"
                          ? "提示：当前“命令内容”与自然语言相同，无法下发。请点击“生成命令”成功后再提交，或清空自然语言后手动填写命令。"
                          : "Note: command content equals the NL prompt and cannot be dispatched. Generate successfully or clear NL and enter a command."}
                      </p>
                    )}
                    {nlGenerating ? (
                      <p className="mt-1 text-xs text-indigo-500 animate-pulse">
                        {lang === "zh" ? `AI 正在生成命令... ${nlElapsed}s` : `AI generating command... ${nlElapsed}s`}
                      </p>
                    ) : nlError ? (
                      <p className="mt-1 text-xs text-red-500">{nlError}</p>
                    ) : (
                      <p className="mt-1 text-xs text-gray-400">{(t as any).nlHint}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      {(t as any).commandContent}
                    </label>
                    <textarea
                      required
                      value={formCommandPayload}
                      onChange={(e) => setFormCommandPayload(e.target.value)}
                      placeholder={(t as any).commandContentPlaceholder}
                      rows={Math.max(3, Math.min(10, formCommandPayload.split("\n").length + 1))}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono bg-gray-900 text-green-400 resize-y"
                    />
                  </div>
                </>
              )}

              {/* Tool mode: tool selector + params */}
              {formCommandType === "tool" && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      {(t as any).selectTool}
                    </label>
                    <div className="grid grid-cols-2 gap-2 max-h-56 overflow-y-auto border border-gray-200 rounded-lg p-2">
                      {TOOL_CATALOG.map((tool) => (
                        <button
                          key={tool.name}
                          type="button"
                          onClick={() => {
                            setFormToolName(tool.name);
                            setFormToolParams({});
                          }}
                          className={`text-left p-2.5 rounded-md border transition-all text-sm ${
                            formToolName === tool.name
                              ? "border-blue-500 bg-blue-50 ring-1 ring-blue-500"
                              : "border-gray-200 hover:border-gray-300 hover:bg-gray-50"
                          }`}
                        >
                          <div className="font-medium text-gray-900 flex items-center justify-between">
                            {tool.label}
                            <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${RISK_COLORS[tool.riskLevel] || "bg-gray-100 text-gray-800"}`}>
                              {tool.riskLevel}
                            </span>
                          </div>
                          <div className="text-xs text-gray-500 mt-0.5 line-clamp-1">{tool.description}</div>
                        </button>
                      ))}
                    </div>
                  </div>

                  {selectedToolDef && selectedToolDef.params.length > 0 && (
                    <div className="space-y-3 border border-gray-200 rounded-lg p-4 bg-gray-50">
                      <h4 className="text-sm font-medium text-gray-700">{(t as any).toolParams}</h4>
                      {selectedToolDef.params.map((p) => (
                        <div key={p.key}>
                          <label className="block text-xs font-medium text-gray-600 mb-1">
                            {p.label} {p.required && <span className="text-red-500">*</span>}
                          </label>
                          {p.enumValues ? (
                            <select
                              value={formToolParams[p.key] || ""}
                              onChange={(e) =>
                                setFormToolParams((prev) => ({ ...prev, [p.key]: e.target.value }))
                              }
                              required={p.required}
                              className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
                            >
                              <option value="">-- {lang === "zh" ? "请选择" : "Select"} --</option>
                              {p.enumValues.map((v) => (
                                <option key={v} value={v}>{v}</option>
                              ))}
                            </select>
                          ) : (
                            <input
                              type="text"
                              value={formToolParams[p.key] || ""}
                              onChange={(e) =>
                                setFormToolParams((prev) => ({ ...prev, [p.key]: e.target.value }))
                              }
                              required={p.required}
                              placeholder={p.placeholder}
                              className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
                            />
                          )}
                        </div>
                      ))}
                    </div>
                  )}

                  {selectedToolDef && selectedToolDef.params.length === 0 && (
                    <p className="text-sm text-gray-500 italic">{(t as any).toolNoParams}</p>
                  )}
                </>
              )}

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
                  disabled={
                    submitting ||
                    formDeviceIds.length === 0 ||
                    (formCommandType === "shell" && !formCommandPayload.trim()) ||
                    (formCommandType === "shell" && nlMustSucceed && nlInput.trim() && formCommandPayload.trim() === nlInput.trim()) ||
                    (formCommandType === "tool" && !formToolName)
                  }
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
