# YAML 配置文件迁移设计文档

> Created: 2026-03-27
> Status: Draft

## 概述

将 `cmd/routes/` 和 `cmd/workflows/` 下的 YAML 配置文件迁移到根目录的 `config/routes/` 和 `config/workflows/`。由于 Go `//go:embed` 只能引用当前 Go 文件所在目录及子目录的相对路径，迁移后 `cmd/` 包无法直接 embed `config/` 下的文件，需要新建 `internal/config/yaml` 包集中管理 embed 和加载逻辑。

## 背景

当前 YAML 文件放在 `cmd/` 包目录下，导致 CLI 命令代码与配置数据混在同一目录。迁移目标：

1. 将配置数据从 `cmd/` 中分离，使目录结构更清晰
2. 通过 `internal/config/yaml` 包集中管理 YAML 的 embed 和加载，提高内聚性

## 当前状态

```
cmd/
  routes/
    agent.yaml         # 路由配置（智能体模块）
    common.yaml        # 路由配置（公共接口模块）
  workflows/
    agent.yaml         # 工作流配置（智能体工作流）
```

涉及 embed 的代码：

- `cmd/root.go:18` -- `//go:embed routes` + `routesFS.ReadDir("routes")` + `routesFS.ReadFile("routes/" + name)`
- `cmd/workflow.go:15` -- `//go:embed workflows` + `fs.ReadDir(workflowsFS, "workflows")` + `workflowsFS.ReadFile("workflows/" + entry.Name())`

涉及文件路径引用的测试：

- `internal/workflow/workflow_test.go:158` -- `os.ReadFile("../../cmd/workflows/agent.yaml")`

## 目标状态

```
config/
  routes/
    agent.yaml
    common.yaml
  workflows/
    agent.yaml
internal/config/yaml/
  embed.go            # embed 声明和加载函数
  embed_test.go       # 测试
```

## 架构

### 新增包：`internal/config/yaml`

负责声明 `//go:embed` 并提供加载函数。选择这个路径的原因：

- `internal/config/yaml` 与已有的 `internal/config` 包职责不同（后者管理用户配置 `~/.ckjr/config.json`），作为子包共存合理
- 集中管理所有 YAML embed，未来新增配置文件类型只需在此扩展
- `cmd/` 包不再持有 embed 变量，职责更单一

### 包接口设计

```go
package yaml

//go:embed all:config
var configFS embed.FS

// RoutesFS 返回 embed 的 routes 目录对应的 fs.FS
// 注意：embed 路径需要从 Go 文件位置引用
func RoutesFS() fs.FS { ... }

// WorkflowsFS 返回 embed 的 workflows 目录对应的 fs.FS
func WorkflowsFS() fs.FS { ... }

// LoadRoutes 加载所有路由 YAML 文件，返回文件名到内容的映射
func LoadRoutes() (map[string][]byte, error) { ... }

// LoadWorkflows 加载所有工作流 YAML 文件，返回文件名到内容的映射
func LoadWorkflows() (map[string][]byte, error) { ... }
```

**关键细节**：`//go:embed` 指令的路径是相对于 Go 源文件所在目录的。`internal/config/yaml/embed.go` 位于 `internal/config/yaml/`，要引用根目录的 `config/routes/` 需要使用 `../../config` 这样的相对路径。但 Go embed 不支持 `..` 路径。

**因此，embed 声明必须放在能直接引用 `config/` 的位置。** 有两种可行方案：

**方案 A（推荐）：在 `internal/config/yaml/` 中创建一个 Go 文件，将 embed 的 YAML 文件软链接或复制到该目录下。**

不，这样做不好。更好的方案：

**方案 B（最终方案）：将 embed 声明放在 `internal/config/yaml/` 包中，但使用 `//go:embed` 引用包目录下的子目录。实际操作：在 `internal/config/yaml/` 下创建 `routes/` 和 `workflows/` 的符号链接指向根目录的 `config/routes/` 和 `config/workflows/`。**

不，Go embed 不跟随符号链接。

