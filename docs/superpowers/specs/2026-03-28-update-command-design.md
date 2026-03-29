# update 命令设计文档

> Created: 2026-03-28
> Status: Draft

## 概述

为 ckjr-cli 添加 `ckjr-cli update` 命令，自动检测 GitHub Release 上的最新版本，下载并替换当前二进制文件，实现 CLI 工具的自更新能力。

## 背景

项目已有完善的 GitHub Release 发布流程（Makefile + install.sh），版本通过 ldflags 注入 `main.Version`。公共分发仓库 `childelins/ckjr-cli` 的 Release 页面提供多平台预编译二进制。当前用户更新需要重新运行 install.sh，体验不够顺畅。update 命令将这一流程内置到 CLI 本身。

## 需求决策

| 决策项 | 结论 |
|--------|------|
| 命令行为 | 一步到位：`ckjr-cli update` 直接检查并自动更新 |
| dev 版本处理 | 报错拒绝，提示需要使用正式版本 |
| 检查触发 | 仅手动执行 update 命令时检查，不做后台静默检查 |

## 架构

```
cmd/update/update.go          # Cobra 命令定义，NewCommand() 入口
internal/updater/updater.go   # 核心更新逻辑（版本检查、下载、替换）
```

遵循项目现有命令模式：`cmd/xxx` 包提供 `NewCommand()` 函数，在 `cmd/root.go` 的 `init()` 或 `Execute()` 中注册。

## 组件

### 1. update 命令（cmd/update/update.go）

提供 `NewCommand() *cobra.Command`，注册为 `ckjr-cli update`。

```
ckjr-cli update
```

命令流程：
1. 检查当前版本是否为 dev，是则报错退出
2. 调用 updater 查询 GitHub Release 最新版本
3. 比较当前版本与最新版本
4. 如已是最新，提示并退出
5. 下载新版本到临时文件
6. 替换当前二进制
7. 提示更新成功

### 2. 更新器（internal/updater/updater.go）

核心逻辑封装，职责：

- **版本查询**：调用 GitHub API `GET /repos/childelins/ckjr-cli/releases/latest`，解析 `tag_name` 获取最新版本号
- **版本比较**：semver 比较（去掉 `v` 前缀后按 major.minor.patch 数值比较）
- **平台检测**：使用 `runtime.GOOS` 和 `runtime.GOARCH` 确定目标平台
- **下载**：从 Release 的 `assets` 中匹配正确的产物 URL，下载到临时目录
- **替换**：用下载的新二进制替换当前运行的二进制

### 接口设计

```go
// updater 包实现 CLI 自更新逻辑
package updater

// CheckLatestVersion 查询 GitHub Release 最新版本号
// 返回最新版本 tag（如 "v0.2.0"）和下载 URL
func CheckLatestVersion() (version string, downloadURL string, err error)

// CompareVersions 比较两个 semver 版本
// 返回 >0 表示 current 比 latest 新，<0 表示有更新可用，0 表示相同
func CompareVersions(current, latest string) (int, error)

// DownloadAndReplace 下载新版本二进制并替换当前文件
func DownloadAndReplace(downloadURL string) error
```

## 数据流

### 更新流程

```
ckjr-cli update
    │
    ├─► 1. 检查当前版本
    │     ├─ version == "dev" → 报错："当前为开发版本，请使用正式版本安装"
    │     └─ version != "dev" → 继续
    │
    ├─► 2. 查询最新版本
    │     ├─ GET https://api.github.com/repos/childelins/ckjr-cli/releases/latest
    │     ├─ 解析 JSON → tag_name + assets[]
    │     ├─ 网络错误 → 报错："无法检查更新: <error>"
    │     └─ 成功 → 得到 latestVersion + downloadURL
    │
    ├─► 3. 版本比较
    │     ├─ current >= latest → 输出 "已是最新版本 (v0.1.0)"，退出
    │     └─ current < latest → 继续
    │
    ├─► 4. 输出更新信息
    │     └─ "发现新版本: v0.1.0 → v0.2.0，正在更新..."
    │
    ├─► 5. 下载新版本
    │     ├─ 创建临时目录
    │     ├─ 下载产物压缩包到临时目录
    │     ├─ 解压（tar.gz 或 zip）
    │     └─ 网络错误 → 报错，清理临时文件
    │
    ├─► 6. 替换二进制
    │     ├─ 获取当前可执行文件路径 (os.Executable)
    │     ├─ 重命名旧文件为 .bak
    │     ├─ 复制新文件到原路径
    │     ├─ 设置可执行权限
    │     ├─ 删除 .bak
    │     └─ 替换失败 → 回滚（恢复 .bak），报错
    │
    └─► 7. 完成
          └─ 输出 "更新成功: v0.1.0 → v0.2.0"
```

