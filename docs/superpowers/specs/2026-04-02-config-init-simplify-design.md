# config init 简化设计文档

> Created: 2026-04-02
> Status: Draft

## 概述

简化 `config init` 交互流程，移除 base_url 输入步骤，仅保留 api_key 配置。base_url 由已有的 `ResolveBaseURL()` 机制根据编译时环境变量自动解析。

## 变更范围

仅涉及 `cmd/config/config.go` 的 `runConfigInit` 函数。

## 当前行为

```
$ ckjr-cli config init
请输入 API 地址 (base_url): <用户输入>
请按以下步骤获取 API Key:
1. 访问公司 SaaS 平台并登录
2. 进入个人设置 -> API 密钥
3. 复制 API Key
请粘贴 API Key: <用户输入>
配置已保存到: ~/.ckjr/config.json
```

## 目标行为

```
$ ckjr-cli config init
请按以下步骤获取 API Key:
1. 访问公司 SaaS 平台并登录
2. 进入个人设置 -> API 密钥
3. 复制 API Key
请粘贴 API Key: <用户输入>
配置已保存到: ~/.ckjr/config.json
```

## 实现方案

1. 从 `runConfigInit` 中删除 base_url 的 prompt 和读取逻辑（第 47-49 行）
2. 保存时 `Config.BaseURL` 留空（`""`）
3. 运行时通过 `ResolveBaseURL()` 自动回退到 `DefaultBaseURL()`

## 已有机制依赖

- `ResolveBaseURL()`: 当 `BaseURL == ""` 时返回 `DefaultBaseURL()`
- `DefaultBaseURL()`: 根据编译时 `environment` 变量返回对应环境 URL
- `config show` 已使用 `ResolveBaseURL()`，无需改动
- `config set base_url <value>` 保留，用户仍可手动覆盖

## 测试策略

- 验证 init 后保存的配置 base_url 为空
- 验证 init 后 `ResolveBaseURL()` 返回环境默认值
- 现有 `TestConfigShowEmptyBaseURL` 已覆盖空 base_url 场景

## 实现注意事项

- 不影响 `config set` 和 `config show` 子命令
- 用户如需自定义 base_url，可使用 `config set base_url <value>`
