package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractExecPath(t *testing.T) {
	tests := []struct {
		name  string
		plist string
		want  string
	}{
		{
			name: "standard plist",
			plist: `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.claumon.dashboard</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/test/.local/bin/claumon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>`,
			want: "/Users/test/.local/bin/claumon",
		},
		{
			name:  "no ProgramArguments",
			plist: `<dict><key>Label</key><string>test</string></dict>`,
			want:  "",
		},
		{
			name:  "empty string",
			plist: "",
			want:  "",
		},
		{
			name: "path with spaces",
			plist: `<key>ProgramArguments</key>
    <array>
        <string>/Users/test/my apps/claumon</string>
    </array>`,
			want: "/Users/test/my apps/claumon",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExecPath(tt.plist)
			if got != tt.want {
				t.Errorf("extractExecPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRestartCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := RestartCommand(); got != "" {
		t.Errorf("RestartCommand() without a plist = %q, want \"\"", got)
	}

	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, darwinLabel+".plist"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	want := "launchctl kickstart -k gui/$(id -u)/com.claumon.dashboard"
	if got := RestartCommand(); got != want {
		t.Errorf("RestartCommand() = %q, want %q", got, want)
	}
}
