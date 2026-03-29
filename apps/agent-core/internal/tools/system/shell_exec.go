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

// Whitelisted commands safe for diagnostics. Only the binary name (first arg) is checked.
var allowedCommands = map[string]map[string]bool{
	"windows": {
		// Network diagnostics
		"ipconfig":    true,
		"netstat":     true,
		"nslookup":    true,
		"ping":        true,
		"tracert":     true,
		"pathping":    true,
		"route":       true,
		"arp":         true,
		"getmac":      true,
		"nbtstat":     true,
		"netsh":       true,
		// System info
		"hostname":    true,
		"systeminfo":  true,
		"tasklist":    true,
		"whoami":      true,
		"net":         true,
		"wmic":        true,
		"sc":          true,
		"schtasks":    true,
		"certutil":    true,
		// File & text
		"findstr":     true,
		"type":        true,
		"dir":         true,
		"set":         true,
		// Explicitly blocked
		"powershell":  false,
		"cmd":         false,
		"reg":         false,
		"regedit":     false,
		"del":         false,
		"rmdir":       false,
		"rd":          false,
		"format":      false,
		"shutdown":    false,
	},
	"linux": {
		// Network diagnostics
		"ip":          true,
		"ifconfig":    true,
		"netstat":     true,
		"ss":          true,
		"nslookup":    true,
		"dig":         true,
		"ping":        true,
		"traceroute":  true,
		"tracepath":   true,
		"route":       true,
		"arp":         true,
		"ethtool":     true,
		"nmcli":       true,
		"resolvectl":  true,
		"iptables":    true,
		"nft":         true,
		"curl":        true,
		// System info
		"hostname":    true,
		"hostnamectl": true,
		"uname":       true,
		"uptime":      true,
		"free":        true,
		"df":          true,
		"ps":          true,
		"lsof":        true,
		"whoami":      true,
		"id":          true,
		"dmesg":       true,
		"lsblk":       true,
		"timedatectl": true,
		"systemctl":   true,
		"journalctl":  true,
		"env":         true,
		"printenv":    true,
		// File & text
		"cat":         true,
		"head":        true,
		"tail":        true,
		"grep":        true,
		"ls":          true,
		"stat":        true,
		"wc":          true,
		"awk":         true,
		"sed":         true,
		// Container & orchestration
		"docker":      true,
		"crictl":      true,
		"kubectl":     true,
		// Explicitly blocked
		"top":         false,
		"htop":        false,
		"wget":        false,
		"mount":       false,
		"umount":      false,
		"rm":          false,
		"mkfs":        false,
		"dd":          false,
		"shutdown":    false,
		"reboot":      false,
	},
	"darwin": {
		// Network diagnostics
		"ifconfig":    true,
		"netstat":     true,
		"nslookup":    true,
		"dig":         true,
		"ping":        true,
		"traceroute":  true,
		"route":       true,
		"arp":         true,
		"scutil":      true,
		"networksetup": true,
		"curl":        true,
		// System info
		"hostname":    true,
		"uname":       true,
		"uptime":      true,
		"df":          true,
		"ps":          true,
		"lsof":        true,
		"whoami":      true,
		"id":          true,
		"system_profiler": true,
		"sw_vers":     true,
		"dscacheutil": true,
		"pmset":       true,
		"diskutil":    true,
		"launchctl":   true,
		"env":         true,
		"printenv":    true,
		// File & text
		"cat":         true,
		"head":        true,
		"tail":        true,
		"grep":        true,
		"ls":          true,
		"stat":        true,
		"wc":          true,
		"awk":         true,
		"sed":         true,
		// Container & orchestration
		"docker":      true,
		"kubectl":     true,
		// Explicitly blocked
		"top":         false,
		"htop":        false,
		"rm":          false,
		"shutdown":    false,
		"reboot":      false,
	},
}

type ShellExecTool struct{}

func NewShellExecTool() *ShellExecTool { return &ShellExecTool{} }

func (t *ShellExecTool) Name() string { return "shell_exec" }
func (t *ShellExecTool) Description() string {
	return "Executes a whitelisted diagnostic command (e.g. ipconfig, netstat, ping, nslookup, systeminfo). Only safe read-only commands are allowed."
}
func (t *ShellExecTool) IsReadOnly() bool  { return true }
func (t *ShellExecTool) RiskLevel() string { return "L1" }

func (t *ShellExecTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	command, _ := params["command"].(string)
	if command == "" {
		return &tools.ToolResult{ToolName: t.Name(), Status: "failed", Error: "missing required parameter: command"}, nil
	}

	parts := parseCommand(command)
	if len(parts) == 0 {
		return &tools.ToolResult{ToolName: t.Name(), Status: "failed", Error: "empty command"}, nil
	}

	binary := strings.ToLower(parts[0])
	osAllowed := allowedCommands[runtime.GOOS]
	if osAllowed == nil {
		osAllowed = allowedCommands["linux"]
	}

	allowed, exists := osAllowed[binary]
	if !exists || !allowed {
		return &tools.ToolResult{
			ToolName: t.Name(),
			Status:   "blocked",
			Summary:  fmt.Sprintf("Command '%s' is not in the allowed whitelist for %s", binary, runtime.GOOS),
			Output: map[string]interface{}{
				"command":  command,
				"binary":   binary,
				"os":       runtime.GOOS,
				"allowed":  listAllowed(),
			},
		}, nil
	}

	start := time.Now()
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, parts[0], parts[1:]...)
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
			Summary:    fmt.Sprintf("Command '%s' exited with code %d", binary, exitCode),
			Output:     map[string]interface{}{"command": command, "exit_code": exitCode, "output": outputStr, "error": err.Error()},
			DurationMs: elapsed.Milliseconds(),
		}, nil
	}

	return &tools.ToolResult{
		ToolName:   t.Name(),
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Command '%s' completed successfully", binary),
		Output:     map[string]interface{}{"command": command, "exit_code": 0, "output": outputStr},
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

func listAllowed() []string {
	osAllowed := allowedCommands[runtime.GOOS]
	if osAllowed == nil {
		osAllowed = allowedCommands["linux"]
	}
	var list []string
	for cmd, ok := range osAllowed {
		if ok {
			list = append(list, cmd)
		}
	}
	return list
}
