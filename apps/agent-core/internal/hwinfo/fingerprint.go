package hwinfo

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
)

type Component struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
}

func CollectComponents() []Component {
	collectors := []struct {
		ctype string
		fn    func() string
	}{
		{"cpu", collectCPU},
		{"board", collectBoard},
		{"mac", collectMAC},
		{"disk", collectDisk},
		{"gpu", collectGPU},
	}

	var components []Component
	for _, c := range collectors {
		raw := c.fn()
		if raw == "" {
			slog.Debug("[hwinfo] collector returned empty", "type", c.ctype)
			continue
		}
		h := sha256.Sum256([]byte(raw))
		components = append(components, Component{
			Type: c.ctype,
			Hash: fmt.Sprintf("%x", h),
		})
	}
	if len(components) == 0 {
		slog.Warn("[hwinfo] all collectors returned empty, trying hostname fallback")
		if hostname := shellCmd(hostnameCmd()); hostname != "" {
			h := sha256.Sum256([]byte(hostname))
			components = append(components, Component{
				Type: "hostname",
				Hash: fmt.Sprintf("%x", h),
			})
		}
	}
	return components
}

func hostnameCmd() string {
	if runtime.GOOS == "windows" {
		return "hostname"
	}
	return "hostname"
}

func CompositeHash(components []Component) string {
	var parts []string
	for _, c := range components {
		parts = append(parts, c.Type+":"+c.Hash)
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", h)
}

func collectCPU() string {
	switch runtime.GOOS {
	case "windows":
		return psQuery("(Get-CimInstance Win32_Processor).ProcessorId")
	case "linux":
		return shellCmd("cat /proc/cpuinfo | grep 'model name' | head -1 | awk -F: '{print $2}'")
	case "darwin":
		return shellCmd("sysctl -n machdep.cpu.brand_string")
	}
	return ""
}

func collectBoard() string {
	switch runtime.GOOS {
	case "windows":
		return psQuery("(Get-CimInstance Win32_BaseBoard).SerialNumber")
	case "linux":
		return shellCmd("cat /sys/class/dmi/id/board_serial 2>/dev/null || echo unknown")
	case "darwin":
		return shellCmd("ioreg -l | grep IOPlatformSerialNumber | awk -F'\"' '{print $4}'")
	}
	return ""
}

func collectMAC() string {
	switch runtime.GOOS {
	case "windows":
		return psQuery("(Get-CimInstance Win32_NetworkAdapterConfiguration | Where-Object {$_.MACAddress -ne $null} | Select-Object -First 1).MACAddress")
	case "linux":
		return shellCmd("ip link show | grep ether | head -1 | awk '{print $2}'")
	case "darwin":
		return shellCmd("ifconfig en0 | grep ether | awk '{print $2}'")
	}
	return ""
}

func collectDisk() string {
	switch runtime.GOOS {
	case "windows":
		return psQuery("(Get-CimInstance Win32_DiskDrive | Select-Object -First 1).SerialNumber")
	case "linux":
		return shellCmd("lsblk -dno SERIAL /dev/sda 2>/dev/null || echo unknown")
	case "darwin":
		return shellCmd("system_profiler SPSerialATADataType 2>/dev/null | grep 'Serial Number' | head -1 | awk -F: '{print $2}'")
	}
	return ""
}

func collectGPU() string {
	switch runtime.GOOS {
	case "windows":
		return psQuery("(Get-CimInstance Win32_VideoController | Select-Object -First 1).PNPDeviceID")
	case "linux":
		return shellCmd("lspci | grep VGA | head -1")
	case "darwin":
		return shellCmd("system_profiler SPDisplaysDataType 2>/dev/null | grep 'Chipset Model' | head -1 | awk -F: '{print $2}'")
	}
	return ""
}

// psQuery runs a PowerShell command (preferred on modern Windows where wmic is deprecated)
func psQuery(command string) string {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err != nil {
		slog.Debug("[hwinfo] powershell query failed", "command", command, "error", err)
		return ""
	}
	result := strings.TrimSpace(string(out))
	if result == "" || strings.EqualFold(result, "null") {
		return ""
	}
	return result
}

func shellCmd(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
