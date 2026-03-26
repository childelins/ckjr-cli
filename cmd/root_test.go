package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmdExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	if rootCmd.Use != "ckjr-cli" {
		t.Errorf("rootCmd.Use = %s, want ckjr-cli", rootCmd.Use)
	}
}

func TestRootCmdHasPrettyFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("pretty")
	if flag == nil {
		t.Fatal("--pretty flag should exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("--pretty default = %s, want false", flag.DefValue)
	}
}

func TestRootCmdHasConfigSubcommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("rootCmd should have config subcommand")
	}
}

func TestRootCmdHasAgentSubcommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("rootCmd should have agent subcommand (auto-registered from routes)")
	}
}

func TestAgentSubcommands(t *testing.T) {
	var agentCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "agent" {
			agentCmd = cmd
			break
		}
	}
	if agentCmd == nil {
		t.Fatal("agent subcommand not found")
	}

	expectedSubs := map[string]bool{
		"list":   false,
		"get":    false,
		"create": false,
		"update": false,
		"delete": false,
	}

	for _, sub := range agentCmd.Commands() {
		// sub.Use is like "list [json]", extract first word
		name := sub.Name()
		if _, ok := expectedSubs[name]; ok {
			expectedSubs[name] = true
		}
	}

	for name, found := range expectedSubs {
		if !found {
			t.Errorf("agent subcommand %s not found", name)
		}
	}
}

func TestAgentListHasTemplateFlag(t *testing.T) {
	var agentCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "agent" {
			agentCmd = cmd
			break
		}
	}
	if agentCmd == nil {
		t.Fatal("agent subcommand not found")
	}

	listCmd, _, _ := agentCmd.Find([]string{"list"})
	if listCmd == nil || listCmd.Name() != "list" {
		t.Fatal("agent list subcommand not found")
	}

	flag := listCmd.Flags().Lookup("template")
	if flag == nil {
		t.Error("agent list should have --template flag")
	}
}

func TestVersionIsSet(t *testing.T) {
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}
}

func TestSetVersion(t *testing.T) {
	SetVersion("v1.2.3")
	if rootCmd.Version != "v1.2.3" {
		t.Errorf("rootCmd.Version = %s, want v1.2.3", rootCmd.Version)
	}
	// 恢复默认值，避免影响其他测试
	SetVersion("dev")
}

func TestSetEnvironment(t *testing.T) {
	SetEnvironment("development")
	if environment != "development" {
		t.Errorf("environment = %s, want development", environment)
	}
	// 恢复默认值，避免影响其他测试
	SetEnvironment("production")
}

func TestVerboseFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("verbose")
	if f == nil {
		t.Fatal("--verbose flag 未注册")
	}
	if f.DefValue != "false" {
		t.Errorf("默认值 = %s, want false", f.DefValue)
	}
}

func TestDefaultVersion(t *testing.T) {
	// 未调用 SetVersion 时，version 应为默认值 "dev"
	if version != "dev" {
		t.Errorf("default version = %s, want dev", version)
	}
}

func TestDefaultEnvironment(t *testing.T) {
	if environment != "production" {
		t.Errorf("default environment = %s, want production", environment)
	}
}
