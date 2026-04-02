package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const systemdUnitPath = "/etc/systemd/system/sing-box.service"

type systemdManager struct{}

func systemdUnitContent(binPath, dataDir string) string {
	return fmt.Sprintf(`[Unit]
Description=sing-box Panel Service
After=network.target nss-lookup.target

[Service]
Type=simple
ExecStart=%s serve --data-dir %s
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, binPath, dataDir)
}

func (m *systemdManager) Install(binPath, dataDir string) error {
	content := systemdUnitContent(binPath, dataDir)
	if err := os.WriteFile(systemdUnitPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	if err := exec.Command("systemctl", "enable", "sing-box").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

func (m *systemdManager) Uninstall() error {
	exec.Command("systemctl", "stop", "sing-box").Run()
	exec.Command("systemctl", "disable", "sing-box").Run()
	os.Remove(systemdUnitPath)
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

func (m *systemdManager) Start() error {
	return exec.Command("systemctl", "start", "sing-box").Run()
}

func (m *systemdManager) Stop() error {
	return exec.Command("systemctl", "stop", "sing-box").Run()
}

func (m *systemdManager) Restart() error {
	return exec.Command("systemctl", "restart", "sing-box").Run()
}

func (m *systemdManager) Status() (string, error) {
	out, err := exec.Command("systemctl", "is-active", "sing-box").Output()
	status := strings.TrimSpace(string(out))
	if err != nil {
		if status == "inactive" {
			return "stopped", nil
		}
		return "not-installed", nil
	}
	if status == "active" {
		return "running", nil
	}
	return "stopped", nil
}