**方案 C（最终方案）：将 embed 声明放在项目根目录附近的包中。最实际的做法是创建 `internal/config/yaml/` 包，在其中放置实际要 embed 的 YAML 文件（即 YAML 文件的实际位置就是 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/`），或者使用 `all:` 前缀和正确的路径。**

重新分析：由于 Go embed 的限制（不能引用上级目录），如果用户要求 YAML 文件放在根目录 `config/` 下，则 embed 声明必须在根目录下的某个 Go 包中。唯一在根目录下的 Go 包是 `main` 包（`main.go`）。

**最终方案**：

在根目录创建一个 Go 包来处理 embed，但由于根目录已经是 `main` 包，需要换一种方式。实际可行的方案是：

1. YAML 文件物理位置放在根目录 `config/routes/` 和 `config/workflows/`
2. 新建 `internal/config/yaml` 包，但该包不直接 embed 根目录的文件
3. 在 `main.go`（根目录 `main` 包）中声明 embed，然后将 `embed.FS` 通过函数参数传给 `cmd` 包
4. 或者：接受 YAML 文件实际放在 `internal/config/yaml/` 下的 `routes/` 和 `workflows/` 子目录，然后通过构建脚本在 `config/` 下创建符号链接供文档/人工使用

**再思考**：最简洁的方案是：

1. 创建 `internal/config/yaml` 包
2. YAML 文件实际存放在 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/`
3. 在根目录 `config/` 下放置指向实际文件的符号链接（用于文档/人工编辑）
4. `internal/config/yaml` 包负责 embed 和加载

但这增加了维护复杂度（符号链接）。

**最简方案（采用）**：

重新审视需求——用户要求将 YAML 文件放到根目录 `config/routes` 和 `config/workflows`。考虑到 Go embed 的限制，最实际的做法是：

1. YAML 文件放在根目录 `config/routes/` 和 `config/workflows/`
2. 在 `cmd/` 包中使用 `//go:embed all:../../config/routes all:../../config/workflows` — 不行，Go 不允许 `..` 路径
3. 因此，embed 必须在根目录或根目录的子目录中的 Go 文件里声明

**最终确定的方案**：

```
config/
  routes/
    agent.yaml
    common.yaml
  workflows/
    agent.yaml
internal/config/yaml/
  embed.go          # embed 声明 + 加载函数
  embed_test.go
```

在 `internal/config/yaml/embed.go` 中：

```go
package yaml

import (
    "embed"
    "io/fs"
)

//go:embed all:routes
var routesFS embed.FS

//go:embed all:workflows
var workflowsFS embed.FS
```

**这意味着 YAML 文件实际需要放在 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/` 目录下。**

但用户明确要求放在根目录 `config/` 下。这两个目标冲突。

**解决方案**：使用 Makefile/构建脚本在编译前将 `config/` 下的 YAML 复制到 `internal/config/yaml/` 对应位置。不，这太复杂了。

**真正的解决方案**：接受 YAML 文件的物理位置就在 `internal/config/yaml/` 下，在根目录 `config/` 创建符号链接指向它们。这样：
- 文档和用户看到的路径是 `config/routes/` 和 `config/workflows/`（通过符号链接）
- Go embed 引用的是 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/`（实际文件）
- `route import --file` 命令的目标路径可以指向 `config/routes/xxx.yaml`（符号链接同样可写）

实际上，**重新考虑**：最简洁且满足用户需求的方案是让 embed 声明在 `internal/config/yaml` 包中，同时将 YAML 文件作为该包的子目录存放。根目录的 `config/` 目录不存放实际文件，而是在 wiki/文档中将路径统一描述为 `config/routes/` 和 `config/workflows/`。

但这与用户"放到根目录下的 config/"的要求不符。

**最终最终方案**：YAML 文件放在根目录 `config/routes/` 和 `config/workflows/`。在 `main.go` 同目录（根目录）创建一个 `embed.go` 文件声明 embed，但由于根目录已经是 `main` 包，`embed.go` 也属于 `main` 包。然后通过 `cmd` 包的公开函数将 `embed.FS` 注入。

```go
// 根目录 embed.go（main 包）
package main

import "embed"

//go:embed all:config
var configFS embed.FS
```

然后在 `cmd/root.go` 中移除 embed 声明，改为接收 `embed.FS` 参数：

```go
// cmd/root.go
var routesFS embed.FS  // 改为在外部设置

func SetRoutesFS(fs embed.FS) {
    routesFS = fs
}
```

在 `main.go` 中：

```go
func main() {
    cmd.SetRoutesFS(configFS)
    cmd.Execute()
}
```

**问题**：`cmd/workflow.go` 也需要 `workflowsFS`。这需要两个 FS 或一个统一的 FS。

如果用 `//go:embed all:config` 得到一个包含 `config/` 前缀的 FS，路径变为 `config/routes/agent.yaml` 而非 `routes/agent.yaml`。

