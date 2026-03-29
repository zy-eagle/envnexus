# Agent-Core Diagnostic Tool Catalog

> Auto-maintained reference of all diagnostic tools registered in `agent-core`.
> Total: **33 tools** across 6 categories.

## Risk Level Legend

| Level | Meaning | Approval Required |
|-------|---------|-------------------|
| L0 | Read-only, no side effects | No |
| L1 | Low-risk write or privileged read | Yes (auto-approve in some modes) |
| L2 | Moderate-risk, may restart services or clear caches | Yes |
| L3 | High-risk, destructive or irreversible | Yes (manual only) |

---

## 1. Network (10 tools)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `read_network_config` | Reads local network interfaces and IP addresses | ✅ | L0 | `network/read_config.go` |
| `read_proxy_config` | Detects proxy configuration from env vars and system settings | ✅ | L0 | `network/read_proxy.go` |
| `dns.flush_cache` | Flushes the local DNS resolver cache | ❌ | L2 | `network/flush_dns.go` |
| `ping_host` | Tests TCP connectivity to a host:port (default port 80) | ✅ | L0 | `network/ping.go` |
| `dns_lookup` | Resolves a domain name to IP addresses (A/CNAME/MX) | ✅ | L0 | `network/dns_lookup.go` |
| `traceroute` | Traces network path to a destination host showing each hop | ✅ | L0 | `network/traceroute.go` |
| `port_scan` | Scans common TCP ports on a host to check availability | ✅ | L0 | `network/port_scan.go` |
| `http_check` | Tests HTTP/HTTPS endpoint availability, status code, headers, response time | ✅ | L0 | `network/http_check.go` |
| `read_route_table` | Reads the system routing table (gateways, network paths) | ✅ | L0 | `network/route_table.go` |
| `check_tls_cert` | Checks TLS/SSL certificate validity, expiry, and chain | ✅ | L0 | `network/tls_check.go` |

## 2. System (13 tools)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `read_system_info` | Reads OS, architecture, hostname, and basic system info | ✅ | L0 | `system/info.go` |
| `read_disk_usage` | Reads disk usage information for mounted volumes | ✅ | L0 | `system/disk_usage.go` |
| `read_process_list` | Lists running processes with basic info (PID, name, CPU, memory) | ✅ | L0 | `system/process_list.go` |
| `read_env_vars` | Reads environment variables (auto-filters sensitive values) | ✅ | L0 | `system/env_vars.go` |
| `read_file_info` | Checks file/directory existence, size, permissions, modification time | ✅ | L0 | `system/file_info.go` |
| `read_file_tail` | Reads the last N lines of a file (default 50, max 200) | ✅ | L0 | `system/file_tail.go` |
| `read_dir_list` | Lists files and subdirectories with size and modification time | ✅ | L0 | `system/dir_list.go` |
| `read_installed_apps` | Lists installed applications/packages (supports name filter) | ✅ | L0 | `system/installed_apps.go` |
| `read_event_log` | Reads recent system event logs (errors/warnings) | ✅ | L0 | `system/event_log.go` |
| `check_runtime_deps` | Checks common runtime dependencies (Java, Python, Node.js, .NET, Go, Docker, Git…) | ✅ | L0 | `system/runtime_deps.go` |
| `shell_exec` | Executes a whitelisted diagnostic command (ipconfig, netstat, ping, systeminfo…) | ✅ | L1 | `system/shell_exec.go` |
| `proxy.toggle` | Enable or disable system/application-level HTTP proxy | ❌ | L1 | `system/proxy_toggle.go` |
| `config.modify` | Modify a whitelisted configuration key in the agent env file | ❌ | L1 | `system/config_modify.go` |

## 3. Container — Docker & Kubernetes (3 tools)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `docker_inspect` | Docker daemon status, container list, logs, images, networks, volumes. Actions: `status\|ps\|logs\|images\|networks\|volumes` | ✅ | L0 | `container/docker_inspect.go` |
| `docker_compose_check` | Docker Compose project status, service logs, config validation. Actions: `ps\|logs\|config` | ✅ | L0 | `container/docker_compose.go` |
| `kubectl_diagnose` | K8s cluster diagnostics: cluster-info, nodes, pods, events, services, pod logs. Actions: `cluster-info\|get-nodes\|get-pods\|describe-pod\|logs\|get-events\|get-services` | ✅ | L0 | `container/kubectl_diagnose.go` |

