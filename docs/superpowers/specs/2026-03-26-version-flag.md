# Version Flag ldflags 注入修复设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

修复 `--version` flag 的 ldflags 注入问题。当前 release.yml 使用 `-X main.Version=${VERSION}` 注入版本号，但 `Version` 和 `Environment` 变量定义在 `cmd` 包而非 `main` 包，导致 release 构建产物的 `--version` 始终输出 `dev`。

## 问题分析

### 当前代码结构

- `cmd/ckjr-cli/main.go` -- `package main`，仅调用 `cmd.Execute()`
- `cmd/root.go` -- `package cmd`，定义 `var Version = "dev"` 和 `var Environment = "production"`
- `main.go`（项目根目录）-- `package main`，遗留文件

### CI 构建命令

```yaml
go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Environment=production" \
  -o dist/ckjr-cli_${VERSION}_${GOOS}_${GOARCH}/${BINARY_NAME} ./cmd/ckjr-cli
```

### 问题根因

`Version` 定义在 `cmd` 包中，ldflags 写的 `main.Version` 找不到变量，注入静默失败。

## 修复方案

将 `Version` 和 `Environment` 变量定义移到 `main` 包，通过 setter 传递给 `cmd` 包。保持 ldflags 短路径 `-X main.Version` 不变。

### 修改文件

1. **`cmd/root.go`** -- `Version`/`Environment` 改为 private，新增 `SetVersion(version string)` 和 `SetEnvironment(env string)`
2. **`cmd/ckjr-cli/main.go`** -- 在 `main` 包定义 `Version`/`Environment`，`init()` 中调用 setter

### 修改后的代码结构

```go
// cmd/ckjr-cli/main.go
package main

import "github.com/childelins/ckjr-cli/cmd"

var (
    Version      = "dev"
    Environment  = "production"
)

func init() {
    cmd.SetVersion(Version)
    cmd.SetEnvironment(Environment)
}

func main() {
    cmd.Execute()
}
```

```go
// cmd/root.go
var (
    version      = "dev"
    environment  = "production"
)

func SetVersion(v string) { version = v }
func SetEnvironment(e string) { environment = e }
```

### 验证方法

```bash
# 带 ldflags 构建
go build -ldflags="-X main.Version=v9.9.9 -X main.Environment=production" ./cmd/ckjr-cli
./ckjr-cli --version
# 预期: ckjr-cli version v9.9.9

# 不带 ldflags 构建（默认值）
go build ./cmd/ckjr-cli
./ckjr-cli --version
# 预期: ckjr-cli version dev
```

## 测试策略（TDD）

### 单元测试

为 `cmd/root.go` 新增 `root_test.go`：

1. 验证 `SetVersion` 能正确更新版本号
2. 验证 `SetEnvironment` 能正确更新环境
3. 验证 `rootCmd.Version` 字段与 version 变量绑定
4. 验证默认值（未调用 setter 时）

## 实现注意事项

1. 根目录的 `main.go` 是遗留文件，不在本次修复范围
2. `rootCmd.Version` 在 `var rootCmd` 声明时初始化为 `version` 的值，需要在 setter 中同步更新 `rootCmd.Version`
