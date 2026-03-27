# cmd 目录结构重组设计文档

> Created: 2026-03-27
> Status: Draft

## 概述

将 `cmd/` 包下平铺的命令代码拆分为独立子包（`cmd/config/`、`cmd/route/`、`cmd/workflow/`，简洁命名不带 cmd 后缀），每个子包暴露 cobra.Command 注册函数，由 `cmd/root.go` 统一注册。同时将 `route.go` 中的辅助函数提取到 `internal/router` 包。

**追加修改**：将 YAML 配置路径精简，去掉 `config/` 中间层：
- `cmd/ckjr-cli/config/routes/*.yaml` -> `cmd/ckjr-cli/routes/*.yaml`
- `cmd/ckjr-cli/config/workflows/*.yaml` -> `cmd/ckjr-cli/workflows/*.yaml`

这影响 `embed.go` 的 embed 指令、`internal/config/yaml` 包的加载路径、以及相关测试。

## 背景

当前 `cmd/` 包下有 10 个文件平铺（5 源码 + 5 测试），通过包级变量（`yamlFS`、`rootCmd`、`version`、`environment`）和 `init()` 函数紧密耦合。随着命令增长，这种平铺结构难以维护。

### 当前结构

```
cmd/
  ckjr-cli/                    # main 包
    main.go                    # 入口，调用 cmd.Execute()
    embed.go                   # //go:embed all:config
    config/routes/*.yaml       # 路由配置（当前路径，将精简）
    config/workflows/*.yaml    # 工作流配置（当前路径，将精简）
  root.go                      # rootCmd 定义、yamlFS、createClient、registerRouteCommands
  config.go                    # config 命令（init/set/show）
  route.go                     # route import 命令 + 辅助函数（inferRouteName 等）
  workflow.go                  # workflow 命令（list/describe）
  root_test.go
  config_test.go
  route_test.go
  workflow_test.go
  embed_test.go                # TestMain 初始化 yamlFS + registerRouteCommands
```

### 耦合分析

| 共享资源 | 定义位置 | 使用位置 |
|----------|----------|----------|
| `yamlFS` | root.go:18 | workflow.go:82 (loadAllWorkflows), root.go:91 (registerRouteCommands), embed_test.go:19 |
| `rootCmd` | root.go:41 | root.go:67-73 (init 注册), root_test.go (多个测试), config_test.go:29 |
| `createClient()` | root.go:115 | root.go:109 (registerRouteCommands 传给 cmdgen.BuildCommand) |
| `registerRouteCommands()` | root.go:90 | root.go:50 (Execute), embed_test.go:20 (TestMain) |

## 目标结构

```
cmd/
  ckjr-cli/                    # main 包（不变）
    main.go
    embed.go                   # //go:embed all:routes all:workflows（精简后）
    routes/*.yaml              # 路由配置（精简后，去掉 config/ 中间层）
    workflows/*.yaml           # 工作流配置（精简后）
  root.go                      # rootCmd 定义、版本管理、initLogging
  root_test.go                 # 测试 root 命令本身
  config/                      # config 子包
    config.go                  # NewCommand() -> *cobra.Command
    config_test.go
  route/                       # route 子包
    route.go                   # NewCommand() -> *cobra.Command
    route_test.go
  workflow/                    # workflow 子包
    workflow.go                # NewCommand(yamlFS *configyaml.FS) -> *cobra.Command
    workflow_test.go
  root_test.go                 # 保留（测试 root 命令注册结果）
```

**注意**：embed_test.go 的 TestMain 初始化 yamlFS 并调用 registerRouteCommands，拆包后需要重新设计。

## 架构

### 核心思路

1. 每个子包提供一个 `NewCommand()` 工厂函数，返回 cobra.Command
2. 需要依赖的共享状态通过函数参数注入，而非包级变量
3. `createClient()` 函数提升为 `cmd` 包的导出函数（或提取到 internal），供子包调用
4. `registerRouteCommands()` 保留在 `cmd/root.go`，因为它直接操作 `rootCmd`

