# YAML 配置文件兜底测试验证 - 设计规格

## 背景

ckjr-cli 的 routes 和 workflows 配置以 YAML 文件形式维护（`cmd/ckjr-cli/routes/*.yaml`、`cmd/ckjr-cli/workflows/*.yaml`），通过 `go:embed` 嵌入二进制。人工编辑这些文件时容易引入错误（字段名拼错、值格式不对、引用不存在等），需要在发布前通过 `go test` 自动拦截。

## 目标

在 `go test` 中验证所有嵌入的 YAML 配置文件的结构完整性和语义正确性，覆盖三层：

1. **基础结构验证**：YAML 能正确解析为 Go 结构体，关键字段非空
2. **字段语义校验**：字段值符合业务规则（method 合法、path 格式正确、type 合法等）
3. **跨文件引用校验**：workflow 引用的 command 在 routes 中存在

## 验证规则

### Routes (*.yaml)

#### 基础结构
- 解析为 `RouteConfig` 不报错
- `name` 非空
- `description` 非空
- `routes` 至少有一个条目

#### 每个 Route
- `method` 非空且是合法 HTTP 方法（GET/POST/PUT/DELETE/PATCH）
- `path` 非空且以 `/` 开头
- `description` 非空
- `template` 中每个 Field 的 `description` 非空

#### 每个 Field（template 中的字段）
- `type` 为空或为合法值（string/int/float/bool/array）
- 如有 `min`/`max`，`min <= max`
- 如有 `minLength`/`maxLength`，`minLength <= maxLength`
- 如有 `default` 且有 `type`，default 值的类型应与 type 兼容

### Workflows (*.yaml)

#### 基础结构
- 解析为 `Config` 不报错
- `name` 非空
- `description` 非空
- `workflows` 至少有一个条目

#### 每个 Workflow
- `description` 非空
- `steps` 至少有一个条目
- 每个 Input 的 `name` 和 `description` 非空
- 每个 Step 的 `id`、`description`、`command` 非空

#### 每个 Step 的 command 引用校验
- `command` 格式为 `"<routeName> <actionName>"`
- `<routeName>` 对应的 route YAML 文件存在（如 `agent` 对应 `agent.yaml`）
- `<actionName>` 在该 route 的 `routes` map 中存在

## 实现方案

### 新增文件

- `cmd/ckjr-cli/yaml_validate_test.go` — 集成测试，读取 embed.FS 中的实际 YAML 文件并验证

### 测试结构

```
TestYAMLValidation
├── TestAllRoutes                     // 遍历所有 routes YAML
│   ├── 解析成功
│   ├── name/description/routes 非空
│   └── 每个 route: method/path/description/template 验证
├── TestAllWorkflows                  // 遍历所有 workflows YAML
│   ├── 解析成功
│   ├── name/description/workflows 非空
│   └── 每个 workflow: steps/inputs 验证
└── TestWorkflowCommandReferences     // 跨文件引用验证
    └── 每个 workflow step 的 command 在 routes 中存在
```

### 关键设计决策

1. **测试位置**：放在 `cmd/ckjr-cli/` 下，因为该包可以直接访问 embed.FS
2. **使用 table-driven tests**：每个 YAML 文件是一个 test case，报告具体哪个文件的哪个字段有问题
3. **错误报告**：使用 `t.Errorf` 而非 `t.Fatalf`，一个文件出错不影响其他文件的测试
4. **embed.FS 直接访问**：测试读取实际嵌入的文件，而非硬编码路径，确保测试的就是发布时的内容
