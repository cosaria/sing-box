package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const openrcInitPath = "/etc/init.d/sing-box"

type openrcManager struct{}

func openrcInitContent(binPath, dataDir string) string {
	return fmt.Sprintf(`#!/sbin/openrc-run

description="sing-box Panel Service"

command="%s"
command_args="serve --data-dir %s"
command_background=true
pidfile="/run/sing-box.pid"

depend() {
	need net
	after firewall
}
`, binPath, dataDir)
}

func (m *openrcManager) Install(binPath, dataDir string) error {
	if err := validatePath(binPath); err != nil {
		return err
	}
	if err := validatePath(dataDir); err != nil {
		return err
	}
	content := openrcInitContent(binPath, dataDir)
	if err := os.WriteFile(openrcInitPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write init script: %w", err)
	}
	if err := exec.Command("rc-update", "add", "sing-box", "default").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

func (m *openrcManager) Uninstall() error {
	exec.Command("rc-service", "sing-box", "stop").Run()
	exec.Command("rc-update", "del", "sing-box", "default").Run()
	os.Remove(openrcInitPath)
	return nil
}

func (m *openrcManager) Start() error {
	return exec.Command("rc-service", "sing-box", "start").Run()
}

func (m *openrcManager) Stop() error {
	return exec.Command("rc-service", "sing-box", "stop").Run()
}

func (m *openrcManager) Restart() error {
	return exec.Command("rc-service", "sing-box", "restart").Run()
}

func (m *openrcManager) Status() (string, error) {
	out, err := exec.Command("rc-service", "sing-box", "status").Output()
	status := strings.TrimSpace(string(out))
	if err != nil {
		return "not-installed", nil
	}
	if strings.Contains(status, "started") {
		return "running", nil
	}
	return "stopped", nil
}
