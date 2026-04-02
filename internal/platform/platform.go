package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	InitSystemd = "systemd"
	InitOpenRC  = "openrc"
	InitUnknown = "unknown"
)

type Info struct {
	OS         string
	Arch       string
	InitSystem string // "systemd", "openrc", or "unknown"
	DataDir    string // e.g. /usr/local/etc/sing-box
	BinPath    string // e.g. /usr/local/bin/sing-box
	LogDir     string // e.g. /var/log/sing-box
}

func Detect() *Info {
	info := &Info{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		DataDir: "/usr/local/etc/sing-box",
		BinPath: "/usr/local/bin/sing-box",
		LogDir:  "/var/log/sing-box",
	}

	info.InitSystem = detectInitSystem()
	return info
}

func detectInitSystem() string {
	data, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return InitUnknown
	}
	return parseInitSystem(strings.TrimSpace(string(data)))
}

func parseInitSystem(pid1 string) string {
	switch {
	case pid1 == "systemd":
		return InitSystemd
	case pid1 == "init" || pid1 == "openrc-init":
		return InitOpenRC
	case pid1 == "":
		return InitUnknown
	default:
		return InitUnknown
	}
}

func (info *Info) DBPath() string {
	return filepath.Join(info.DataDir, "panel.db")
}

func (info *Info) ServiceName() string {
	return "sing-box"
}
