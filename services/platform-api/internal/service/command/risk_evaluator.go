package command

import "strings"

var l3Keywords = []string{
	"rm -rf", "mkfs", "dd if=", "fdisk", "format", "> /dev/",
	"chmod 777", "chown root", "iptables -F", "systemctl disable",
	"apt remove", "apt purge", "yum remove", "dnf remove",
	"docker compose up", "docker compose down",
	"reboot", "shutdown", "halt", "poweroff", "init 0", "init 6",
}

var l2Keywords = []string{
	"systemctl restart", "systemctl stop", "systemctl start",
	"service restart", "service stop", "service start",
	"kill", "pkill", "nginx -s reload",
	"docker restart", "docker stop", "docker rm",
	"redis-cli flushall", "redis-cli flushdb",
}

func EvaluateRisk(commandType, commandPayload string) string {
	if commandType == "tool" {
		return "L1"
	}
	lower := strings.ToLower(commandPayload)
	for _, kw := range l3Keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return "L3"
		}
	}
	for _, kw := range l2Keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return "L2"
		}
	}
	return "L1"
}

func EffectiveRisk(userRisk, systemRisk string) string {
	levels := map[string]int{"L1": 1, "L2": 2, "L3": 3}
	u, ok1 := levels[userRisk]
	s, ok2 := levels[systemRisk]
	if !ok1 {
		u = 1
	}
	if !ok2 {
		s = 1
	}
	if s > u {
		return systemRisk
	}
	return userRisk
}
