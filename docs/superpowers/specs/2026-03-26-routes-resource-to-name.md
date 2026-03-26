# Routes YAML resource 字段重命名为 name

> Created: 2026-03-26
> Status: Draft

## 概述

将 routes YAML 配置中的 `resource` 字段重命名为 `name`，与 workflows YAML 保持命名一致性。这是一次破坏性变更，不保留向后兼容。

**决策要点：**
- YAML 解析直接废弃 `resource` 字段，不向后兼容
- CLI 参数 `--resource` 改名为 `--name`
- Go 结构体字段使用 `Name`

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                      变更范围                             │
├─────────────────────────────────────────────────────────┤
│  1. YAML 文件 (2个)                                      │
│     - cmd/routes/agent.yaml                              │
│     - cmd/routes/common.yaml                             │
│                                                          │
│  2. Go 结构体 (1处)                                      │
│     - internal/router/router.go: RouteConfig.Resource    │
│                                                          │
│  3. 使用该字段的代码 (5个文件)                           │
│     - internal/cmdgen/cmdgen.go                          │
│     - internal/yamlgen/generate.go                       │
│     - cmd/route.go                                       │
│     - 相关测试文件                                       │
│                                                          │
│  4. CLI 参数 (2个)                                       │
│     - --resource → --name                                │
│     - --resource-desc → --name-desc                      │
└─────────────────────────────────────────────────────────┘
```

## 组件

### 1. RouteConfig 结构体

**变更前：**
```go
type RouteConfig struct {
    Resource    string          `yaml:"resource"`
    Description string          `yaml:"description"`
    Routes      map[string]Route `yaml:"routes"`
}
```

**变更后：**
```go
type RouteConfig struct {
    Name        string          `yaml:"name"`
    Description string          `yaml:"description"`
    Routes      map[string]Route `yaml:"routes"`
}
```

### 2. YAML 文件格式

**变更前：**
```yaml
resource: common
description: 平台公共接口
routes:
  qrcodeImg:
    method: GET
    path: /admin/common/qrcodeImg
```

**变更后：**
```yaml
name: common
description: 平台公共接口
routes:
  qrcodeImg:
    method: GET
    path: /admin/common/qrcodeImg
```

### 3. CLI 参数

| 变更前 | 变更后 | 说明 |
|--------|--------|------|
| `--resource` | `--name` | 资源名称 |
| `--resource-desc` | `--name-desc` | 资源描述 |

## 数据流

### route import 命令流程

```
用户输入:
  ckjr-cli route import \
    --curl "..." \
    --file cmd/routes/common.yaml \
    --name common \
    --name-desc "平台公共接口"

       ↓
route.go: runImport()
  - 解析参数
  - 调用 yamlgen.CreateFile(file, name, nameDesc, ...)
       ↓
yamlgen.CreateFile()
  - 创建 RouteConfig{Name: name, ...}
  - 写入 YAML (name 字段)
       ↓
生成文件: cmd/routes/common.yaml
  - name: common
```

### cmdgen 命令生成流程

```
读取 RouteConfig
       ↓
cfg.Name // 替代原 cfg.Resource
       ↓
生成子命令: Use: cfg.Name
```

## 错误处理

1. **YAML 解析失败**：现有 resource 字段的文件将无法解析，提示用户更新 YAML 文件
2. **CLI 参数废弃**：使用旧的 `--resource` 参数时提示使用 `--name`

## 测试策略

1. **单元测试**：更新所有测试文件中的断言，将 `cfg.Resource` 改为 `cfg.Name`
2. **集成测试**：验证 route import 命令正确生成 name 字段
3. **回归测试**：确保 cmdgen 正确使用 Name 字段生成命令

**需要更新的测试文件：**
- `internal/router/router_test.go`
- `cmd/route_test.go`
- `internal/yamlgen/generate_test.go`

## 实施步骤

### 阶段 1：代码修改
1. 修改 `internal/router/router.go`：`RouteConfig.Resource` → `RouteConfig.Name`
2. 修改 `internal/cmdgen/cmdgen.go`：`cfg.Resource` → `cfg.Name`
3. 修改 `internal/yamlgen/generate.go`：`resource` 参数 → `name` 参数
4. 修改 `cmd/route.go`：
   - `--resource` → `--name`
   - `--resource-desc` → `--name-desc`
   - 相关变量名更新

### 阶段 2：测试更新
1. 更新 `internal/router/router_test.go` 断言
2. 更新 `cmd/route_test.go` 断言
3. 更新 `internal/yamlgen/generate_test.go` 断言

### 阶段 3：YAML 文件更新
1. 更新 `cmd/routes/agent.yaml`：`resource` → `name`
2. 更新 `cmd/routes/common.yaml`：`resource` → `name`

### 阶段 4：验证
1. 运行所有测试
2. 测试 route import 命令
3. 验证生成的 CLI 命令正常工作

## 实施注意事项

1. **一次性变更**：由于不保留向后兼容，需确保所有 YAML 文件同时更新
2. **变量命名**：Go 代码中与 `resource` 相关的变量名统一改为 `name`，如 `resource` → `name`，`resourceDesc` → `nameDesc`
3. **注释更新**：代码注释中的 "resource" 改为 "name"
4. **帮助文本**：CLI 参数的帮助文本同步更新