### 包依赖关系

```
cmd/root.go
  ├── imports cmd/config    -> rootCmd.AddCommand(config.NewCommand())
  ├── imports cmd/route     -> rootCmd.AddCommand(route.NewCommand())
  ├── imports cmd/workflow  -> rootCmd.AddCommand(workflow.NewCommand(yamlFS))
  └── registerRouteCommands()  -> 直接在 root.go 中实现（操作 rootCmd + createClient）

cmd/ckjr-cli/main.go
  └── imports cmd
      └── cmd.Execute()
```

## 组件设计

### 1. `cmd/config/config.go`

config 命令完全自包含，不依赖任何 cmd 包的共享状态。

```go
package config

import (
    "github.com/spf13/cobra"
    "github.com/childelins/ckjr-cli/internal/config"
    "github.com/childelins/ckjr-cli/internal/output"
)

func NewCommand() *cobra.Command {
    configCmd := &cobra.Command{
        Use:   "config",
        Short: "管理 CLI 配置",
    }

    configInitCmd := &cobra.Command{
        Use:   "init",
        Short: "交互式初始化配置",
        Run:   runConfigInit,
    }
    // ... setCmd, showCmd 同理

    configCmd.AddCommand(configInitCmd, configSetCmd, configShowCmd)
    return configCmd
}
```

**迁移内容**：`config.go` 全部代码，`config_test.go` 全部代码。

**测试调整**：
- `config_test.go` 中的 `setupTestConfig()` 直接迁移，无需修改（不依赖 rootCmd）
- 删除对 `rootCmd` 的引用（`config_test.go:29` 的 `cmd := rootCmd` 测试实际没有执行 rootCmd，而是直接测试 `config.Load()`，所以可以安全迁移）

### 2. `cmd/route/route.go`

route 命令不依赖 `yamlFS` 或 `rootCmd`。

```go
package route

import (
    "github.com/spf13/cobra"
    "github.com/childelins/ckjr-cli/internal/curlparse"
    "github.com/childelins/ckjr-cli/internal/router"
    "github.com/childelins/ckjr-cli/internal/yamlgen"
)

func NewCommand() *cobra.Command {
    routeCmd := &cobra.Command{
        Use:    "route",
        Short:  "路由配置管理",
        Hidden: true,
    }

    routeImportCmd := &cobra.Command{
        Use:   "import",
        Short: "从 curl 命令导入路由配置",
        RunE:  runRouteImport,
    }
    // ... flags 注册

    routeCmd.AddCommand(routeImportCmd)
    return routeCmd
}
```

**辅助函数处理**：`inferRouteName`、`inferNameFromPath`、`splitPath`、`split`、`toLower` 提取到 `internal/router` 包（用户选择 B）。这些函数与路由推导逻辑相关，放在 `internal/router` 中语义合理。

**迁移内容**：
- `route.go` 中的 `routeCmd`、`routeImportCmd`、`runImport` 移到 `cmd/route/route.go`
- `route.go` 中的辅助函数移到 `internal/router/infer.go`
- `route_test.go` 中的 `TestRouteCmd_IsHidden`、`TestRouteImport_*` 移到 `cmd/route/route_test.go`
- `route_test.go` 中的 `TestInferRouteName` 移到 `internal/router/infer_test.go`

### 3. `cmd/workflow/workflow.go`

workflow 命令依赖 `yamlFS`，通过参数注入。

```go
package workflow

import (
    "github.com/spf13/cobra"
    configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
    "github.com/childelins/ckjr-cli/internal/workflow"
)

func NewCommand(yamlFS *configyaml.FS) *cobra.Command {
    workflowCmd := &cobra.Command{
        Use:   "workflow",
        Short: "工作流管理",
    }

    workflowListCmd := &cobra.Command{
        Use:   "list",
        Short: "列出所有可用的工作流",
        RunE: func(cmd *cobra.Command, args []string) error {
            // ... 使用 yamlFS 加载
        },
    }

    // ... describeCmd 同理

    workflowCmd.AddCommand(workflowListCmd, workflowDescribeCmd)
    return workflowCmd
}
```

