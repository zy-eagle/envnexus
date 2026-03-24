package feishu

import "fmt"

// InteractiveCard builds Feishu interactive card messages.
// Ref: https://open.feishu.cn/document/common-capabilities/message-card/message-cards-content

type InteractiveCard struct {
	Config   *CardConfig  `json:"config,omitempty"`
	Header   *CardHeader  `json:"header,omitempty"`
	Elements []CardElement `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

type CardHeader struct {
	Title    *CardText `json:"title"`
	Template string    `json:"template,omitempty"`
}

type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type CardElement interface {
	cardElement()
}

type CardDiv struct {
	Tag  string    `json:"tag"`
	Text *CardText `json:"text,omitempty"`
}

func (CardDiv) cardElement() {}

type CardNote struct {
	Tag      string     `json:"tag"`
	Elements []*CardText `json:"elements"`
}

func (CardNote) cardElement() {}

type CardAction struct {
	Tag     string         `json:"tag"`
	Actions []CardButton   `json:"actions"`
	Layout  string         `json:"layout,omitempty"`
}

func (CardAction) cardElement() {}

type CardButton struct {
	Tag   string            `json:"tag"`
	Text  *CardText         `json:"text"`
	Type  string            `json:"type,omitempty"`
	URL   string            `json:"url,omitempty"`
	Value map[string]string `json:"value,omitempty"`
}

type CardHR struct {
	Tag string `json:"tag"`
}

func (CardHR) cardElement() {}

// -- Builder helpers --

// NewApprovalCard builds an interactive card for an approval request.
func NewApprovalCard(tenantID, deviceID, sessionID, approvalID, toolName, riskLevel, actionDesc string) *InteractiveCard {
	headerColor := "blue"
	if riskLevel == "L2" {
		headerColor = "orange"
	} else if riskLevel == "L3" {
		headerColor = "red"
	}

	return &InteractiveCard{
		Config: &CardConfig{WideScreenMode: true, EnableForward: true},
		Header: &CardHeader{
			Title:    &CardText{Tag: "plain_text", Content: "🔧 EnvNexus 修复审批请求"},
			Template: headerColor,
		},
		Elements: []CardElement{
			CardDiv{Tag: "div", Text: &CardText{
				Tag:     "lark_md",
				Content: "**设备**: " + deviceID + "\n**会话**: " + sessionID + "\n**工具**: `" + toolName + "`\n**风险等级**: " + riskLevel + "\n**操作描述**: " + actionDesc,
			}},
			CardHR{Tag: "hr"},
			CardAction{
				Tag:    "action",
				Layout: "bisected",
				Actions: []CardButton{
					{
						Tag:  "button",
						Text: &CardText{Tag: "plain_text", Content: "✅ 批准"},
						Type: "primary",
						Value: map[string]string{
							"action":      "approve",
							"approval_id": approvalID,
							"session_id":  sessionID,
						},
					},
					{
						Tag:  "button",
						Text: &CardText{Tag: "plain_text", Content: "❌ 拒绝"},
						Type: "danger",
						Value: map[string]string{
							"action":      "deny",
							"approval_id": approvalID,
							"session_id":  sessionID,
						},
					},
				},
			},
			CardNote{Tag: "note", Elements: []*CardText{
				{Tag: "plain_text", Content: "来自 EnvNexus 平台 · 租户 " + tenantID},
			}},
		},
	}
}

// NewDiagnosisResultCard builds a card to display diagnosis results.
func NewDiagnosisResultCard(deviceID, sessionID, summary string, findings []string) *InteractiveCard {
	findingsText := ""
	for i, f := range findings {
		findingsText += fmt.Sprintf("%d. %s\n", i+1, f)
	}
	if findingsText == "" {
		findingsText = "无异常发现"
	}

	return &InteractiveCard{
		Config: &CardConfig{WideScreenMode: true, EnableForward: true},
		Header: &CardHeader{
			Title:    &CardText{Tag: "plain_text", Content: "🔍 EnvNexus 诊断结果"},
			Template: "green",
		},
		Elements: []CardElement{
			CardDiv{Tag: "div", Text: &CardText{
				Tag:     "lark_md",
				Content: "**设备**: " + deviceID + "\n**会话**: " + sessionID + "\n\n**诊断摘要**:\n" + summary,
			}},
			CardHR{Tag: "hr"},
			CardDiv{Tag: "div", Text: &CardText{
				Tag:     "lark_md",
				Content: "**发现**:\n" + findingsText,
			}},
		},
	}
}

// NewSessionCompleteCard builds a card summarizing a completed diagnosis session.
func NewSessionCompleteCard(deviceID, sessionID, summary string) *InteractiveCard {
	return &InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    &CardText{Tag: "plain_text", Content: "✅ EnvNexus 诊断会话完成"},
			Template: "green",
		},
		Elements: []CardElement{
			CardDiv{Tag: "div", Text: &CardText{
				Tag:     "lark_md",
				Content: "**设备**: " + deviceID + "\n**会话**: " + sessionID + "\n\n" + summary,
			}},
			CardHR{Tag: "hr"},
			CardNote{Tag: "note", Elements: []*CardText{
				{Tag: "plain_text", Content: "发送任意消息可继续诊断 · /unbind 解绑设备"},
			}},
		},
	}
}

// NewAlertCard builds a card for governance drift or system alerts.
func NewAlertCard(deviceID, alertType, message string) *InteractiveCard {
	return &InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    &CardText{Tag: "plain_text", Content: "⚠️ EnvNexus 告警"},
			Template: "red",
		},
		Elements: []CardElement{
			CardDiv{Tag: "div", Text: &CardText{
				Tag:     "lark_md",
				Content: "**设备**: " + deviceID + "\n**类型**: " + alertType + "\n**详情**: " + message,
			}},
		},
	}
}
