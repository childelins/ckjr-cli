# curl-to-yaml 设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

用户从浏览器开发者工具复制 curl 命令后，通过 `ckjr-cli route import` 命令自动解析并生成 YAML 路由配置。能明确提取的字段（URL path、method、JSON body keys）直接生成，模糊字段（description）用占位符让用户后续补充。

## CLI 接口

```bash
# 从 stdin 读取 curl（推荐，避免转义问题）
pbpaste | ckjr-cli route import --file cmd/routes/agent.yaml --name update

# 直接传参
ckjr-cli route import --curl "curl 'https://...' --data-raw '{...}'" --file cmd/routes/agent.yaml --name update

# 新建文件
pbpaste | ckjr-cli route import --file cmd/routes/order.yaml --name list --resource order --resource-desc "订单管理"
```

**参数说明：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `--curl` | 否 | curl 命令字符串（与 stdin 二选一） |
| `--file` / `-f` | 是 | 目标 YAML 文件路径 |
| `--name` / `-n` | 否 | route 名称，默认从 URL 末段推导 |
| `--resource` | 否 | resource 名称（新建文件时必填） |
| `--resource-desc` | 否 | resource 描述（新建文件时可选） |

## 架构

```
cmd/route.go                  -- route 命令组 + import 子命令
internal/curlparse/parse.go   -- curl 字符串解析
internal/curlparse/parse_test.go
internal/yamlgen/generate.go  -- YAML 路由条目生成 + 文件读写
internal/yamlgen/generate_test.go
```

## 组件

### 1. curlparse - curl 命令解析器

**职责：** 从 curl 命令字符串中提取 method、URL path、JSON body fields。

```go
package curlparse

// Result 保存 curl 解析结果
type Result struct {
    Method string            // HTTP method (GET/POST/PUT/DELETE...)
    Path   string            // URL path 部分 (不含 host)
    Fields map[string]Field  // 从 JSON body 提取的字段
}

// Field 解析出的字段信息
type Field struct {
    Example interface{} // 原始值作为 example
    Type    string      // 推断的类型: string/int/bool
}

// Parse 解析 curl 命令字符串
func Parse(curl string) (*Result, error)
```

**解析逻辑：**

1. **预处理**：去除行尾 `\` 续行符，合并为单行
2. **提取 URL**：匹配 `curl` 后第一个引号包裹或空格分隔的 URL
3. **提取 method**：
   - 有 `-X`/`--request` 参数则取其值
   - 有 `--data-raw`/`-d`/`--data` 则默认 POST
   - 否则默认 GET
4. **提取 path**：从 URL 中解析 path 部分（忽略 host、scheme、query）
5. **提取 fields**：解析 `--data-raw` 的 JSON body，遍历顶层 key，推断类型：
   - Go `float64` 且无小数 -> `int`
   - Go `bool` -> `bool`
   - Go `string` -> `string`（默认）
   - 数组/对象 -> 跳过（复杂类型不映射为 CLI field）

**注意事项：**
- 仅解析顶层 key，嵌套对象/数组不展开
- Header 信息（authorization 等）不提取，因为 ckjr-cli 有自己的认证机制
- Query string 参数暂不解析（当前 API 都是 POST + JSON body）

### 2. yamlgen - YAML 路由生成器

**职责：** 将 curlparse.Result 转换为 YAML 格式，支持追加到已有文件或新建文件。

```go
package yamlgen

import "github.com/childelins/ckjr-cli/internal/router"

// Options 生成选项
type Options struct {
    RouteName    string // route 名称（如 "list", "update"）
    Resource     string // resource 名称（新建文件时使用）
    ResourceDesc string // resource 描述（新建文件时使用）
}

// GenerateRoute 从 curlparse.Result 生成 Route
func GenerateRoute(result *curlparse.Result) router.Route

// AppendToFile 追加路由到已有 YAML 文件
func AppendToFile(path string, name string, route router.Route) error

// CreateFile 创建新的 YAML 路由文件
func CreateFile(path string, opts Options, route router.Route) error
```

**生成规则：**

- `method`: 直接取 curlparse 结果
- `path`: 直接取 curlparse 结果
- `description`: 固定 `"TODO: 补充描述"`
- `template` 中的每个 field：
  - `description`: 固定 `"TODO"`
  - `required`: 默认 `false`
  - `type`: 取 curlparse 推断结果（int/bool 时写入，string 时省略）
  - `example`: 如果原始值不是复杂类型，转为字符串写入

**追加模式流程：**
1. 读取已有 YAML 文件 -> router.Parse 解析
2. 检查 route name 是否已存在，已存在则报错
3. 在 routes map 中添加新 route
4. 整体序列化写回文件

**新建模式流程：**
1. 检查文件是否已存在，已存在则报错
2. 构建完整 RouteConfig
3. 序列化写入文件

### 3. cmd/route.go - route 命令组

```go
// route 命令组
var routeCmd = &cobra.Command{
    Use:   "route",
    Short: "路由配置管理",
}