**迁移内容**：`workflow.go` 全部代码，`workflow_test.go` 全部代码。

**测试调整**：`workflow_test.go` 直接访问 `rootCmd` 执行命令。拆包后测试需要改为：
- 构造一个临时 rootCmd，添加 workflow 子命令
- 或者测试 `NewCommand()` 返回的命令对象

推荐方案：使用 `NewCommand(yamlFS)` 构造命令，然后用 `cmd.SetArgs()` 和 `cmd.Execute()` 测试。需要初始化一个 `yamlFS` 实例（可以复用 embed_test.go 的方式）。

### 4. `cmd/root.go`（修改）

保留 root 命令核心逻辑，移除子命令定义，改为 import 子包注册。

```go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"

    "github.com/childelins/ckjr-cli/internal/api"
    "github.com/childelins/ckjr-cli/internal/cmdgen"
    "github.com/childelins/ckjr-cli/internal/config"
    "github.com/childelins/ckjr-cli/internal/logging"
    "github.com/childelins/ckjr-cli/internal/router"
    configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"

    "github.com/childelins/ckjr-cli/cmd/config"
    "github.com/childelins/ckjr-cli/cmd/route"
    "github.com/childelins/ckjr-cli/cmd/workflow"
)

var yamlFS *configyaml.FS

func SetYAMLFS(fs *configyaml.FS) {
    yamlFS = fs
}

var (
    version      = "dev"
    environment  = "production"
)

func SetVersion(v string) {
    version = v
    rootCmd.Version = v
}

func SetEnvironment(e string) {
    environment = e
}

var rootCmd = &cobra.Command{
    Use:               "ckjr-cli",
    Short:             "创客匠人 CLI - 知识付费 SaaS 系统的命令行工具",
    Version:           version,
    CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func Execute() {
    registerRouteCommands()
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func init() {
    rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
    rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
    cobra.OnInitialize(initLogging)

    // 注册子命令
    rootCmd.AddCommand(config.NewCommand())
    rootCmd.AddCommand(route.NewCommand())
    rootCmd.AddCommand(workflow.NewCommand(yamlFS))
}

func initLogging() { /* 不变 */ }

func registerRouteCommands() { /* 不变 */ }

func createClient() (*api.Client, error) { /* 不变 */ }
```

### 5. `internal/router/infer.go`（新增）

从 `cmd/route.go` 提取辅助函数。

```go
package router

// InferRouteName 从 URL path 末段推导 route name
func InferRouteName(path string) string { ... }

// InferNameFromPath 从文件路径推导 name（resource 名称）
func InferNameFromPath(path string) string { ... }
```

函数名改为首字母大写以导出。`splitPath`、`split`、`toLower` 作为包内未导出函数留在 `infer.go` 中。

### 6. 测试基础设施调整

**embed_test.go**：当前 `TestMain` 初始化 `yamlFS` 并调用 `registerRouteCommands()`。拆包后：
- `registerRouteCommands()` 仍在 `cmd/root.go` 中，需要 `yamlFS` 已初始化
- `root_test.go` 测试 rootCmd 的子命令注册结果（如 agent 子命令是否存在），依赖 `registerRouteCommands()` 被调用

方案：embed_test.go 保留在 `cmd/` 包，逻辑不变：

```go
package cmd

func TestMain(m *testing.M) {
    // ... 初始化 yamlFS
    yamlFS = configyaml.New(subFS)
    // registerRouteCommands 在 Execute() 中调用，但 root_test.go 直接检查 rootCmd
    // 所以需要在 TestMain 中手动调用
    registerRouteCommands()
    m.Run()
}
```