```go
// main 包
//go:embed all:config
var configFS embed.FS

func main() {
    cmd.SetConfigFS(configFS)
    cmd.Execute()
}
```

```go
// cmd/root.go
var configFS embed.FS

func SetConfigFS(fs embed.FS) {
    configFS = fs
}

func registerRouteCommands() {
    entries, err := configFS.ReadDir("config/routes")
    // ...
    data, err := configFS.ReadFile("config/routes/" + name)
    // ...
}
```

```go
// cmd/workflow.go
func loadAllWorkflows() ([]*workflow.Config, error) {
    entries, err := fs.ReadDir(configFS, "config/workflows")
    // ...
    data, err := configFS.ReadFile("config/workflows/" + entry.Name())
    // ...
}
```

这个方案的问题：
- `configFS` 变成了包级别的全局变量，在 `cmd` 包内两个文件共享
- `cmd` 包需要导出 `SetConfigFS` 供 `main` 调用
- 路径前缀从 `routes/` 变为 `config/routes/`

虽然可以工作，但违反了用户选择的方案 C（新建 `internal/config/yaml` 包集中管理）。

**综合方案**：结合方案 C 的精神和 Go embed 的实际约束：

1. YAML 文件放在根目录 `config/routes/` 和 `config/workflows/`
2. 在根目录（`main` 包）声明 `//go:embed all:config`
3. 创建 `internal/config/yaml` 包，提供加载函数，接收 `embed.FS` 作为参数
4. `main.go` 将 embed.FS 传给 `internal/config/yaml`，`cmd` 包从 `internal/config/yaml` 获取加载结果

```go
// internal/config/yaml/yaml.go
package yaml

import (
    "io/fs"
    "io"
    "fmt"
    "strings"
)

// FS 持有嵌入的文件系统
type FS struct {
    fs fs.FS
}

// New 创建一个新的 YAML 配置加载器
func New(embedFS fs.FS) *FS {
    return &FS{fs: embedFS}
}

// LoadRoutes 读取所有路由 YAML 文件
func (f *FS) LoadRoutes() (map[string][]byte, error) {
    entries, err := fs.ReadDir(f.fs, "config/routes")
    if err != nil {
        return nil, err
    }
    result := make(map[string][]byte)
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
            continue
        }
        data, err := f.fs.ReadFile("config/routes/" + entry.Name())
        if err != nil {
            return nil, fmt.Errorf("读取路由文件 %s 失败: %w", entry.Name(), err)
        }
        result[entry.Name()] = data
    }
    return result, nil
}

// LoadWorkflows 读取所有工作流 YAML 文件
func (f *FS) LoadWorkflows() (map[string][]byte, error) {
    entries, err := fs.ReadDir(f.fs, "config/workflows")
    if err != nil {
        return nil, err
    }
    result := make(map[string][]byte)
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
            continue
        }
        data, err := f.fs.ReadFile("config/workflows/" + entry.Name())
        if err != nil {
            return nil, fmt.Errorf("读取工作流文件 %s 失败: %w", entry.Name(), err)
        }
        result[entry.Name()] = data
    }
    return result, nil
}
```

**等等**，这样 `main` 包仍然需要 embed。那不如让 `internal/config/yaml` 提供加载逻辑，但 embed 在 `main` 中。这样 `internal/config/yaml` 是一个纯粹的加载工具包，不持有 embed 状态。

不，用户选择方案 C 的理由是"集中管理 YAML 的 embed 和加载逻辑"。如果 embed 在 main 中，加载在 yaml 包中，就不算"集中管理 embed"了。

**最终结论**：在 Go embed 的硬约束下，方案 C 的真正实现是：

1. YAML 文件放在 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/`（实际文件位置）
2. 根目录 `config/` 通过符号链接指向 `internal/config/yaml/routes/` 和 `internal/config/yaml/workflows/`
3. `internal/config/yaml` 包集中管理 embed 和加载

但符号链接在跨平台时可能有问题。

**最务实的方案**（放弃符号链接）：

1. YAML 文件放在根目录 `config/routes/` 和 `config/workflows/`
2. 在根目录 `main` 包中声明 embed（这是 Go embed 约束决定的，无法避免）
3. 新建 `internal/config/yaml` 包，提供加载函数（接收 `fs.FS` 参数），封装路径前缀和加载逻辑
4. `cmd` 包通过 `internal/config/yaml` 加载 YAML，不再直接操作 embed.FS

这个方案：
- YAML 文件在用户要求的 `config/` 路径下
- `internal/config/yaml` 集中管理加载逻辑和路径
- embed 声明在 `main` 中（Go 语言的硬约束）
- `cmd` 包解耦，不直接操作 embed

## 组件

### 1. `config/routes/` 和 `config/workflows/`（新增目录 + 文件迁移）

从 `cmd/routes/` 和 `cmd/workflows/` 移动 YAML 文件到此处。

### 2. 根目录 `embed.go`（新增文件，main 包）

```go
package main

