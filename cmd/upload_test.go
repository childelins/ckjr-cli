package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAssetCmdHasUploadImageSubcommand(t *testing.T) {
	// 在 TestMain 中 registerRouteCommands 已执行，asset 命令已注册
	var assetCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "asset" {
			assetCmd = cmd
			break
		}
	}
	if assetCmd == nil {
		t.Fatal("asset command not found in rootCmd")
	}

	// 检查 upload-image 子命令
	found := false
	for _, sub := range assetCmd.Commands() {
		if sub.Name() == "upload-image" {
			found = true
			break
		}
	}
	if !found {
		t.Error("asset command should have upload-image subcommand")
	}
}

func TestUploadImageCmdHasCorrectUse(t *testing.T) {
	var assetCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "asset" {
			assetCmd = cmd
			break
		}
	}
	if assetCmd == nil {
		t.Fatal("asset command not found in rootCmd")
	}

	var uploadCmd *cobra.Command
	for _, sub := range assetCmd.Commands() {
		if sub.Name() == "upload-image" {
			uploadCmd = sub
			break
		}
	}
	if uploadCmd == nil {
		t.Fatal("upload-image subcommand not found")
	}

	if uploadCmd.Short == "" {
		t.Error("upload-image command should have a short description")
	}
}
