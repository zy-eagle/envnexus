package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// EventSink receives EnvNexus session/approval events and pushes them to the
// corresponding Feishu chat as conversational messages or interactive cards.
// It is called by platform-api handlers whenever a session event occurs.
type EventSink struct {
	client *Client
	bridge *ChatBridge
}

func NewEventSink(client *Client, bridge *ChatBridge) *EventSink {
	return &EventSink{client: client, bridge: bridge}
}

// SessionEvent represents an incoming platform event to push to Feishu.
type SessionEventPayload struct {
	EventType string          `json:"event_type"`
	SessionID string          `json:"session_id"`
	DeviceID  string          `json:"device_id"`
	TenantID  string          `json:"tenant_id"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// HandleEvent routes a session event to the appropriate Feishu chat.
// This is the main entry point called from platform-api whenever a session
// event occurs (diagnosis progress, tool result, approval request, etc.).
func (s *EventSink) HandleEvent(ctx context.Context, evt SessionEventPayload) {
	if s.client == nil || !s.client.Enabled() {
		return
	}

	chatID := s.bridge.ChatForSession(ctx, evt.SessionID)
	if chatID == "" {
		chatID = s.bridge.ChatForDevice(ctx, evt.DeviceID)
	}
	if chatID == "" {
		return
	}

	switch evt.EventType {
	case "diagnosis.started":
		s.pushText(ctx, chatID, "🔄 诊断已启动，正在采集系统信息...")

	case "diagnosis.collecting":
		step := extractField(evt.Data, "step")
		s.pushText(ctx, chatID, fmt.Sprintf("📊 正在采集: %s", step))

	case "diagnosis.analyzing":
		s.pushText(ctx, chatID, "🧠 AI 正在分析采集的数据...")

	case "diagnosis.result":
		summary := extractField(evt.Data, "summary")
		findings := extractStringSlice(evt.Data, "findings")
		card := NewDiagnosisResultCard(evt.DeviceID, evt.SessionID, summary, findings)
		s.pushCard(ctx, chatID, card)

	case "tool.proposed":
		toolName := extractField(evt.Data, "tool_name")
		riskLevel := extractField(evt.Data, "risk_level")
		desc := extractField(evt.Data, "description")
		s.pushText(ctx, chatID, fmt.Sprintf(
			"🔧 AI 建议执行修复操作:\n\n**工具**: `%s`\n**风险等级**: %s\n**描述**: %s",
			toolName, riskLevel, desc,
		))

	case "approval.created":
		approvalID := extractField(evt.Data, "approval_id")
		toolName := extractField(evt.Data, "tool_name")
		riskLevel := extractField(evt.Data, "risk_level")
		desc := extractField(evt.Data, "description")
		card := NewApprovalCard(evt.TenantID, evt.DeviceID, evt.SessionID, approvalID, toolName, riskLevel, desc)
		s.pushCard(ctx, chatID, card)

	case "approval.approved":
		approvalID := extractField(evt.Data, "approval_id")
		s.pushText(ctx, chatID, fmt.Sprintf("✅ 审批 `%s` 已通过，正在执行修复...", approvalID))

	case "approval.denied":
		approvalID := extractField(evt.Data, "approval_id")
		reason := extractField(evt.Data, "reason")
		msg := fmt.Sprintf("❌ 审批 `%s` 已被拒绝", approvalID)
		if reason != "" {
			msg += fmt.Sprintf("（原因: %s）", reason)
		}
		s.pushText(ctx, chatID, msg)

	case "tool.executing":
		toolName := extractField(evt.Data, "tool_name")
		s.pushText(ctx, chatID, fmt.Sprintf("⚙️ 正在执行: `%s` ...", toolName))

	case "tool.succeeded":
		toolName := extractField(evt.Data, "tool_name")
		output := extractField(evt.Data, "output")
		msg := fmt.Sprintf("✅ 工具 `%s` 执行成功", toolName)
		if output != "" {
			msg += fmt.Sprintf("\n\n```\n%s\n```", truncate(output, 500))
		}
		s.pushText(ctx, chatID, msg)

	case "tool.failed":
		toolName := extractField(evt.Data, "tool_name")
		errMsg := extractField(evt.Data, "error")
		s.pushText(ctx, chatID, fmt.Sprintf("❌ 工具 `%s` 执行失败: %s", toolName, errMsg))

	case "session.completed":
		summary := extractField(evt.Data, "summary")
		if summary == "" {
			summary = "会话结束"
		}
		card := NewSessionCompleteCard(evt.DeviceID, evt.SessionID, summary)
		s.pushCard(ctx, chatID, card)

	case "governance.drift":
		driftType := extractField(evt.Data, "drift_type")
		detail := extractField(evt.Data, "detail")
		card := NewAlertCard(evt.DeviceID, driftType, detail)
		s.pushCard(ctx, chatID, card)

	default:
		slog.Debug("[feishu-sink] Unhandled event type", "event_type", evt.EventType)
	}
}

func (s *EventSink) pushText(ctx context.Context, chatID, text string) {
	if err := s.client.SendTextMessage(ctx, "chat_id", chatID, text); err != nil {
		slog.Error("[feishu-sink] Push text failed", "chat_id", chatID, "error", err)
	}
}

func (s *EventSink) pushCard(ctx context.Context, chatID string, card *InteractiveCard) {
	if err := s.client.SendInteractiveCard(ctx, "chat_id", chatID, card); err != nil {
		slog.Error("[feishu-sink] Push card failed", "chat_id", chatID, "error", err)
	}
}

func extractField(raw json.RawMessage, key string) string {
	if raw == nil {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func extractStringSlice(raw json.RawMessage, key string) []string {
	if raw == nil {
		return nil
	}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	arr, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
