package system

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

// safeReadOnlyCommands lists commands that are safe to run without user approval.
// Commands NOT in this list will trigger the approval flow — no hard blocking.
var safeReadOnlyCommands = map[string]map[string]bool{
	"windows": {
		"ipconfig": true, "netstat": true, "nslookup": true, "ping": true,
		"tracert": true, "pathping": true, "route": true, "arp": true,
		"getmac": true, "nbtstat": true, "netsh": true,
		"hostname": true, "systeminfo": true, "tasklist": true, "whoami": true,
		"net": true, "wmic": true, "sc": true, "schtasks": true, "certutil": true,
		"findstr": true, "type": true, "dir": true, "set": true, "where": true,
		"echo": true, "ver": true, "vol": true, "tree": true,
	},
	"linux": {
		"ip": true, "ifconfig": true, "netstat": true, "ss": true,
		"nslookup": true, "dig": true, "ping": true, "traceroute": true,
		"tracepath": true, "route": true, "arp": true, "ethtool": true,
		"nmcli": true, "resolvectl": true, "iptables": true, "nft": true, "curl": true,
		"hostname": true, "hostnamectl": true, "uname": true, "uptime": true,
		"free": true, "df": true, "ps": true, "lsof": true, "whoami": true,
		"id": true, "dmesg": true, "lsblk": true, "timedatectl": true,
		"systemctl": true, "journalctl": true, "env": true, "printenv": true,
		"cat": true, "head": true, "tail": true, "grep": true, "ls": true,
		"stat": true, "wc": true, "awk": true, "find": true, "which": true,
		"file": true, "du": true, "top": true, "htop": true,
		"docker": true, "crictl": true, "kubectl": true,
	},
	"darwin": {
		"ifconfig": true, "netstat": true, "nslookup": true, "dig": true,
		"ping": true, "traceroute": true, "route": true, "arp": true,
		"scutil": true, "networksetup": true, "curl": true,
		"hostname": true, "uname": true, "uptime": true, "df": true,
		"ps": true, "lsof": true, "whoami": true, "id": true,
		"system_profiler": true, "sw_vers": true, "dscacheutil": true,
		"pmset": true, "diskutil": true, "launchctl": true,
		"env": true, "printenv": true,
		"cat": true, "head": true, "tail": true, "grep": true, "ls": true,
		"stat": true, "wc": true, "awk": true, "find": true, "which": true,
		"file": true, "du": true, "top": true, "htop": true,
		"docker": true, "kubectl": true,
	},
}

type ShellExecTool struct{}

func NewShellExecTool() *ShellExecTool { return &ShellExecTool{} }

func (t *ShellExecTool) Name() string { return "shell_exec" }
func (t *ShellExecTool) Description() string {
	return "Executes a shell command on the local machine. Whitelisted diagnostic commands (e.g. ipconfig, netstat, ping, systeminfo, dir, ls, ps, df) run directly. Non-whitelisted or write commands (e.g. mkdir, copy, move) require user approval before execution."
}
func (t *ShellExecTool) IsReadOnly() bool  { return false }
func (t *ShellExecTool) RiskLevel() string { return "L2" }

func (t *ShellExecTool) NeedsApproval(params map[string]interface{}) bool {
	command, _ := params["command"].(string)
	if command == "" {
		return false
	}
	parts := parseCommand(command)
	if len(parts) == 0 {
		return false
	}
	binary := strings.ToLower(parts[0])
	safe := safeReadOnlyCommands[runtime.GOOS]
	if safe == nil {
		safe = safeReadOnlyCommands["linux"]
	}
	return !safe[binary]
}

func (t *ShellExecTool) Parameters() *tools.ParamSchema {
	return &tools.ParamSchema{
		Type: "object",
		Properties: map[string]tools.ParamProperty{
			"command": {
				Type:        "string",
				Description: "Shell command to execute. Whitelisted diagnostic commands (ipconfig, netstat, ping, systeminfo, dir, ls, ps, df, etc.) run directly. Other commands (mkdir, copy, move, etc.) require user approval first.",
			},
		},
		Required: []string{"command"},
	}
}

func (t *ShellExecTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return &tools.ToolResult{ToolName: t.Name(), Status: "failed", Error: "missing required parameter: command"}, nil
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return &tools.ToolResult{ToolName: t.Name(), Status: "failed", Error: "empty command"}, nil
	}

	start := time.Now()
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(execCtx, "cmd", "/C", command)
	default:
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	outputStr := strings.TrimSpace(string(output))

	const maxOutput = 8192
	if len(outputStr) > maxOutput {
		outputStr = outputStr[:maxOutput] + "\n... (output truncated)"
	}

	if err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return &tools.ToolResult{
			ToolName:   t.Name(),
			Status:     "succeeded",
			Summary:    fmt.Sprintf("Command exited with code %d (%dms)", exitCode, elapsed.Milliseconds()),
			Output:     map[string]interface{}{"command": command, "exit_code": exitCode, "output": outputStr, "error": err.Error(), "os": runtime.GOOS},
			DurationMs: elapsed.Milliseconds(),
		}, nil
	}

	return &tools.ToolResult{
		ToolName:   t.Name(),
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Command completed successfully (%dms)", elapsed.Milliseconds()),
		Output:     map[string]interface{}{"command": command, "exit_code": 0, "output": outputStr, "os": runtime.GOOS},
		DurationMs: elapsed.Milliseconds(),
	}, nil
}

func parseCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

