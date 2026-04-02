package service

import (
	"fmt"
	"regexp"
)

var validPathRe = regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`)

func validatePath(path string) error {
	if !validPathRe.MatchString(path) {
		return fmt.Errorf("invalid path %q: only [a-zA-Z0-9/._-] allowed", path)
	}
	return nil
}

type Manager interface {
	Install(binPath, dataDir string) error
	Uninstall() error
	Start() error
	Stop() error
	Restart() error
	Status() (string, error)
}

func NewManager(initSystem string) Manager {
	switch initSystem {
	case "systemd":
		return &systemdManager{}
	case "openrc":
		return &openrcManager{}
	default:
		return nil
	}
}
