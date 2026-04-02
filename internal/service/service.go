package service

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
