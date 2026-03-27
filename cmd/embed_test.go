package cmd

import (
	"embed"
	"io/fs"
	"testing"

	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
	workflowcmd "github.com/childelins/ckjr-cli/cmd/workflow"
)

//go:embed all:ckjr-cli/routes all:ckjr-cli/workflows
var testEmbedFS embed.FS

func TestMain(m *testing.M) {
	subFS, err := fs.Sub(testEmbedFS, "ckjr-cli")
	if err != nil {
		panic(err)
	}
	yamlFS = configyaml.New(subFS)
	registerRouteCommands()
	rootCmd.AddCommand(workflowcmd.NewCommand(yamlFS))
	m.Run()
}
