package lifecycle

import (
	"os"
	"runtime"
)

// RuntimeEnvironment is sent on platform heartbeat so NL command generation and audits see real execution context.
type RuntimeEnvironment struct {
	OSVersion string `json:"os_version,omitempty"`
	Shell     string `json:"shell,omitempty"`
	WorkDir   string `json:"work_dir,omitempty"`
	User      string `json:"user,omitempty"`
}

// CollectRuntimeEnvironment captures local fields safe to report to the platform.
func CollectRuntimeEnvironment() *RuntimeEnvironment {
	wd, _ := os.Getwd()
	env := &RuntimeEnvironment{WorkDir: wd}
	switch runtime.GOOS {
	case "windows":
		env.Shell = "powershell.exe -NoProfile -NonInteractive -Command"
		if v := os.Getenv("OS"); v != "" {
			env.OSVersion = v
		}
		if v := os.Getenv("USERPROFILE"); v != "" {
			env.User = v
		}
	default:
		env.Shell = "sh -c"
		if v := os.Getenv("HOME"); v != "" {
			env.User = v
		}
	}
	return env
}
