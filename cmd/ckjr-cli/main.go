package main

import (
	"github.com/childelins/ckjr-cli/cmd"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

var (
	// Version 版本号，通过 -ldflags "-X main.Version=x.x.x" 注入
	Version = "dev"
	// Environment 环境模式，通过 -ldflags "-X main.Environment=production" 注入
	Environment = "development"
)

func init() {
	cmd.SetVersion(Version)
	cmd.SetEnvironment(Environment)
	cmd.SetYAMLFS(configyaml.New(configFS))
}

func main() {
	cmd.Execute()
}