### 产物匹配逻辑

从 Release JSON 的 `assets` 数组中匹配：

```go
// 目标产物名模式：ckjr-cli_{version}_{os}_{arch}.tar.gz (或 .zip)
pattern := fmt.Sprintf("ckjr-cli_%s_%s_%s", version, runtime.GOOS, runtime.GOARCH)
// windows 用 .zip，其他用 .tar.gz
```

## 技术决策

### 自更新替换策略

采用 **重命名-替换** 策略：

1. `os.Executable()` 获取当前二进制的绝对路径
2. 将旧二进制重命名为 `<name>.bak`
3. 将新二进制复制到原路径
4. 设置可执行权限（`chmod +x`）
5. 删除 `.bak` 文件
6. 如果步骤 3 失败，从 `.bak` 回滚

选择此方案的原因：
- 跨平台兼容（Linux/macOS/Windows 均支持）
- 不依赖外部命令
- 原子性回滚保证安全

### 版本比较：semver 数值比较

去掉 `v` 前缀，按 `major.minor.patch` 拆分后逐段数值比较：

```go
func CompareVersions(current, latest string) (int, error) {
    // 去掉 v 前缀
    current = strings.TrimPrefix(current, "v")
    latest = strings.TrimPrefix(latest, "v")

    // 按 . 拆分
    curParts := strings.Split(current, ".")
    latParts := strings.Split(latest, ".")

    // 逐段数值比较
    for i := 0; i < max(len(curParts), len(latParts)); i++ {
        cur, _ := strconv.Atoi(getPart(curParts, i))
        lat, _ := strconv.Atoi(getPart(latParts, i))
        if cur != lat {
            return cur - lat, nil
        }
    }
    return 0, nil
}
```

不引入第三方 semver 库，保持项目零外部依赖的风格（项目当前仅依赖 cobra + yaml.v3）。

### GITHUB_TOKEN 支持

不主动支持。原因：
- ckjr-cli 是公开仓库，GitHub API 匿名访问有 60 次/小时的限额
- update 命令是手动触发，不会频繁调用，60 次/小时足够
- 如果未来需要，可通过环境变量 `GITHUB_TOKEN` 扩展，但当前不加

### 下载实现

使用 Go 标准库 `net/http`，不依赖 curl/wget 等外部命令。原因：
- Go 二进制本身已内置 HTTP 客户端
- 无需假设用户系统上安装了 curl/wget
- 更好的错误处理和跨平台一致性

下载超时设置：30 秒连接超时，总超时由响应 Content-Length 动态计算（最大 5 分钟）。

## 错误处理

| 错误场景 | 处理策略 |
|---------|---------|
| 当前版本为 dev | 立即报错退出：`当前为开发版本 (dev)，请使用 install.sh 安装正式版本` |
| 网络不可达 | 报错：`无法检查更新: <具体错误>` |
| GitHub API 返回非 200 | 报错：`检查更新失败: GitHub API 返回 <status>` |
| JSON 解析失败 | 报错：`解析版本信息失败: <错误>` |
| 当前平台无匹配产物 | 报错：`未找到 <os>/<arch> 平台的更新包` |
| 下载失败 | 报错：`下载更新失败: <错误>`，清理临时文件 |
| 解压失败 | 报错：`解压更新包失败: <错误>` |
| 替换二进制失败（权限不足等） | 回滚旧版本，报错：`替换二进制失败: <错误>，已回滚` |
| 回滚也失败 | 报错：`更新失败且回滚失败，请手动安装: <原路径>.bak 备份已保留` |

## 测试策略

### 单元测试（TDD）

**internal/updater/updater_test.go**：

