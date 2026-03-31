package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
	"github.com/childelins/ckjr-cli/internal/workflow"
)

// loadRouteFiles reads all .yaml files from configFS under routes/ directory.
func loadRouteFiles(t *testing.T) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := fs.WalkDir(configFS, "routes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		data, err := fs.ReadFile(configFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		files[path] = data
		return nil
	})
	if err != nil {
		t.Fatalf("load route files: %v", err)
	}
	return files
}

// loadWorkflowFiles reads all .yaml files from configFS under workflows/ directory.
func loadWorkflowFiles(t *testing.T) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := fs.WalkDir(configFS, "workflows", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		data, err := fs.ReadFile(configFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		files[path] = data
		return nil
	})
	if err != nil {
		t.Fatalf("load workflow files: %v", err)
	}
	return files
}

// validateRouteConfig checks structural integrity of a route config.
func validateRouteConfig(t *testing.T, filename string, cfg *router.RouteConfig) {
	t.Helper()
	if cfg.Name == "" {
		t.Errorf("%s: route config name is empty", filename)
	}
	if cfg.Description == "" {
		t.Errorf("%s: route config description is empty", filename)
	}
	if len(cfg.Routes) < 1 {
		t.Errorf("%s: route config has no routes", filename)
	}
}

// validateRouteFields checks field semantics of a route config.
func validateRouteFields(t *testing.T, filename string, cfg *router.RouteConfig) {
	t.Helper()
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true,
	}
	validTypes := map[string]bool{
		"": true, "string": true, "int": true, "float": true,
		"bool": true, "array": true, "path": true,
	}

	for actionName, route := range cfg.Routes {
		prefix := fmt.Sprintf("%s/routes/%s", filename, actionName)

		if !validMethods[route.Method] {
			t.Errorf("%s: invalid method %q", prefix, route.Method)
		}
		if !strings.HasPrefix(route.Path, "/") {
			t.Errorf("%s: path %q does not start with /", prefix, route.Path)
		}
		if route.Description == "" {
			t.Errorf("%s: description is empty", prefix)
		}

		for fieldName, field := range route.Template {
			fieldPrefix := fmt.Sprintf("%s/template/%s", prefix, fieldName)

			if field.Description == "" {
				t.Errorf("%s: description is empty", fieldPrefix)
			}
			if !validTypes[field.Type] {
				t.Errorf("%s: invalid type %q", fieldPrefix, field.Type)
			}
			if field.Min != nil && field.Max != nil && *field.Min > *field.Max {
				t.Errorf("%s: min (%v) > max (%v)", fieldPrefix, *field.Min, *field.Max)
			}
			if field.MinLength != nil && field.MaxLength != nil && *field.MinLength > *field.MaxLength {
				t.Errorf("%s: minLength (%d) > maxLength (%d)", fieldPrefix, *field.MinLength, *field.MaxLength)
			}
		}
	}
}

// validateWorkflowConfig checks structural integrity of a workflow config.
func validateWorkflowConfig(t *testing.T, filename string, cfg *workflow.Config) {
	t.Helper()
	if cfg.Name == "" {
		t.Errorf("%s: workflow config name is empty", filename)
	}
	if cfg.Description == "" {
		t.Errorf("%s: workflow config description is empty", filename)
	}
	if len(cfg.Workflows) < 1 {
		t.Errorf("%s: workflow config has no workflows", filename)
	}

	for wfName, wf := range cfg.Workflows {
		prefix := fmt.Sprintf("%s/workflows/%s", filename, wfName)

		if wf.Description == "" {
			t.Errorf("%s: description is empty", prefix)
		}
		if len(wf.Steps) < 1 {
			t.Errorf("%s: has no steps", prefix)
		}

		for i, input := range wf.Inputs {
			inputPrefix := fmt.Sprintf("%s/inputs[%d]", prefix, i)
			if input.Name == "" {
				t.Errorf("%s: name is empty", inputPrefix)
			}
			if input.Description == "" {
				t.Errorf("%s: description is empty", inputPrefix)
			}
		}

		for i, step := range wf.Steps {
			stepPrefix := fmt.Sprintf("%s/steps[%d]", prefix, i)
			if step.ID == "" {
				t.Errorf("%s: id is empty", stepPrefix)
			}
			if step.Description == "" {
				t.Errorf("%s: description is empty", stepPrefix)
			}
			if step.Command == "" {
				t.Errorf("%s: command is empty", stepPrefix)
			}
		}
	}
}

