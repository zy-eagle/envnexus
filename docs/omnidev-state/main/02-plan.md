# 开发计划: Agent Loop (ReAct) 架构改造

## 任务总览

共 7 个模块，按依赖顺序执行。

---

## M1: Tool 接口扩展 & Schema 定义
> 依赖: 无 | 优先级: P0

- [ ] M1.1 在 `tools/tool.go` 中新增 `ToolParamSchema` / `ToolParamProperty` 结构体
- [ ] M1.2 `Tool` 接口新增 `Parameters() *ToolParamSchema` 方法
- [ ] M1.3 `Registry` 新增 `ToOpenAITools()` 方法，将所有工具转为 Function Calling 格式
- [ ] M1.4 为全部 33 个工具实现 `Parameters()` 方法:
  - [ ] network: read_config, read_proxy, ping, dns_lookup, traceroute, port_scan, http_check, route_table, tls_check (9个)
  - [ ] system: info, disk_usage, process_list, env_vars, file_info, file_tail, dir_list, installed_apps, event_log, runtime_deps, shell_exec, proxy_toggle, config_modify (13个)
  - [ ] container: docker_inspect, docker_compose, kubectl_diagnose (3个)
  - [ ] database: mysql_check, postgres_check, redis_check, mongo_check (4个)
  - [ ] service: restart, container_reload (2个)
  - [ ] cache: rebuild (1个)
  - [ ] network: flush_dns (1个)

## M2: LLM Router 协议层扩展
> 依赖: M1 | 优先级: P0

- [ ] M2.1 `router.go` 新增 `ToolDefinition` / `ToolCall` / `FunctionCall` 结构体
- [ ] M2.2 `CompletionRequest` 增加 `Tools []ToolDefinition` 字段
- [ ] M2.3 `CompletionResponse` 增加 `ToolCalls []ToolCall` 字段
- [ ] M2.4 `Message` 结构体扩展: 支持 `ToolCallID` / `Name` 字段 (用于 role=tool 消息)

## M3: LLM Provider 适配 (Function Calling)
> 依赖: M2 | 优先级: P0

- [ ] M3.1 OpenAI Provider: 请求体增加 `tools`，响应解析 `tool_calls`，Message 支持 role=tool/assistant+tool_calls
- [ ] M3.2 DeepSeek Provider: 同 OpenAI 兼容协议适配
- [ ] M3.3 OpenRouter Provider: 同 OpenAI 兼容协议适配
- [ ] M3.4 Anthropic Provider: 适配 Anthropic tool_use 协议
- [ ] M3.5 Gemini Provider: 适配 Gemini function calling 协议
- [ ] M3.6 Ollama Provider: 适配 Ollama tools 协议 (OpenAI 兼容)

## M4: Agent Loop 引擎
> 依赖: M2, M3 | 优先级: P0

- [ ] M4.1 新建 `internal/agent/loop.go`，实现 `AgentLoop` 结构体
- [ ] M4.2 实现 `Run()` 方法: ReAct 循环核心逻辑
- [ ] M4.3 实现工具执行 + 审批机制: ReadOnly 直接执行，Write 暂停等审批
- [ ] M4.4 实现 SSE 回调: `OnThinking` / `OnToolStart` / `OnToolResult` / `OnApprovalRequired` / `OnMessage`
- [ ] M4.5 实现最大迭代次数限制 (默认 10)
- [ ] M4.6 System Prompt 设计: 告知 LLM 它是一个本地 IT 诊断助手，拥有哪些能力

## M5: API Server 新端点
> 依赖: M4 | 优先级: P0

- [ ] M5.1 新增 `POST /local/v1/chat` 端点，SSE 流式输出
- [ ] M5.2 实现 `ChatRequest` 结构体: `messages []Message` + `session_id`
- [ ] M5.3 SSE 事件类型: `thinking` / `tool_start` / `tool_result` / `approval_required` / `message` / `error`
- [ ] M5.4 实现审批回调机制: 前端通过 `POST /local/v1/chat/:session_id/approve` 提交审批结果
- [ ] M5.5 在 `bootstrap.go` 中初始化 `AgentLoop` 并注入到 `LocalServer`

## M6: Desktop 前端适配
> 依赖: M5 | 优先级: P1

- [ ] M6.1 `preload.ts` 新增 `sendChat` / `onChatEvent` / `approveChatTool` API
- [ ] M6.2 `main.ts` 新增 `send-chat` IPC handler (SSE 解析，转发多种事件)
- [ ] M6.3 `main.ts` 新增 `approve-chat-tool` IPC handler
- [ ] M6.4 `index.html` 诊断对话页面改用新 chat 端点
- [ ] M6.5 `index.html` 支持显示工具调用过程 (折叠面板: 工具名、参数、结果)
- [ ] M6.6 `index.html` 支持内联审批卡片 (Write 工具触发时弹出批准/拒绝按钮)
- [ ] M6.7 `index.html` i18n 补充新增文案

## M7: 集成测试 & 收尾
> 依赖: M6 | 优先级: P2

- [ ] M7.1 端到端测试: 用户发送 "查看 IP" → LLM 调用 read_network_config → 返回结果
- [ ] M7.2 审批流测试: 用户发送 "重启 nginx" → 弹出审批 → 批准/拒绝
- [ ] M7.3 多轮对话测试: 连续提问保持上下文
- [ ] M7.4 LLM 降级测试: 无 LLM 可用时的 fallback 行为
