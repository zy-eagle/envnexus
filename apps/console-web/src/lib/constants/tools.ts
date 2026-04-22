export interface ToolParamDef {
  key: string;
  label: string;
  required: boolean;
  placeholder: string;
  enumValues?: string[];
}

export interface ToolDef {
  name: string;
  label: string;
  description: string;
  riskLevel: string;
  params: ToolParamDef[];
}

export function getToolCatalog(lang: string): ToolDef[] {
  return [
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
}