import "embed"

//go:embed all:config
var configFS embed.FS
```

### 3. `internal/config/yaml/yaml.go`（新增包）

集中管理 YAML 加载逻辑，封装路径前缀。

```go
package yaml

import (
    "fmt"
    "io/fs"
    "strings"
)

type FS struct {
    fs fs.FS
}

func New(embedFS fs.FS) *FS {
    return &FS{fs: embedFS}
}

// LoadRoutes 读取 config/routes/ 下所有 .yaml 文件
func (f *FS) LoadRoutes() (map[string][]byte, error) {
    return f.loadDir("config/routes")
}

// LoadWorkflows 读取 config/workflows/ 下所有 .yaml 文件
func (f *FS) LoadWorkflows() (map[string][]byte, error) {
    return f.loadDir("config/workflows")
}

func (f *FS) loadDir(dir string) (map[string][]byte, error) {
    entries, err := fs.ReadDir(f.fs, dir)
    if err != nil {
        return nil, fmt.Errorf("读取目录 %s 失败: %w", dir, err)
    }
    result := make(map[string][]byte)
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
            continue
        }
        data, err := f.fs.ReadFile(dir + "/" + entry.Name())
        if err != nil {
            return nil, fmt.Errorf("读取文件 %s 失败: %w", entry.Name(), err)
        }
        result[entry.Name()] = data
    }
    return result, nil
}
```

### 4. `cmd/root.go`（修改）

移除 `//go:embed routes` 和 `routesFS` 变量。改为从 `yaml.FS` 加载。

```go
// 删除:
// //go:embed routes
// var routesFS embed.FS

// 新增包级别变量:
var yamlFS *configyaml.FS

func SetYAMLFS(fs *configyaml.FS) {
    yamlFS = fs
}

// registerRouteCommands 修改为使用 yamlFS:
func registerRouteCommands() {
    if yamlFS == nil {
        fmt.Fprintf(os.Stderr, "YAML 文件系统未初始化\n")
        return
    }
    files, err := yamlFS.LoadRoutes()
    if err != nil {
        fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
        return
    }
    for name, data := range files {
        cfg, err := router.Parse(data)
        // ... 后续逻辑不变
    }
}
```

### 5. `cmd/workflow.go`（修改）

移除 `//go:embed workflows` 和 `workflowsFS` 变量。改为从 `yaml.FS` 加载。

```go
// 删除:
// //go:embed workflows
// var workflowsFS embed.FS

// loadAllWorkflows 修改为使用 yamlFS:
func loadAllWorkflows() ([]*workflow.Config, error) {
    if yamlFS == nil {
        return nil, fmt.Errorf("YAML 文件系统未初始化")
    }
    files, err := yamlFS.LoadWorkflows()
    if err != nil {
        return nil, err
    }
    var configs []*workflow.Config
    for _, data := range files {
        cfg, err := workflow.Parse(data)
        if err != nil {
            return nil, err
        }
        configs = append(configs, cfg)
    }
    return configs, nil
}
```

### 6. `main.go`（修改）

传递 embed.FS 给 cmd 包。

```go
package main

import "github.com/childelins/ckjr-cli/cmd"

func main() {
    cmd.SetYAMLFS(configyaml.New(configFS))
    cmd.Execute()
}
```

### 7. `internal/workflow/workflow_test.go`（修改）

更新文件路径引用。

```go
// 修改前:
data, err := os.ReadFile("../../cmd/workflows/agent.yaml")
// 修改后:
data, err := os.ReadFile("../../config/workflows/agent.yaml")
```

### 8. `cmd/route_test.go`（检查）

测试中使用 `t.TempDir()` 创建临时文件，不涉及 embed 路径，无需修改。

### 9. 文档更新

需要更新路径引用的文档：

