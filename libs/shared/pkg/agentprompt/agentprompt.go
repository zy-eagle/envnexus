// Package agentprompt holds LLM system-prompt fragments shared by the on-device
// agent and the platform NL command generator so execution context stays aligned.
package agentprompt

import (
	"fmt"
	"strings"
)

// Snapshot describes the machine execution environment for LLM prompts.
// Fill from the local agent (runtime + env) or from a device row + optional heartbeat metadata.
type Snapshot struct {
	OS         string
	Arch       string
	Hostname   string
	DeviceName string

	OSVersion   string
	Shell       string // how remote/shell_exec invokes commands, when known
	WorkDir     string
	UserProfile string
	SystemRoot  string
	Processor   string
	Lang        string
	PathHead    string
	ExtraLines  []string
}

// NormalizeSnapshot fills missing shell labels and defaults legacy OS aliases.
func NormalizeSnapshot(s Snapshot) Snapshot {
	s.OS = strings.TrimSpace(strings.ToLower(s.OS))
	if s.OS == "" {
		s.OS = "windows"
	}
	switch s.OS {
	case "win32":
		s.OS = "windows"
	}
	if strings.TrimSpace(s.Shell) == "" {
		switch s.OS {
		case "windows":
			s.Shell = `Agent runs: powershell.exe -NoProfile -NonInteractive -Command "<command>"`
		default:
			s.Shell = `Agent runs: sh -c '<command>'`
		}
	}
	return s
}

// FormatSystemEnvironmentBlock renders the "System Environment" section used in
// both the agent loop system prompt and console NL command generation.
func FormatSystemEnvironmentBlock(s Snapshot) string {
	s = NormalizeSnapshot(s)
	var lines []string
	lines = append(lines, fmt.Sprintf("OS: %s", s.OS))
	if strings.TrimSpace(s.Arch) != "" {
		lines = append(lines, fmt.Sprintf("Architecture: %s", s.Arch))
	}
	if strings.TrimSpace(s.DeviceName) != "" {
		lines = append(lines, fmt.Sprintf("Device name: %s", s.DeviceName))
	}
	if strings.TrimSpace(s.Hostname) != "" {
		lines = append(lines, fmt.Sprintf("Hostname: %s", s.Hostname))
	}
	if strings.TrimSpace(s.OSVersion) != "" {
		lines = append(lines, fmt.Sprintf("OS version / build hint: %s", s.OSVersion))
	}
	if strings.TrimSpace(s.Processor) != "" {
		lines = append(lines, fmt.Sprintf("Processor: %s", s.Processor))
	}
	lines = append(lines, fmt.Sprintf("Command invocation: %s", s.Shell))

	switch s.OS {
	case "windows":
		lines = append(lines,
			"On Windows, shell_exec and remote shell use PowerShell 5.x+ — use PowerShell syntax (cmdlets, ';' to chain). Do NOT use '&&' or '||' (require PowerShell 7+); agents default to 5.x.",
			"Prefer PowerShell cmdlets (New-Item, Set-Content, Remove-Item, Get-ChildItem) over cmd.exe syntax. Avoid cmd idioms like 'echo.' or 'dir /s' — use Get-ChildItem -Recurse.",
		)
		if strings.TrimSpace(s.UserProfile) != "" {
			lines = append(lines, fmt.Sprintf("Home (USERPROFILE): %s", s.UserProfile))
		}
		if strings.TrimSpace(s.SystemRoot) != "" {
			lines = append(lines, fmt.Sprintf("SystemRoot: %s", s.SystemRoot))
		}
	default:
		lines = append(lines,
			"On this target, shell_exec and remote shell use POSIX sh -c. Use /bin/sh compatible syntax.",
		)
		if strings.TrimSpace(s.Lang) != "" {
			lines = append(lines, fmt.Sprintf("Locale: %s", s.Lang))
		}
		if strings.TrimSpace(s.UserProfile) != "" {
			lines = append(lines, fmt.Sprintf("Home: %s", s.UserProfile))
		}
	}

	if strings.TrimSpace(s.WorkDir) != "" {
		lines = append(lines, fmt.Sprintf("Agent working directory (last reported): %s", s.WorkDir))
	}
	if strings.TrimSpace(s.PathHead) != "" {
		lines = append(lines, fmt.Sprintf("PATH (excerpt): %s", s.PathHead))
	}
	for _, e := range s.ExtraLines {
		if t := strings.TrimSpace(e); t != "" {
			lines = append(lines, t)
		}
	}

	return "System Environment:\n" + strings.Join(lines, "\n")
}

