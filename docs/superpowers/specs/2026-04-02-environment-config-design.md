# 环境配置设计文档

> Created: 2026-04-02
> Status: Draft (v2 - 简化方案)

## 概述

根据已有的 `Environment` 变量（`main.go` 中定义，production 通过 LDFLAGS 注入）自动选择对应环境的 `base_url` 默认值。当用户配置文件中 `base_url` 为空时，自动使用该默认值。不修改 `config init` / `config set` / `config show` 的逻辑。

OSS 域名由 `imageSign` API 的 `host` 字段动态返回，不需要在配置中管理。

## 背景

当前 `~/.ckjr/config.json` 结构：

```json
{
  "base_url": "https://kpapi-cs.ckjr001.com/api",
  "api_key": "Bearer eyJ..."
}
```

问题：
- `config init` 要求用户手动输入 `base_url`，新用户不知道填什么
- 环境地址是固定的，应由编译时 Environment 变量自动决定

两套环境地址：

| Environment 值 | base_url |
|----------------|----------|
| development（默认） | `https://kpapi-cs.ckjr001.com/api` |
| production（LDFLAGS 注入） | `https://kpapi0.kw.ckjr.cn/api` |

## 设计

### 核心思路

1. `config.go` 新增 `envBaseURLs` map 和 `DefaultBaseURL()` 函数，根据 `main.Environment` 返回对应环境的 base_url
2. `Config` 新增 `ResolveBaseURL()` 方法：`base_url` 非空则用它，否则返回 `DefaultBaseURL()`
3. `cmd/root.go` 的 `createClient` 调用 `ResolveBaseURL()` 代替直接读 `cfg.BaseURL`
4. `cmd/config/config.go` 不做任何修改

### 变更范围

```
internal/
  config/
    config.go       # 修改: 新增 DefaultBaseURL 常量
                    #       新增 ResolveBaseURL() 方法
    config_test.go  # 修改: 新增 ResolveBaseURL 测试

cmd/
  root.go           # 修改: createClient 调用 ResolveBaseURL()
```

不涉及的模块：
- `cmd/config/config.go` -- 不修改，init/set/show 逻辑不变
- `internal/api/client.go` -- 不修改，仍接收 `baseURL` 字符串
- `internal/ossupload/` -- 不修改，OSS 域名由 API 动态返回

## 组件设计

### 1. Config 变更 (`internal/config/config.go`)

```go
// envBaseURLs 各环境对应的默认 base_url
var envBaseURLs = map[string]string{
    "development": "https://kpapi-cs.ckjr001.com/api",
    "production":  "https://kpapi0.kw.ckjr.cn/api",
}

// Environment 由 main 包通过 SetEnvironment 注入
var environment string

// SetEnvironment 由 main.go 调用，注入编译时的 Environment 值
func SetEnvironment(env string) {
    environment = env
}

// DefaultBaseURL 根据当前 Environment 返回对应的默认 base_url
func DefaultBaseURL() string {
    if u, ok := envBaseURLs[environment]; ok {
        return u
    }
    return envBaseURLs["development"] // 未知环境回退到开发环境
}

// ResolveBaseURL 返回最终使用的 base_url
//
// 优先级: config.json 中的 base_url > DefaultBaseURL()
func (c *Config) ResolveBaseURL() string {
    if c.BaseURL != "" {
        return c.BaseURL
    }
    return DefaultBaseURL()
}
```

说明：
- 不引入 environment 配置字段，不修改 config.json 格式
- `DefaultBaseURL()` 根据 `main.Environment`（通过 `SetEnvironment` 注入）动态返回
- 默认 development → 测试环境地址；production（LDFLAGS 注入）→ 生产地址
- `ResolveBaseURL()` 返回 `string`，不返回 error（永远有默认值兜底）
- Config 结构体不变，config.json 格式不变

### 2. createClient 变更 (`cmd/root.go`)

```go
func createClient() (*api.Client, error) {
    cfg, err := internalconfig.Load()
    if err != nil {
        return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr-cli config init")
    }
    return api.NewClient(cfg.ResolveBaseURL(), cfg.APIKey), nil
}
```

变更：
1. `cfg.BaseURL` -> `cfg.ResolveBaseURL()`
2. `main()` 中调用 `config.SetEnvironment(Environment)` 注入编译时环境值

