# Routes Resource to Name 实施计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 routes YAML 配置中的 `resource` 字段重命名为 `name`，与 workflows YAML 保持命名一致性。

**Architecture:**
- 修改 Go 结构体 `RouteConfig.Resource` → `RouteConfig.Name`
- 更新所有使用该字段的代码（cmdgen, yamlgen, route）
- 更新 CLI 参数 `--resource` → `--name`
- 更新现有 YAML 文件

**Tech Stack:** Go 1.23, Cobra CLI, gopkg.in/yaml.v3

---

## 文件结构

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/router/router.go` | 修改 | 结构体字段 `Resource` → `Name` |
| `internal/cmdgen/cmdgen.go` | 修改 | `cfg.Resource` → `cfg.Name` |
| `internal/yamlgen/generate.go` | 修改 | `resource` 参数 → `name` 参数 |
| `cmd/route.go` | 修改 | CLI 参数重命名 |
| `cmd/routes/agent.yaml` | 修改 | `resource` → `name` |
| `cmd/routes/common.yaml` | 修改 | `resource` → `name` |
| `internal/router/router_test.go` | 修改 | 测试断言更新 |
| `cmd/route_test.go` | 修改 | 测试断言更新 |
| `internal/yamlgen/generate_test.go` | 修改 | 测试断言更新 |

---

### Task 1: 修改 RouteConfig 结构体

**Files:**
- Modify: `internal/router/router.go:26-31`

- [ ] **Step 1: 修改结构体字段定义**

```go
// RouteConfig 路由配置
type RouteConfig struct {
    Name        string          `yaml:"name"`
    Description string          `yaml:"description"`
    Routes      map[string]Route `yaml:"routes"`
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/router/... -v`
Expected: FAIL (cfg.Resource 不存在)

- [ ] **Step 3: 提交修改**

```bash
git add internal/router/router.go
git commit -m "refactor(router): rename Resource field to Name"
```

---

### Task 2: 更新 cmdgen 代码

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:25,30`

- [ ] **Step 1: 修改 cfg.Resource 为 cfg.Name**

```go
// BuildCommand 从路由配置构建 cobra 命令
func BuildCommand(cfg *router.RouteConfig, clientFactory APIClientFactory) *cobra.Command {
    cmd := &cobra.Command{
        Use:   cfg.Name,  // 原: cfg.Resource
        Short: cfg.Description,
    }

    for name, route := range cfg.Routes {
        subCmd := buildSubCommand(cfg.Name, name, route, clientFactory)  // 原: cfg.Resource
        cmd.AddCommand(subCmd)
    }

    return cmd
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./internal/cmdgen/... -v`
Expected: PASS

- [ ] **Step 3: 提交修改**

```bash
git add internal/cmdgen/cmdgen.go
git commit -m "refactor(cmdgen): use cfg.Name instead of cfg.Resource"
```

---

### Task 3: 更新 yamlgen 代码

**Files:**
- Modify: `internal/yamlgen/generate.go:51,57`

- [ ] **Step 1: 修改参数名和字段名**

```go
// CreateFile 创建新的 YAML 路由文件
func CreateFile(path string, name string, nameDesc string, routeName string, route router.Route) error {
    if _, err := os.Stat(path); err == nil {
        return fmt.Errorf("文件已存在: %s", path)
    }

    cfg := &router.RouteConfig{
        Name:        name,  // 原: Resource
        Description: nameDesc,  // 原: resourceDesc
        Routes:      map[string]router.Route{routeName: route},
    }
    return writeConfig(path, cfg)
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./internal/yamlgen/... -v`
Expected: PASS

- [ ] **Step 3: 提交修改**

```bash
git add internal/yamlgen/generate.go
git commit -m "refactor(yamlgen): use Name field instead of Resource"
```

---

### Task 4: 更新 route 命令 CLI 参数

**Files:**
- Modify: `cmd/route.go:28-29,50,60,78-81,158-159`

- [ ] **Step 1: 修改参数定义**

```go
func init() {
    routeImportCmd.Flags().String("curl", "", "curl 命令字符串")
    routeImportCmd.Flags().StringP("file", "f", "", "目标 YAML 文件路径")
    routeImportCmd.Flags().StringP("name", "n", "", "路由名称（默认从 URL 推导）")
    routeImportCmd.Flags().String("name-desc", "", "资源名称描述")  // 原: --resource-desc

    routeCmd.AddCommand(routeImportCmd)
}
```

- [ ] **Step 2: 修改 runImport 函数签名和逻辑**

```go
var routeImportCmd = &cobra.Command{
    Use:   "import",
    Short: "从 curl 命令导入路由配置",
    Long:  "解析 curl 命令并生成 YAML 路由配置。支持 stdin 管道输入或 --curl 参数。",
    RunE: func(cmd *cobra.Command, args []string) error {
        curlStr, _ := cmd.Flags().GetString("curl")
        file, _ := cmd.Flags().GetString("file")
        name, _ := cmd.Flags().GetString("name")
        nameDesc, _ := cmd.Flags().GetString("name-desc")  // 原: resource-desc

        // 从 stdin 读取
        if curlStr == "" {
            stat, _ := os.Stdin.Stat()
            if (stat.Mode() & os.ModeCharDevice) == 0 {
                data, err := io.ReadAll(os.Stdin)
                if err != nil {
                    return fmt.Errorf("读取 stdin 失败: %w", err)
                }
                curlStr = string(data)
            }
        }

        if curlStr == "" {
            return fmt.Errorf("请通过 --curl 参数或 stdin 管道提供 curl 命令")
        }
        if file == "" {
            return fmt.Errorf("请通过 --file 参数指定目标 YAML 文件路径")
        }

        if err := runImport(curlStr, file, name, nameDesc); err != nil {  // 移除 resource 参数
            return err
        }

        fmt.Fprintf(os.Stdout, "已添加路由 %s 到 %s\n", name, file)
        return nil
    },
}

// runImport 核心逻辑，方便测试
func runImport(curlStr, file, routeName, nameDesc string) error {  // 移除 resource 参数
    result, err := curlparse.Parse(curlStr)
    if err != nil {
        return fmt.Errorf("curl 解析失败: %w", err)
    }

    // 自动推导 route name
    if routeName == "" {
        routeName = inferRouteName(result.Path)
    }

    route := yamlgen.GenerateRoute(result)

    // 判断追加还是新建
    if _, err := os.Stat(file); err == nil {
        return yamlgen.AppendToFile(file, routeName, route)
    }

    if nameDesc == "" {
        return fmt.Errorf("新建文件需要通过 --name-desc 指定资源描述")
    }
    return yamlgen.CreateFile(file, routeName, nameDesc, routeName, route)  // 参数顺序调整
}
```

- [ ] **Step 3: 运行测试**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 4: 提交修改**

```bash
git add cmd/route.go
git commit -m "refactor(route): rename --resource to --name CLI flag"
```

---

### Task 5: 更新 YAML 文件

**Files:**
- Modify: `cmd/routes/common.yaml:1-3`
- Modify: `cmd/routes/agent.yaml:1-3`

- [ ] **Step 1: 更新 common.yaml**

```yaml
name: common  # 原: resource: common
description: 平台公共接口，可以获取公众号端访问 URL 等
routes:
    qrcodeImg:
        method: GET
        path: /admin/common/qrcodeImg
        description: 获取公众号端访问 URL 和二维码图片
        template:
            prodId:
                description: 产品ID
                required: true
                type: int
            prodType:
                description: 产品类型, ai_service-智能体
                required: true
                example: ai_service
```

- [ ] **Step 2: 更新 agent.yaml**

```yaml
name: agent  # 原: resource: agent
description: 智能体相关接口
routes:
    create:
        method: POST
        path: /admin/agent/create
        description: 创建智能体
        # ... 其余内容保持不变
```

- [ ] **Step 3: 验证 YAML 解析**

Run: `go run . agent describe`
Expected: 正常显示 agent 命令帮助

- [ ] **Step 4: 提交修改**

```bash
git add cmd/routes/common.yaml cmd/routes/agent.yaml
git commit -m "refactor(routes): rename resource to name in YAML files"
```

---

### Task 6: 更新测试文件

**Files:**
- Modify: `internal/router/router_test.go`
- Modify: `cmd/route_test.go`
- Modify: `internal/yamlgen/generate_test.go`

- [ ] **Step 1: 更新 router_test.go**

```go
func TestParse(t *testing.T) {
    data := []byte(`
name: test  # 原: resource: test
description: Test resource
routes:
    get:
        method: GET
        path: /test
        template: {}
`)

    cfg, err := Parse(data)
    assert.NoError(t, err)
    assert.Equal(t, "test", cfg.Name)  // 原: cfg.Resource
    assert.Equal(t, "Test resource", cfg.Description)
}
```

- [ ] **Step 2: 更新 route_test.go**

```go
func TestRunImport(t *testing.T) {
    // 更新测试中的 resource 参数为 name
    // ...
}

func TestRunImportNewFile(t *testing.T) {
    // 更新建文件测试，使用 name-desc 替代 resource-desc
    // ...
}
```

- [ ] **Step 3: 更新 generate_test.go**

```go
func TestCreateFile(t *testing.T) {
    route := router.Route{
        Method:   "GET",
        Path:     "/test",
        Template: map[string]router.Field{},
    }

    err := CreateFile(tmpfile, "test", "Test description", "get", route)  // 原: resource, resourceDesc
    assert.NoError(t, err)

    data, _ := os.ReadFile(tmpfile)
    cfg, _ := router.Parse(data)
    assert.Equal(t, "test", cfg.Name)  // 原: cfg.Resource
}
```

- [ ] **Step 4: 运行所有测试**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 5: 提交修改**

```bash
git add internal/router/router_test.go cmd/route_test.go internal/yamlgen/generate_test.go
git commit -m "test: update assertions for Name field"
```

---

### Task 7: 验证和集成测试

- [ ] **Step 1: 运行完整测试套件**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 2: 测试 CLI 命令**

```bash
# 测试 route import 新建文件
echo 'curl -X POST https://api.example.com/test' | go run . route import --file /tmp/test.yaml --name test --name-desc "Test resource"

# 验证生成的 YAML
cat /tmp/test.yaml | grep "^name: test"

# 测试生成的命令
go run . test describe
```

Expected: 正常工作

- [ ] **Step 3: 最终提交**

```bash
git add -A
git commit -m "chore: final verification for resource to name migration"
```

---

## 验收标准

1. 所有测试通过
2. `go run . agent describe` 显示正确的 agent 命令
3. `go run . route import --name-desc` 正常工作
4. 生成的 YAML 文件使用 `name` 字段
5. 代码中不再有 `cfg.Resource` 引用

## 注意事项

- **破坏性变更**：现有使用 `resource` 字段的 YAML 文件将无法解析
- **参数变更**：CLI 参数 `--resource` 和 `--resource-desc` 已废弃
- **变量命名**：Go 代码中变量名 `resource`/`resourceDesc` 改为 `name`/`nameDesc`
