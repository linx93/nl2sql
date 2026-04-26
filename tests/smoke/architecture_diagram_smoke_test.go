package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureDiagramScriptReportsResolvedFonts(t *testing.T) {
	output := runPythonScript(t, "--print-fonts")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two font paths, got %q", output)
	}

	for _, line := range lines {
		if _, err := os.Stat(strings.TrimSpace(line)); err != nil {
			t.Fatalf("expected reported font path to exist: %v", err)
		}
	}
}

func TestArchitectureDiagramScriptSupportsCustomOutputPath(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "diagram.jpg")
	runPythonScript(t, "--output", outputPath)

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("expected custom output file to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty output file")
	}
}

func runPythonScript(t *testing.T, args ...string) string {
	t.Helper()

	scriptPath := filepath.Join("..", "..", "scripts", "generate_architecture_diagram.py")
	cmd := exec.Command("python", append([]string{scriptPath}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python script failed: %v\n%s", err, string(output))
	}

	return string(output)
}