### 3. config init / config set / config show

不做任何修改。现有行为：

- `config init`：提示输入 `base_url` 和 `api_key`。用户可以直接回车跳过 `base_url`（值为空字符串），此时 `ResolveBaseURL()` 根据 Environment 返回对应默认地址。
- `config set base_url <url>`：手动设置自定义地址，覆盖默认值。
- `config show`：显示 config.json 的原始值（`base_url` 可能为空）。

## 数据流

```
config init / config set
        |
        v
~/.ckjr/config.json
  { "base_url": "", "api_key": "..." }    // base_url 可能为空
        |
        v
config.Load() -> Config
        |
        v
Config.ResolveBaseURL()
  |-- base_url 非空 -> 使用 base_url
  |-- base_url 为空 -> 使用 DefaultBaseURL() (根据 Environment 决定)
        |
        v
api.NewClient(resolvedURL, apiKey)
```

### config.json 格式

格式不变。以下场景均可正常工作：

只设置了 api_key（推荐的最简配置，base_url 由 Environment 决定）：

```json
{
  "api_key": "eyJ0eXAiOiJKV1Qi..."
}
```

手动覆盖了 base_url（优先级最高，覆盖 Environment 默认值）：

```json
{
  "base_url": "https://kpapi-cs.ckjr001.com/api",
  "api_key": "eyJ0eXAiOiJKV1Qi..."
}
```

旧配置文件（已有 base_url，向后兼容）：

```json
{
  "base_url": "https://kpapi0.kw.ckjr.cn/api",
  "api_key": "eyJ0eXAiOiJKV1Qi..."
}
```

## 错误处理

| 场景 | 处理 |
|------|------|
| 配置文件不存在 | `Load` 返回 `ErrConfigNotFound`，提示执行 `config init` |
| base_url 为空 | `ResolveBaseURL` 返回 `DefaultBaseURL()`（根据 Environment），不报错 |
| base_url 非空 | `ResolveBaseURL` 返回该值 |

不再需要 `ErrNoBaseURL` 错误，因为永远有默认值。

## 测试策略

### 单元测试 (`internal/config/config_test.go`)

1. **TestResolveBaseURL_WithBaseURL** -- base_url="custom" -> 返回 "custom"
2. **TestResolveBaseURL_EmptyBaseURL_Development** -- environment=development, base_url="" -> 返回测试环境地址
3. **TestResolveBaseURL_EmptyBaseURL_Production** -- environment=production, base_url="" -> 返回生产环境地址
4. **TestDefaultBaseURL_UnknownEnv** -- environment="unknown" -> 回退到开发环境地址

### 无需 cmd 测试

config init / config set / config show 逻辑不变，无需新增测试。

## 实现注意事项

### 关键点

1. **Config 结构体不变** -- 不新增字段，config.json 格式完全兼容。

2. **向后兼容** -- 旧配置文件已有 `base_url`，`ResolveBaseURL()` 优先使用它，行为与原来一致。

3. **config init 跳过 base_url** -- 用户在 `config init` 时直接回车不输入 `base_url`，保存的空字符串由 `ResolveBaseURL()` 根据 Environment 处理为对应默认值。不需要修改 init 逻辑。

4. **手动覆盖 base_url** -- 用户通过 `config set base_url <url>` 手动设置自定义地址。

5. **恢复默认** -- 用户通过 `config set base_url ""` 清空，恢复使用 Environment 对应的默认地址。

6. **Environment 注入** -- `main.go` 在 `init()` 或 `main()` 中调用 `config.SetEnvironment(Environment)`，将编译时注入的值传递给 config 包。

### 实现顺序

1. `internal/config/config.go` -- 新增 `envBaseURLs` map、`SetEnvironment()` 函数、`DefaultBaseURL()` 函数、`ResolveBaseURL()` 方法
2. `internal/config/config_test.go` -- 新增测试
3. `cmd/ckjr-cli/main.go` -- 调用 `config.SetEnvironment(Environment)`
4. `cmd/root.go` -- `createClient` 使用 `cfg.ResolveBaseURL()`

### 不在本期实现

- 不引入 environment 配置字段到 config.json
- 不修改 config init 交互流程
- 不修改 config set 支持的键
- 不修改 config show 输出
- 不提供环境选择菜单