**wiki/**:
- `wiki/core-concepts.md` -- `cmd/routes/agent.yaml` -> `config/routes/agent.yaml`，`cmd/workflows/` -> `config/workflows/`
- `wiki/extending.md` -- `cmd/routes/` -> `config/routes/`（所有引用）
- `wiki/project-structure.md` -- 目录结构描述、数据流图、embed 示例代码

**docs/** 中的 spec 和 plan 文件为历史记录，不修改。

### 10. 清理

删除 `cmd/routes/` 和 `cmd/workflows/` 目录。

## 数据流

```
编译时:
  config/routes/*.yaml, config/workflows/*.yaml
      |
      v  (main 包 //go:embed all:config)
  configFS (embed.FS)
      |
      v  (main.go 传入 cmd.SetYAMLFS)
  yaml.FS
      |
      v  (cmd 包调用 yamlFS.LoadRoutes/LoadWorkflows)
  map[string][]byte
      |
      v  (router.Parse / workflow.Parse)
  运行时命令注册和执行
```

## 变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| Create | `config/routes/agent.yaml` | 从 `cmd/routes/agent.yaml` 移动 |
| Create | `config/routes/common.yaml` | 从 `cmd/routes/common.yaml` 移动 |
| Create | `config/workflows/agent.yaml` | 从 `cmd/workflows/agent.yaml` 移动 |
| Create | `internal/config/yaml/yaml.go` | YAML 加载包 |
| Create | `internal/config/yaml/yaml_test.go` | 加载函数测试 |
| Create | `embed.go`（根目录） | main 包 embed 声明 |
| Modify | `cmd/root.go` | 移除 embed，使用 yaml.FS |
| Modify | `cmd/workflow.go` | 移除 embed，使用 yaml.FS |
| Modify | `internal/workflow/workflow_test.go` | 更新文件路径 |
| Modify | `main.go` | 传入 yaml.FS |
| Modify | `wiki/core-concepts.md` | 更新路径引用 |
| Modify | `wiki/extending.md` | 更新路径引用 |
| Modify | `wiki/project-structure.md` | 更新目录结构描述 |
| Delete | `cmd/routes/agent.yaml` | 已迁移 |
| Delete | `cmd/routes/common.yaml` | 已迁移 |
| Delete | `cmd/workflows/agent.yaml` | 已迁移 |
| Delete | `cmd/routes/` | 空目录 |
| Delete | `cmd/workflows/` | 空目录 |

## 错误处理

- `yaml.FS` 未初始化时调用加载函数，返回明确错误（`"YAML 文件系统未初始化"`）
- `loadDir` 中 ReadDir 和 ReadFile 失败时返回带文件名的错误信息
- 非目录、非 .yaml 文件被静默跳过（与当前行为一致）

## 测试策略

### `internal/config/yaml/yaml_test.go`

使用 `testing/fstest.MapFS` 构造虚拟文件系统进行测试，不依赖实际 embed：

```go
func TestLoadRoutes(t *testing.T) {
    memFS := fstest.MapFS{
        "config/routes/agent.yaml": {Data: []byte("name: agent\ndescription: test\nroutes: {}")},
        "config/routes/readme.txt": {Data: []byte("ignored")},
        "config/routes/sub/.keep":  {Data: []byte("")}, // 子目录，应跳过
    }
    loader := yaml.New(memFS)
    files, err := loader.LoadRoutes()
    // 验证: 只返回 .yaml 文件，跳过非 .yaml 和子目录
}

func TestLoadRoutes_EmptyDir(t *testing.T) { ... }
func TestLoadRoutes_NonexistentDir(t *testing.T) { ... }
func TestLoadWorkflows(t *testing.T) { ... }
```

### `internal/workflow/workflow_test.go`

更新路径引用后运行现有测试确认解析不变。

### 集成测试

编译后验证：
- `ckjr-cli agent list --template` 正常输出
- `ckjr-cli workflow list` 正常输出
- `ckjr-cli workflow describe create-agent` 正常输出

## 实现注意事项

1. `//go:embed all:config` 中的 `all:` 前缀确保 embed 包含以 `.` 或 `_` 开头的文件（如果将来有的话）
2. `config/` 目录与 `internal/config/` 包不同，不会产生包名冲突
3. `internal/config/yaml` 的包导入路径为 `github.com/childelins/ckjr-cli/internal/config/yaml`，注意在代码中使用别名避免与 `internal/config` 冲突：`import configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"`
4. `cmd/route.go` 中的 `route import --file` 命令是运行时文件操作，不依赖 embed，用户使用时路径改为 `config/routes/xxx.yaml`（但此命令为隐藏命令，影响较小）
5. docs/ 下的 spec 和 plan 文件为历史记录，不做修改
