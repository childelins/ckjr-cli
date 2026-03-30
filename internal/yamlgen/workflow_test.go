package yamlgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/workflow"
)

func TestInitWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asset.yaml")

	if err := InitWorkflowFile(path, "asset"); err != nil {
		t.Fatalf("InitWorkflowFile() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	cfg, err := workflow.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Name != "asset" {
		t.Errorf("Name = %q, want asset", cfg.Name)
	}
	if cfg.Description != "asset" {
		t.Errorf("Description = %q, want asset", cfg.Description)
	}
	if _, ok := cfg.Workflows["workflow-name"]; !ok {
		t.Fatal("workflow-name not found")
	}
	wf := cfg.Workflows["workflow-name"]
	if wf.Description != "工作流描述" {
		t.Errorf("Description = %q, want 工作流描述", wf.Description)
	}
	if len(wf.Triggers) != 0 {
		t.Errorf("Triggers = %v, want empty", wf.Triggers)
	}
	if len(wf.Inputs) != 0 {
		t.Errorf("Inputs = %v, want empty", wf.Inputs)
	}
	if len(wf.Steps) != 0 {
		t.Errorf("Steps = %v, want empty", wf.Steps)
	}
}

func TestInitWorkflowFile_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.yaml")
	os.WriteFile(path, []byte("test"), 0644)

	err := InitWorkflowFile(path, "test")
	if err == nil {
		t.Error("expected file exists error")
	}
}
