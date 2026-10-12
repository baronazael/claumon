package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestartCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := RestartCommand(); got != "" {
		t.Errorf("RestartCommand() without a unit = %q, want \"\"", got)
	}

	dir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, linuxServiceName+".service"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got, want := RestartCommand(), "systemctl --user restart claumon"; got != want {
		t.Errorf("RestartCommand() = %q, want %q", got, want)
	}
}