// route import 子命令
var routeImportCmd = &cobra.Command{
    Use:   "import",
    Short: "从 curl 命令导入路由配置",
    Run:   runRouteImport,
}
```

在 `cmd/root.go` 的 `init()` 中注册 `routeCmd`。

## 数据流

```
浏览器复制 curl
       |
       v
stdin / --curl 参数
       |
       v
curlparse.Parse(curl) -> Result{Method, Path, Fields}
       |
       v
yamlgen.GenerateRoute(result) -> router.Route
       |
       v
--file 存在?
  |         |
  是        否
  |         |
  v         v
AppendToFile  CreateFile
  |         |
  v         v
写入 YAML 文件
       |
       v
输出: "已添加路由 update 到 cmd/routes/agent.yaml"
```

## Route Name 推导

当 `--name` 未指定时，从 URL path 末段推导：

```
/admin/aiCreationCenter/modifyApp  -> modifyApp -> modify (去掉常见后缀 App/List/Info)
/admin/aiCreationCenter/listApp    -> listApp   -> list
/admin/aiCreationCenter/describeApp -> describeApp -> describe -> get (常见映射)
/admin/aiCreationCenter/createApp  -> createApp  -> create
/admin/aiCreationCenter/deleteApp  -> deleteApp  -> delete
```

**映射表：**
- modify* -> update
- describe*/get*Info -> get
- 其他保持原样去掉后缀

如果推导结果不确定，保持原始末段并提示用户可通过 `--name` 覆盖。

## 错误处理

| 场景 | 处理 |
|------|------|
| curl 格式无法解析 | 报错并提示正确的 curl 格式 |
| JSON body 解析失败 | 报错，提示检查 --data-raw 部分 |
| 目标文件不存在（追加模式） | 提示使用 --resource 创建新文件 |
| route name 已存在 | 报错，提示已存在的 route name |
| YAML 写入失败 | 报错并提示检查文件权限 |

## 测试策略

### curlparse 测试

```go
// 基本 POST + JSON body
func TestParse_PostWithBody(t *testing.T)

// GET 请求（无 body）
func TestParse_GetRequest(t *testing.T)

// 多行 curl（带 \ 续行）
func TestParse_MultiLine(t *testing.T)

// body 中各种类型推断
func TestParse_TypeInference(t *testing.T)

// 无效 curl
func TestParse_Invalid(t *testing.T)

// 复杂嵌套 body（只提取顶层 key）
func TestParse_NestedBody(t *testing.T)
```

### yamlgen 测试

```go
// 生成 route
func TestGenerateRoute(t *testing.T)

// 追加到已有文件
func TestAppendToFile(t *testing.T)

// 追加时 name 冲突
func TestAppendToFile_Conflict(t *testing.T)

// 创建新文件
func TestCreateFile(t *testing.T)

// 创建时文件已存在
func TestCreateFile_Exists(t *testing.T)
```

### 集成测试

```go
// 端到端: curl -> YAML 文件
func TestRouteImport_EndToEnd(t *testing.T)
```

## 实现注意事项

1. **curl 解析不用做完美**：浏览器复制的 curl 格式相对固定（单引号包裹 URL，`--data-raw` 传 body），覆盖这个场景即可，不需要支持所有 curl 选项。

2. **YAML 序列化顺序**：gopkg.in/yaml.v3 的 map 序列化顺序不固定。如果需要保持字段顺序（resource -> description -> routes），需要使用 `yaml.Node` API 或手动模板拼接。初版可接受无序输出，后续按需优化。

3. **不修改现有模块**：curlparse 和 yamlgen 是纯新增包，不改动 router/cmdgen 等已有代码。router.Route/Field 结构体作为共享数据类型使用。

4. **stdin 优先**：curl 命令通常很长且包含特殊字符，stdin 管道输入比 --curl 参数更可靠，推荐作为主要使用方式。

5. **安全考虑**：curl 中的 Authorization header 包含 JWT token，解析时应忽略而非写入 YAML。生成的 YAML 不应包含任何敏感信息。