**config_test.go**：迁移到 `cmd/config/`，移除对 `rootCmd` 的引用。

**route_test.go**：拆分到 `cmd/route/route_test.go` 和 `internal/router/infer_test.go`。

**workflow_test.go**：迁移到 `cmd/workflow/`，改为测试 `NewCommand(yamlFS)` 返回的命令。

**root_test.go**：保留在 `cmd/`。`TestRootCmdHasAgentSubcommand` 等测试依赖 `registerRouteCommands()`，通过 `TestMain` 初始化已满足。

## 数据流

```
编译时:
  cmd/ckjr-cli/routes/*.yaml, workflows/*.yaml
      |
      v  (//go:embed all:routes all:workflows)
  configFS (embed.FS)
      |
      v  (main.go -> cmd.SetYAMLFS)
  yamlFS (*configyaml.FS)  [cmd 包级别]
      |
      v  (cmd/init -> workflow.NewCommand(yamlFS))
  workflowCmd 持有 yamlFS 引用

运行时:
  rootCmd.Execute()
      |
      +-> registerRouteCommands() -> yamlFS.LoadRoutes() -> 动态注册命令
      +-> config.NewCommand()  -> init/set/show 子命令
      +-> route.NewCommand()   -> import 子命令
      +-> workflow.NewCommand(yamlFS) -> list/describe 子命令
```

## 变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| Create | `cmd/config/config.go` | 从 cmd/config.go 迁移 |
| Create | `cmd/config/config_test.go` | 从 cmd/config_test.go 迁移 |
| Create | `cmd/route/route.go` | 从 cmd/route.go 迁移命令定义 |
| Create | `cmd/route/route_test.go` | 从 cmd/route_test.go 迁移命令测试 |
| Create | `cmd/workflow/workflow.go` | 从 cmd/workflow.go 迁移 |
| Create | `cmd/workflow/workflow_test.go` | 从 cmd/workflow_test.go 迁移 |
| Create | `internal/router/infer.go` | 从 cmd/route.go 提取辅助函数 |
| Create | `internal/router/infer_test.go` | 从 cmd/route_test.go 迁移 |
| Modify | `cmd/root.go` | 移除子命令定义，改为 import 子包 |
| Delete | `cmd/config.go` | 已迁移 |
| Delete | `cmd/config_test.go` | 已迁移 |
| Delete | `cmd/route.go` | 已迁移 |
| Delete | `cmd/route_test.go` | 已迁移 |
| Delete | `cmd/workflow.go` | 已迁移 |
| Delete | `cmd/workflow_test.go` | 已迁移 |
| Move | `cmd/ckjr-cli/config/routes/*.yaml` -> `cmd/ckjr-cli/routes/*.yaml` | YAML 路径精简 |
| Move | `cmd/ckjr-cli/config/workflows/*.yaml` -> `cmd/ckjr-cli/workflows/*.yaml` | YAML 路径精简 |
| Modify | `cmd/ckjr-cli/embed.go` | `all:config` -> `all:routes all:workflows` |
| Modify | `internal/config/yaml/yaml.go` | `"config/routes"` -> `"routes"`，`"config/workflows"` -> `"workflows"` |
| Modify | `internal/config/yaml/yaml_test.go` | MapFS key 路径同步更新 |
| Delete | `cmd/ckjr-cli/config/` | 精简后整个 config/ 目录删除 |
| Modify | `internal/workflow/workflow_test.go` | 路径从 `../../cmd/ckjr-cli/config/workflows/` -> `../../cmd/ckjr-cli/workflows/` |
| Modify | `wiki/*.md` | 所有 `cmd/ckjr-cli/config/routes/` -> `cmd/ckjr-cli/routes/`，同理 workflows |

## 错误处理

- `yamlFS` 为 nil 时 `workflow.NewCommand()` 创建的命令在执行时报错（"YAML 文件系统未初始化"），与当前行为一致
- 辅助函数提取到 `internal/router` 后保持相同的错误处理逻辑

