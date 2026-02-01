package prompts

import (
	"os"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	// 1. Test Embedded (Assuming templates/gap_analysis.md exists and is embedded)
	// We can't easily verify exact content without fragile tests, but we can check if it returns non-empty.
	// Or we can mock the fs? embed.FS is a global var in prompts.go, we can't easily swap it without changing code.
	// However, we can test the fallback logic.

	// 1. Test Embedded Success
	got, err := GetTemplate("gap_analysis.md", "")
	if err != nil {
		t.Fatalf("expected embedded gap_analysis.md to be found, got error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty content for embedded template")
	}

	// 1b. Test Invalid embedded file
	_, err = GetTemplate("non_existent_template.md", "")
	if err == nil {
		t.Error("expected error for non-existent embedded template, got nil")
	}

	// 2. Test External Override
	tmpFile, err := os.CreateTemp("", "prompt_override_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := "External Content"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	got, err = GetTemplate("any_name", tmpFile.Name())
	if err != nil {
		t.Fatalf("GetTemplate with override failed: %v", err)
	}
	if got != content {
		t.Errorf("GetTemplate() = %q, want %q", got, content)
	}

	// 3. Test Invalid External Path
	_, err = GetTemplate("name", "/path/to/nowhere")
	if err == nil {
		t.Error("expected error for invalid external path, got nil")
	}
}
