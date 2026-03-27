package yaml

import (
	"testing"
	"testing/fstest"
)

func TestLoadRoutes(t *testing.T) {
	memFS := fstest.MapFS{
		"routes/agent.yaml":  {Data: []byte("name: agent\ndescription: test\nroutes: {}")},
		"routes/common.yaml": {Data: []byte("name: common\ndescription: common\nroutes: {}")},
		"routes/readme.txt":  {Data: []byte("ignored")},
		"routes/sub/.keep":   {Data: []byte("")},
	}
	loader := New(memFS)
	files, err := loader.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("LoadRoutes() got %d files, want 2", len(files))
	}
	if _, ok := files["agent.yaml"]; !ok {
		t.Error("LoadRoutes() missing agent.yaml")
	}
	if _, ok := files["common.yaml"]; !ok {
		t.Error("LoadRoutes() missing common.yaml")
	}
	if _, ok := files["readme.txt"]; ok {
		t.Error("LoadRoutes() should skip .txt files")
	}
}

func TestLoadRoutes_EmptyDir(t *testing.T) {
	memFS := fstest.MapFS{
		"routes/readme.txt": {Data: []byte("ignored")},
	}
	loader := New(memFS)
	files, err := loader.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("LoadRoutes() got %d files, want 0", len(files))
	}
}

func TestLoadRoutes_NonexistentDir(t *testing.T) {
	memFS := fstest.MapFS{}
	loader := New(memFS)
	_, err := loader.LoadRoutes()
	if err == nil {
		t.Fatal("LoadRoutes() expected error for nonexistent dir")
	}
}

func TestLoadWorkflows(t *testing.T) {
	memFS := fstest.MapFS{
		"workflows/agent.yaml": {Data: []byte("name: agent\nworkflows: {}")},
		"workflows/note.txt":   {Data: []byte("ignored")},
	}
	loader := New(memFS)
	files, err := loader.LoadWorkflows()
	if err != nil {
		t.Fatalf("LoadWorkflows() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("LoadWorkflows() got %d files, want 1", len(files))
	}
	if _, ok := files["agent.yaml"]; !ok {
		t.Error("LoadWorkflows() missing agent.yaml")
	}
}

func TestLoadWorkflows_NonexistentDir(t *testing.T) {
	memFS := fstest.MapFS{}
	loader := New(memFS)
	_, err := loader.LoadWorkflows()
	if err == nil {
		t.Fatal("LoadWorkflows() expected error for nonexistent dir")
	}
}
