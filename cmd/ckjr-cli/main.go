package main

import "github.com/childelins/ckjr-cli/cmd"

var (
	// Version 版本号，通过 -ldflags "-X main.Version=x.x.x" 注入
	Version = "dev"
	// Environment 环境模式，通过 -ldflags "-X main.Environment=production" 注入
	Environment = "production"
)

func init() {
	cmd.SetVersion(Version)
	cmd.SetEnvironment(Environment)
}

func main() {
	cmd.Execute()
}
