> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

# Config Show Base URL 修复计划

## Task 1: 添加失败测试

在 `cmd/config/config_test.go` 中新增测试：
验证 base_url 为空时 config show 返回环境默认值。

**文件**: `cmd/config/config_test.go`

## Task 2: 修复 runConfigShow

在 `cmd/config/config.go:102` 将 `cfg.BaseURL` 替换为 `cfg.ResolveBaseURL()`。

**文件**: `cmd/config/config.go`

## Task 3: 运行测试验证

运行全部 config 相关测试，确认修复正确且无回归。