- `TestCompareVersions`：覆盖各种版本比较场景
  - 相等版本："0.1.0" vs "0.1.0" → 0
  - 小版本更新："0.1.0" vs "0.2.0" → <0
  - 大版本更新："0.9.0" vs "1.0.0" → <0
  - 带 v 前缀："v0.1.0" vs "v0.2.0" → <0
  - 不同段数："0.1.0" vs "0.1" → 0
  - 无效输入的 error 处理

- `TestParseAssetURL`：从 Release JSON 中解析正确的下载 URL
  - 正常匹配 linux/amd64
  - 正常匹配 darwin/arm64
  - 正常匹配 windows/amd64（.zip）
  - 无匹配产物时返回错误

- `TestCheckLatestVersion`：使用 httptest 模拟 GitHub API
  - 正常响应返回版本号和 URL
  - 非 200 响应返回错误
  - 无效 JSON 返回错误

- `TestDownloadAndReplace`：使用临时文件系统
  - 正常替换流程
  - 替换失败时回滚
  - 权限保留

**cmd/update/update_test.go**：

- 测试 dev 版本报错
- 测试已是最新版本的输出
- 测试更新成功的输出

### 集成测试

- 构建测试版本，手动执行 `ckjr-cli update`，验证与 GitHub Release 交互
- 测试从旧版本更新到新版本的完整流程

## 实现注意事项

### 文件结构变更

```
新增文件：
  cmd/update/update.go              # update 命令
  cmd/update/update_test.go         # update 命令测试
  internal/updater/updater.go       # 更新核心逻辑
  internal/updater/updater_test.go  # 更新逻辑测试

修改文件：
  cmd/root.go                       # 在 init() 中注册 update 命令
```

### root.go 注册方式

```go
func init() {
    // ... 已有代码 ...
    rootCmd.AddCommand(updatecmd.NewCommand())
}
```

update 命令不依赖 yamlFS 或 config，可在 `init()` 中直接注册。

### 版本获取

update 命令需要读取当前版本号。版本号已在 `cmd` 包中定义为包级变量 `version`，update 命令作为 `cmd` 包的子包无法直接访问。

解决方案：通过 `NewCommand()` 参数传入，或在 `cmd/update` 包中定义 `SetVersion` 函数由 root.go 调用。推荐后者，与项目现有的 `SetVersion`/`SetYAMLFS` 模式一致。

```go
// cmd/update/update.go
var currentVersion = "dev"

func SetVersion(v string) {
    currentVersion = v
}

// cmd/root.go init()
func init() {
    // ...
    updatecmd.SetVersion(version)
    rootCmd.AddCommand(updatecmd.NewCommand())
}
```

注意：需要在 `SetVersion()` 之后调用 `updatecmd.SetVersion(version)`，确保版本号已注入。由于 `init()` 中 `SetVersion` 在 main.go 的 `init()` 中已调用（main.go init 先于 cmd/root.go init 执行），时序是正确的。

### Windows 兼容

- Windows 上可执行文件带 `.exe` 后缀
- Windows 产物为 `.zip` 格式
- 替换二进制时，Windows 上正在运行的 exe 文件无法直接覆盖，需要使用 `MoveFileEx` 系统调用标记为重启后替换，或采用重命名策略（`ren` 旧文件后复制新文件）

为简化首版实现，Windows 上如果遇到文件占用，提示用户关闭其他终端窗口后重试。

### 进度提示

下载过程中使用简单的文本进度：

```
正在检查更新...
发现新版本: v0.1.0 → v0.2.0
正在下载 ckjr-cli_v0.2.0_linux_amd64.tar.gz...
正在安装...
更新成功！v0.1.0 → v0.2.0
```

不使用进度条库，保持零外部依赖。

### 临时文件清理

使用 `os.CreateTemp` 创建临时目录，下载和解压在此目录中进行。无论成功或失败，最终都清理临时文件。使用 `defer` 确保清理。

## 实现步骤

按 TDD 流程：

1. **版本比较**：实现 `CompareVersions` + 测试
2. **产物 URL 解析**：实现 `ParseAssetURL` + 测试（使用 httptest 模拟 JSON）
3. **版本查询**：实现 `CheckLatestVersion` + 测试（使用 httptest 模拟 API）
4. **下载替换**：实现 `DownloadAndReplace` + 测试（使用临时文件）
5. **命令集成**：实现 `cmd/update/update.go` + 测试
6. **注册命令**：修改 `cmd/root.go` 注册 update 命令