## 测试策略

### TDD 步骤

1. **先在 `internal/router/` 创建辅助函数的测试**（`infer_test.go`），从 `route_test.go` 中的 `TestInferRouteName` 迁移
2. **实现 `internal/router/infer.go`**，通过测试
3. **创建 `cmd/route/route_test.go`**，迁移 `TestRouteCmd_IsHidden`、`TestRouteImport_*`
4. **实现 `cmd/route/route.go`**，通过测试
5. **创建 `cmd/config/config_test.go`**，迁移 config 测试
6. **实现 `cmd/config/config.go`**，通过测试
7. **创建 `cmd/workflow/workflow_test.go`**，迁移 workflow 测试（需要 mock yamlFS）
8. **实现 `cmd/workflow/workflow.go`**，通过测试
9. **修改 `cmd/root.go`**，删除子命令定义，import 子包注册
10. **运行 `go test ./...`** 确认所有测试通过
11. **删除旧文件**（config.go、route.go、workflow.go 及对应测试）

### 关键测试点

- `embed_test.go` 的 `TestMain` 必须在 `cmd/root_test.go` 测试前初始化 `yamlFS` 和调用 `registerRouteCommands()`
- `workflow_test.go` 迁移后需要构造 `yamlFS` 实例，可以使用 `testing/fstest.MapFS`
- `config_test.go` 中的测试不依赖 `rootCmd`，可以直接迁移

## 实现注意事项

1. **循环依赖**：`cmd/root.go` import `cmd/config`、`cmd/route`、`cmd/workflow`。子包不能反向 import `cmd` 包。如果子包需要 `createClient()`，需要将其提取到 `internal/` 包（但当前只有 `registerRouteCommands` 需要，而它在 root.go 中，所以无此问题）。

2. **`yamlFS` 传递**：`workflow.NewCommand(yamlFS)` 在 `cmd/init()` 中调用。此时 `yamlFS` 可能还是 nil（由 main 包通过 `SetYAMLFS` 设置）。需要确认 Go init 执行顺序：main 包的 init 先于 cmd 包的 init 执行（因为 main import cmd，被 import 的包的 init 先执行）。等等，Go 规范是被 import 的包的 init 先执行。所以执行顺序是：`cmd` 包 init -> `config/route/workflow` 包 init（被 cmd import） -> `main` 包 init。这意味着 `cmd/init()` 中调用 `workflow.NewCommand(yamlFS)` 时，`main` 的 `SetYAMLFS` 还没执行。

   **解决方案**：将 `workflowCmd` 的创建延迟到 `Execute()` 中，而非 `init()` 中：
   ```go
   func Execute() {
       registerRouteCommands()
       rootCmd.AddCommand(workflow.NewCommand(yamlFS))  // 延迟到此时 yamlFS 已设置
       rootCmd.Execute()
   }
   ```
   或者更好的方案：所有子命令注册都移到 `Execute()` 中（但 config 和 route 不需要 yamlFS，在 init 中注册也没问题）。

   **推荐方案**：config 和 route 在 init 中注册，workflow 在 Execute 中注册（因为它需要 yamlFS）。

3. **辅助函数导出**：`inferRouteName` -> `router.InferRouteName`，`inferNameFromPath` -> `router.InferNameFromPath`。`splitPath`、`split`、`toLower` 保持包内私有。

4. **测试文件包名**：`cmd/` 包的测试文件使用 `package cmd`（非 `package cmd_test`），因为 `TestMain` 需要设置包级变量 `yamlFS` 并调用包内函数 `registerRouteCommands()`。

## YAML 路径精简

### 背景

当前 YAML 配置文件位于 `cmd/ckjr-cli/config/routes/` 和 `cmd/ckjr-cli/config/workflows/`。这个 `config/` 中间层是 Q1 迁移时引入的，现在可以精简掉。原因是这些 YAML 文件本身就是路由和工作流的定义，直接用 `routes/` 和 `workflows/` 命名更直观。

