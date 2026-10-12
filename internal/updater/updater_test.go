package updater

import (
	"runtime"
	"testing"
)

func TestAssetName(t *testing.T) {
	name := AssetName()
	if name == "" {
		t.Fatal("AssetName returned empty string")
	}
	want := "claumon-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Errorf("AssetName() = %q, want %q", name, want)
	}
}

func TestIsHomebrewPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/Cellar/claumon/0.21.0/bin/claumon", true},
		{"/usr/local/Cellar/claumon/0.21.0/bin/claumon", true},
		{"/home/linuxbrew/.linuxbrew/Cellar/claumon/0.21.0/bin/claumon", true},
		{"/usr/local/bin/claumon", false},
		{"/home/me/.local/bin/claumon", false},
		{"/opt/homebrew/Cellar/other/1.0/bin/claumon", false},
		{`C:\Users\me\bin\claumon.exe`, false},
	}
	for _, tt := range tests {
		if got := isHomebrewPath(tt.path); got != tt.want {
			t.Errorf("isHomebrewPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestUpdateCommand(t *testing.T) {
	// The test binary is never in a Homebrew Cellar.
	if got := UpdateCommand("systemctl --user restart claumon"); got != "claumon update" {
		t.Errorf("UpdateCommand() = %q, want %q", got, "claumon update")
	}
}

func TestBrewCommand(t *testing.T) {
	tests := []struct {
		restart string
		want    string
	}{
		{"", "brew update && brew upgrade claumon"},
		{"systemctl --user restart claumon", "brew update && brew upgrade claumon && systemctl --user restart claumon"},
	}
	for _, tt := range tests {
		if got := BrewCommand(tt.restart); got != tt.want {
			t.Errorf("BrewCommand(%q) = %q, want %q", tt.restart, got, tt.want)
		}
	}
}

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v0.7.1", "v0.7.1", false},
		{"v0.7.1", "v0.8.0", true},
		{"0.7.1", "v0.7.1", false},
		{"dev", "v0.8.0", false},
		{"v0.7.0", "v0.7.1", true},
	}
	for _, tt := range tests {
		got := NeedsUpdate(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("NeedsUpdate(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}
