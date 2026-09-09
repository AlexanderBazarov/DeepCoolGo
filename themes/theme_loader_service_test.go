package themes_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeLoaderServiceLoad(t *testing.T) {
	dir := t.TempDir()
	runner := filepath.Join(dir, "loader")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", runner, "./testdata/loader")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build loader: %v\n%s", err, output)
	}

	invalidFile := filepath.Join(dir, "invalid.so")
	if err := os.WriteFile(invalidFile, []byte("invalid plugin"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		packagePath string
		path        string
		want        string
		wantError   bool
	}{
		{
			name:        "load and pass all CPU data",
			packagePath: "./testdata/valid",
			want:        "37.5/82.25/4.2/63.5",
		},
		{
			name:        "custom theme example",
			packagePath: "../plugins/custom",
			want:        "Rectangle",
		},
		{
			name:      "empty path",
			want:      "theme plugin path is empty",
			wantError: true,
		},
		{
			name:      "missing file",
			path:      filepath.Join(dir, "missing.so"),
			want:      "read theme plugin",
			wantError: true,
		},
		{
			name:      "directory",
			path:      dir,
			want:      "is not a regular file",
			wantError: true,
		},
		{
			name:      "invalid plugin file",
			path:      invalidFile,
			want:      "load theme plugin",
			wantError: true,
		},
		{
			name:        "missing Screen",
			packagePath: "./testdata/missing_screen",
			want:        "find Screen in theme plugin",
			wantError:   true,
		},
		{
			name:        "wrong Screen signature",
			packagePath: "./testdata/wrong_signature",
			want:        "Screen must be func(themes.CPUData) []deepcool.FrameObject",
			wantError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pluginPath := test.path
			if test.packagePath != "" {
				pluginPath = filepath.Join(t.TempDir(), "theme.so")
				cmd := exec.Command("go", "build", "-buildvcs=false", "-buildmode=plugin", "-o", pluginPath, test.packagePath)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("build plugin: %v\n%s", err, output)
				}
			}

			cmd := exec.Command(runner, pluginPath)
			output, err := cmd.CombinedOutput()
			if (err != nil) != test.wantError {
				t.Fatalf("load theme: %v, wantError=%v\n%s", err, test.wantError, output)
			}
			if test.wantError {
				if !strings.Contains(string(output), test.want) {
					t.Fatalf("error %q does not contain %q", output, test.want)
				}
			} else if got := strings.TrimSpace(string(output)); got != test.want {
				t.Fatalf("Screen() returned %q, want %q", got, test.want)
			}
		})
	}
}
