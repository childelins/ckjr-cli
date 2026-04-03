# Config Show Base URL 修复设计

## 问题

`ckjr-cli config show` 在 `base_url` 未设置（空字符串）时显示空值，
而非当前环境的默认 base_url。

## 根因

`cmd/config/config.go:runConfigShow` 直接读取 `cfg.BaseURL`，
未使用已有的 `ResolveBaseURL()` 方法。

## 方案

将 `cfg.BaseURL` 替换为 `cfg.ResolveBaseURL()`，
该方法优先返回配置值，空时回退到 `DefaultBaseURL()`。

## 影响范围

- `cmd/config/config.go` — runConfigShow 函数
- `cmd/config/config_test.go` — 新增测试用例