### 影响范围

#### 1. `cmd/ckjr-cli/embed.go`

当前：
```go
//go:embed all:config
var configFS embed.FS
```

精简后：
```go
//go:embed all:routes all:workflows
var configFS embed.FS
```

变量名 `configFS` 保持不变（语义仍合理，它是配置文件的 FS），如需改名可后续重构。

#### 2. `internal/config/yaml/yaml.go`

当前：
```go
func (f *FS) LoadRoutes() (map[string][]byte, error) {
    return f.loadDir("config/routes")
}

func (f *FS) LoadWorkflows() (map[string][]byte, error) {
    return f.loadDir("config/workflows")
}
```

精简后：
```go
func (f *FS) LoadRoutes() (map[string][]byte, error) {
    return f.loadDir("routes")
}

func (f *FS) LoadWorkflows() (map[string][]byte, error) {
    return f.loadDir("workflows")
}
```

注释也同步更新。

#### 3. `internal/config/yaml/yaml_test.go`

MapFS 的 key 需要同步更新。例如：

```go
// 当前
"config/routes/agent.yaml": {Data: []byte("...")},
"config/workflows/agent.yaml": {Data: []byte("...")},

// 精简后
"routes/agent.yaml": {Data: []byte("...")},
"workflows/agent.yaml": {Data: []byte("...")},
```

所有 6 处 MapFS key 需要更新。

#### 4. `internal/workflow/workflow_test.go:158`

当前：
```go
data, err := os.ReadFile("../../cmd/ckjr-cli/config/workflows/agent.yaml")
```

精简后：
```go
data, err := os.ReadFile("../../cmd/ckjr-cli/workflows/agent.yaml")
```

#### 5. 文件物理迁移

```
cmd/ckjr-cli/config/routes/agent.yaml  -> cmd/ckjr-cli/routes/agent.yaml
cmd/ckjr-cli/config/routes/common.yaml -> cmd/ckjr-cli/routes/common.yaml
cmd/ckjr-cli/config/workflows/agent.yaml -> cmd/ckjr-cli/workflows/agent.yaml
```

迁移后删除空的 `cmd/ckjr-cli/config/` 目录。

#### 6. 文档引用更新

| 文件 | 变更 |
|------|------|
| `wiki/core-concepts.md` | `cmd/ckjr-cli/config/routes/` -> `cmd/ckjr-cli/routes/`（2 处），`cmd/ckjr-cli/config/workflows/` -> `cmd/ckjr-cli/workflows/`（2 处） |
| `wiki/extending.md` | `cmd/ckjr-cli/config/routes/` -> `cmd/ckjr-cli/routes/`（5 处） |
| `wiki/project-structure.md` | `cmd/ckjr-cli/config/routes/*.yaml` -> `cmd/ckjr-cli/routes/*.yaml`（1 处） |

### YAML 路径精简的 TDD 步骤

建议作为独立的一组变更，在 cmd 子包拆分之前或之后执行（两者互不依赖，但建议先做路径精简，减少后续混淆）：

1. **先修改 `internal/config/yaml/yaml_test.go`** - 更新 MapFS key（`config/routes` -> `routes`，`config/workflows` -> `workflows`），确认测试失败（当前路径不存在）
2. **修改 `internal/config/yaml/yaml.go`** - 更新 loadDir 路径，测试通过
3. **修改 `cmd/ckjr-cli/embed.go`** - embed 指令从 `all:config` 改为 `all:routes all:workflows`
4. **迁移物理文件** - `config/routes/*.yaml` -> `routes/`，`config/workflows/*.yaml` -> `workflows/`
5. **删除 `cmd/ckjr-cli/config/` 目录**
6. **修改 `internal/workflow/workflow_test.go`** - 更新 `os.ReadFile` 路径
7. **更新 wiki 文档** - 3 个文件共约 9 处引用
8. **运行 `go test ./...`** 确认全部通过