// validateWorkflowCommandRefs checks cross-file references between workflows and routes.
func validateWorkflowCommandRefs(t *testing.T, wfFiles map[string][]byte, routeConfigs map[string]*router.RouteConfig) {
	t.Helper()

	// Build a lookup map: routeConfigName -> RouteConfig
	for wfFile, wfData := range wfFiles {
		wfCfg, err := workflow.Parse(wfData)
		if err != nil {
			t.Errorf("%s: parse error: %v", wfFile, err)
			continue
		}

		for wfName, wf := range wfCfg.Workflows {
			for i, step := range wf.Steps {
				parts := strings.Fields(step.Command)
				stepPrefix := fmt.Sprintf("%s/%s/steps[%d](%s)", wfFile, wfName, i, step.ID)

				if len(parts) != 2 {
					t.Errorf("%s: command %q should have exactly 2 parts (routeName actionName), got %d",
						stepPrefix, step.Command, len(parts))
					continue
				}

				routeName := parts[0]
				actionName := parts[1]

				routeCfg, ok := routeConfigs[routeName]
				if !ok {
					t.Errorf("%s: command references route config %q which does not exist",
						stepPrefix, routeName)
					continue
				}

				if _, ok := routeCfg.Routes[actionName]; !ok {
					available := sortedKeys(routeCfg.Routes)
					t.Errorf("%s: command references action %q not found in route config %q (available: %v)",
						stepPrefix, actionName, routeName, available)
				}
			}
		}
	}
}

func sortedKeys(m map[string]router.Route) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestAllRoutes validates all route YAML files for structure and field semantics.
func TestAllRoutes(t *testing.T) {
	routeFiles := loadRouteFiles(t)
	if len(routeFiles) == 0 {
		t.Fatal("no route YAML files found in configFS routes/")
	}

	// Sort filenames for deterministic test order
	filenames := make([]string, 0, len(routeFiles))
	for f := range routeFiles {
		filenames = append(filenames, f)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		data := routeFiles[filename]
		t.Run(filename, func(t *testing.T) {
			cfg, err := router.Parse(data)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			validateRouteConfig(t, filename, cfg)
			validateRouteFields(t, filename, cfg)
		})
	}
}

// TestAllWorkflows validates all workflow YAML files for structure.
func TestAllWorkflows(t *testing.T) {
	wfFiles := loadWorkflowFiles(t)
	if len(wfFiles) == 0 {
		t.Fatal("no workflow YAML files found in configFS workflows/")
	}

	// Sort filenames for deterministic test order
	filenames := make([]string, 0, len(wfFiles))
	for f := range wfFiles {
		filenames = append(filenames, f)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		data := wfFiles[filename]
		t.Run(filename, func(t *testing.T) {
			cfg, err := workflow.Parse(data)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			validateWorkflowConfig(t, filename, cfg)
		})
	}
}

// TestWorkflowCommandReferences validates cross-file references between workflow steps and route configs.
func TestWorkflowCommandReferences(t *testing.T) {
	routeFiles := loadRouteFiles(t)
	wfFiles := loadWorkflowFiles(t)

	if len(routeFiles) == 0 {
		t.Fatal("no route YAML files found")
	}
	if len(wfFiles) == 0 {
		t.Fatal("no workflow YAML files found")
	}

	// Parse all route configs, indexed by config name
	routeConfigs := make(map[string]*router.RouteConfig)
	for filename, data := range routeFiles {
		cfg, err := router.Parse(data)
		if err != nil {
			t.Fatalf("parse route %s: %v", filename, err)
		}
		routeConfigs[cfg.Name] = cfg
	}

	// Validate each workflow step's command references
	for wfFile, wfData := range wfFiles {
		wfCfg, err := workflow.Parse(wfData)
		if err != nil {
			t.Fatalf("parse workflow %s: %v", wfFile, err)
		}

		for wfName, wf := range wfCfg.Workflows {
			for i, step := range wf.Steps {
				testName := fmt.Sprintf("%s/%s/step[%d]_%s", wfFile, wfName, i, step.ID)
				t.Run(testName, func(t *testing.T) {
					parts := strings.Fields(step.Command)
					if len(parts) != 2 {
						t.Errorf("command %q should have exactly 2 parts, got %d", step.Command, len(parts))
						return
					}

					routeName := parts[0]
					actionName := parts[1]

					routeCfg, ok := routeConfigs[routeName]
					if !ok {
						available := make([]string, 0, len(routeConfigs))
						for k := range routeConfigs {
							available = append(available, k)
						}
						sort.Strings(available)
						t.Errorf("route config %q not found (available: %v)", routeName, available)
						return
					}

					if _, ok := routeCfg.Routes[actionName]; !ok {
						available := sortedKeys(routeCfg.Routes)
						t.Errorf("action %q not found in route config %q (available: %v)",
							actionName, routeName, available)
					}
				})
			}
		}
	}
}
