package tmpltools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SSHKey looks up one of the current user's SSH keys.
func SSHKey(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	bs, err := os.ReadFile(filepath.Join(home, ".ssh", name))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(bs)), nil
}

// SSHKnownHost looks up a known host entry for the given host.
func SSHKnownHost(name string) (string, error) {
	ksCmd := exec.Command("ssh-keyscan", name)
	out, err := ksCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