func nlJSONContract() string {
	return `You convert operations requests into ONE command line. Your entire reply MUST be a single JSON object and nothing else.

Required JSON shape (all three keys):
{"command":"<one runnable command line for the target environment below>","risk_level":"<L1|L2|L3>","title":"<short title>"}

Hard rules:
- First character MUST be "{". Last MUST be "}". No text before or after the JSON.
- Do NOT explain, paraphrase the user request, restate these rules, or think step-by-step in the reply. No markdown, no code fences.
- Put the real command only inside the "command" string.`
}

func nlSyntaxRulesForTarget(s Snapshot) string {
	s = NormalizeSnapshot(s)
	if s.OS == "windows" {
		return `PowerShell / JSON specifics for the "command" field on this Windows target:
- Chain steps with semicolon ";" or newlines (\n). Never use "&&" or "||" (require PowerShell 7+; agents default to 5.x).
- Drive letters like E: are fine in quoted paths, e.g. New-Item -Path 'E:\folder\file.txt'.`
	}
	return `Shell specifics for the "command" field on this Unix-like target:
- The command is passed to sh -c. Use POSIX shell syntax; chain with ";" or newlines.`
}

// BuildNLCommandSystemPrompt is the full system message for platform NL → command JSON generation.
func BuildNLCommandSystemPrompt(s Snapshot) string {
	s = NormalizeSnapshot(s)
	return nlJSONContract() + "\n\n" + nlSyntaxRulesForTarget(s) + "\n\n" + FormatSystemEnvironmentBlock(s)
}

// BuildAgentLoopSystemPrompt is the full system message for the on-device agent chat loop.
func BuildAgentLoopSystemPrompt(s Snapshot) string {
	s = NormalizeSnapshot(s)
	return fmt.Sprintf(`You are EnvNexus Agent, a local IT diagnostic and operations assistant running on the user's machine.

%s

You have access to a set of tools that can inspect and operate on this machine. Use them to answer the user's questions accurately.

Guidelines:
- When the user asks about system state (IP, files, processes, etc.), call the appropriate tool to get real data. Do NOT guess or make up information.
- You may call multiple tools in sequence if needed to fully answer a question.
- Present results in a clear, readable format. Summarize key findings.
- ALWAYS call tools directly when the user requests an action. NEVER ask the user for confirmation yourself — the system has a built-in approval mechanism that will automatically prompt the user for approval on sensitive operations. Just call the tool and the system handles the rest.
- Do NOT simulate or role-play an approval process in your text responses. Do NOT say things like "Do you want me to proceed?" or "Please confirm". Simply call the appropriate tool.
- If a tool call fails, explain the error and suggest alternatives.
- When using shell_exec, generate commands appropriate for the shell described in System Environment above.
- Respond in the same language as the user's message.`, FormatSystemEnvironmentBlock(s))
}

// DefaultNLTargetWhenNoDevice is used when the console has not selected a device yet (legacy behavior: Windows).
func DefaultNLTargetWhenNoDevice() Snapshot {
	return NormalizeSnapshot(Snapshot{
		OS:         "windows",
		Arch:       "amd64",
		Hostname:   "(not tied to a device — commands assume Windows PowerShell)",
		DeviceName: "",
	})
}