## 4. Database (4 tools)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `mysql_check` | MySQL TCP connectivity + handshake version detection. Params: `host`, `port` (default 3306) | ✅ | L0 | `database/mysql_check.go` |
| `postgres_check` | PostgreSQL TCP connectivity + SSL probe. Params: `host`, `port` (default 5432) | ✅ | L0 | `database/postgres_check.go` |
| `redis_check` | Redis connectivity, PING/PONG, INFO server (version, memory, role). Params: `host`, `port` (default 6379) | ✅ | L0 | `database/redis_check.go` |
| `mongo_check` | MongoDB TCP connectivity + OP_MSG wire protocol probe. Params: `host`, `port` (default 27017) | ✅ | L0 | `database/mongo_check.go` |

## 5. Service (2 tools)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `service.restart` | Restarts a specified system service | ❌ | L2 | `service/restart.go` |
| `container.reload` | Reloads a Docker container or sends SIGHUP to a process. Modes: `docker\|process\|systemd` | ❌ | L2 | `service/container_reload.go` |

## 6. Cache (1 tool)

| Tool Name | Description | ReadOnly | Risk | Source |
|-----------|-------------|----------|------|--------|
| `cache.rebuild` | Clears and rebuilds a specified cache directory | ❌ | L2 | `cache/rebuild.go` |

---

## Diagnosis Problem Types & Tool Mapping

The diagnosis engine automatically selects tools based on the detected problem type:

| Problem Type | Matched Keywords | Tools Used |
|-------------|-----------------|------------|
| `network` | network, proxy, connect, ping, timeout, vpn, firewall… | `read_network_config`, `read_proxy_config`, `read_system_info`, `ping_host`, `dns_lookup`, `read_route_table`, `read_env_vars` |
| `dns` | dns, resolve, 域名 | `read_network_config`, `read_system_info`, `dns_lookup`, `read_env_vars` |
| `service` | service, restart, 服务, 重启 | `read_system_info`, `read_process_list`, `read_env_vars`, `http_check`, `read_event_log`, `check_runtime_deps` |
| `performance` | slow, cpu, memory, 慢, 卡 | `read_system_info`, `read_process_list`, `read_disk_usage` |
| `disk` | disk, space, 磁盘, 存储, 容量 | `read_system_info`, `read_disk_usage`, `read_dir_list` |
| `auth` | auth, password, 认证, 登录, 权限 | `read_system_info`, `read_env_vars`, `check_tls_cert` |
| `install` | install, setup, dependency, dll, missing, version… | `read_system_info`, `read_disk_usage`, `read_installed_apps`, `check_runtime_deps`, `read_event_log`, `read_env_vars`, `read_file_info` |
| `docker` | docker, container, compose, 容器, 镜像 | `read_system_info`, `docker_inspect`, `docker_compose_check`, `read_process_list`, `read_disk_usage`, `read_env_vars` |
| `kubernetes` | k8s, kubernetes, kubectl, pod, deployment, 集群, helm | `read_system_info`, `kubectl_diagnose`, `docker_inspect`, `read_process_list`, `read_env_vars`, `read_event_log` |
| `database` | mysql, postgres, redis, mongo, database, db, sql, 数据库, 缓存 | `read_system_info`, `mysql_check`, `postgres_check`, `redis_check`, `mongo_check`, `read_process_list`, `read_env_vars`, `read_disk_usage` |
| `general` | (fallback) | `read_system_info`, `read_network_config`, `read_proxy_config`, `read_env_vars` |

---

## Statistics

- **Total tools**: 33
- **Read-only (L0)**: 26
- **Low-risk write (L1)**: 3
- **Moderate-risk (L2)**: 4
- **High-risk (L3)**: 0
- **Problem types supported**: 11
