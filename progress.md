# Progress

## 2026-03-25 ckjr-cli 实现

### Phase 1: 项目初始化
- Status: complete (30ebabe)
- 创建 go.mod，初始化 Go 模块
- 创建 main.go 骨架
- 验证编译通过

### Phase 2: 配置模块 (internal/config)
- Status: complete (68308f1)
- 实现 Config 结构体、Load、Save、MaskedAPIKey
- 修复测试用例中的 ConfigPath 问题
- 实现 API Key 脱敏逻辑

### Phase 3: 输出模块 (internal/output)
- Status: complete (183b363)
- 实现 Print 和 PrintError 函数
- 支持 pretty 模式格式化 JSON

### Phase 4: 路由模块 (internal/router)
- Status: complete (eb13b86)
- 实现 Parse、RouteConfig、Route、Field 结构体
- 添加 gopkg.in/yaml.v3 依赖

### Phase 5: API 客户端模块 (internal/api)
- Status: complete (2b45c8b)
- 实现 Client、Do、NewClient
- 处理 401/422 错误，支持 ValidationError

### Phase 7: 路由 YAML文件 (agent.yaml)
- Status: complete (0aa4a2)
- **Description**: 创建智能体路由配置 agent.yaml

- 处理 API 错误

### Phase 7: 路由 YAML 文件
- Status: complete (0aa4a2)
- 创建 cmd/routes/agent.yaml
- 包含 list, get, create, update, delete 路由

### Phase 8: Config 命令 (cmd/config.go)
- Status: complete (4688c89)
- 实现 config init/set/show 子命令
- 添加 cmd/config_test.go 测试覆盖: 配置读写、key 验证、脱敏、文件权限

### Phase 9: 根命令 (cmd/root.go)
- Status: complete (6ec5b24)
- 实现根命令，注册 config 和动态路由命令
- 添加 cmd/root_test.go 测试覆盖: 命令注册、子命令、flag、版本

### Phase 10: 主入口更新
- Status: complete (5ef54c9)
- main.go 调用 cmd.Execute()，验证编译和功能正常

### Phase 11: 集成测试与修复
- Status: complete
- 全部 20 个测试通过，go vet 无警告

### Phase 12: 最终验证
- Status: complete
- 完整构建通过
- 验收标准全部满足:
  - ckjr config init 交互式配置
  - ckjr config show api_key 脱敏
  - ckjr agent list --template 参数模板
  - --pretty 全局格式化
  - 所有测试通过

## 2026-03-25 API Client 错误处理改进

### Phase 13: 新增 ResponseError 类型
- Status: complete (3bf29bc)
- 新增 ResponseError 结构体（StatusCode, ContentType, Body, Message）
- 实现 Error() 和 Detail() 方法
- 新增 IsResponseError 辅助函数
- 新增 isJSONContentType、truncate 辅助函数
- 3 个测试覆盖：Error()、Detail()、errors.As 兼容性

### Phase 14: 重构 Do() 响应处理流程
- Status: complete (41886b3)
- 重构响应处理顺序：先读取 body -> 检查状态码+CT -> 2xx 非 JSON -> JSON 解码 -> 业务错误
- 非 2xx + 非 JSON 返回 ResponseError（含友好提示）
- 2xx + 非 JSON 返回 ResponseError（提示配置错误）
- 空 Content-Type 允许 JSON 解码尝试
- 修复原有测试 handler 未设置 Content-Type 的问题
- 4 个新测试：HTMLResponse、Non2xxWithHTML、Non2xxWithJSON、EmptyContentType

### Phase 15: 添加 --verbose 全局 flag
- Status: complete (c5ed6d3)
- 在 root.go init() 中注册 --verbose PersistentFlag
- 1 个新测试覆盖

### Phase 16: handleAPIError 增加 verbose 支持
- Status: complete (54832c7)
- 拆分 handleAPIError 为 handleAPIError + handleAPIErrorTo（可测试）
- handleAPIErrorTo 接受 io.Writer 和 verbose 参数
- ResponseError 在非 verbose 模式输出友好消息，verbose 模式追加 Detail()
- 2 个新测试覆盖

### Phase 17: 验收测试
- Status: complete
- 全部 29 个测试通过（-count=1）
- 编译成功
- --verbose flag 已在 help 输出中显示

## 2026-03-25 Request Logging

### Phase 18: logging 包 - requestId 生成与 context 透传
- Status: complete (bba2c93)
- 创建 internal/logging/logging.go
- NewRequestID 生成 UUID v4（crypto/rand）
- WithRequestID/RequestIDFrom context 透传
- 4 个测试覆盖

### Phase 19: logging 包 - Init 和 multiHandler
- Status: complete (8ef281a)
- 创建 internal/logging/multi_handler.go 实现 slog.Handler 接口
- multiHandler 同时写入多个 handler
- Init(verbose, baseDir) 创建日志目录和按日期滚动的 JSON 日志文件
- verbose 模式通过 multiHandler 同时写文件和 stderr
- 6 个新测试覆盖（3 multiHandler + 3 Init）

### Phase 20: api.Client 新增 DoCtx 方法
- Status: complete (1d5237c)
- 新增 DoCtx(ctx, method, path, body, result) 方法
- Do() 委托给 DoCtx(context.Background(), ...) 保持向后兼容
- http.NewRequest 改为 http.NewRequestWithContext
- 每次请求记录 api_request/api_response 结构化日志（request_id, method, url, status, duration_ms）
- 所有错误路径均有 ERROR 级别日志
- 4 个新测试覆盖

### Phase 21: cmdgen 集成 - 生成 requestId 并调用 DoCtx
- Status: complete (b3653b5)
- buildSubCommand.Run 中生成 requestId 并构建 context
- client.Do() 替换为 client.DoCtx(ctx, ...)
- 1 个集成测试验证日志中出现 UUID v4 requestId

### Phase 22: cmd/root.go 初始化日志 + 端到端验证
- Status: complete (c0000c2)
- cobra.OnInitialize 中调用 logging.Init
- 日志文件写入 ~/.ckjr/logs/YYYY-MM-DD.log
- 端到端验证：日志文件创建、结构化 JSON 格式、requestId 关联

## 2026-03-25 ckjr-agent Skill 实现

### Phase 23: 创建 Skill 文件
- Status: complete
- 创建 skills/ckjr-agent/SKILL.md
- 包含 YAML frontmatter (name, description, triggers, allowed-tools)
- 包含完整命令参考文档

### Phase 24: 更新 README.md
- Status: complete (abd06c3)
- 在 README.md 末尾添加 Claude Code Skill 安装章节
- 包含安装二进制、安装 Skill、使用示例

## 2026-03-25 私有仓库安装分发方案

### Phase 25: 创建 GitHub Actions Release 流水线
- Status: complete
- 创建 .github/workflows/release.yml
- 支持 linux/darwin/windows 多平台构建
- 支持 amd64/arm64 架构
- 推送 tag 时自动触发 Release

### Phase 26: 创建 install.sh 一键安装脚本
- Status: complete
- 创建 install.sh 并添加执行权限
- 自动检测操作系统和架构
- 支持 go install 和下载预编译二进制两种方式
- 支持 PAT 和 SSH 认证
- 自动配置 PATH 环境变量

### Phase 27: 创建 Skills 安装说明
- Status: complete
- 创建 skills/ckjr-agent/README.md
- 包含本地文件安装和远程 URL 安装两种方式
- 包含使用说明和可用命令列表

### Phase 28: 更新 README.md 添加私有仓库安装指南
- Status: complete
- 更新安装章节: 一键安装脚本、go install、源码构建
- 添加 Fork 自定义说明
- 更新 Skill 安装章节: 本地文件和远程 URL 两种方式

### Phase 29: 提交变更
- Status: complete (0836c05)
- 全部测试通过 (39 个测试)
- 提交 feat: add private repo install and distribution support

## 2026-03-26 Field Type/Example 字段扩展

### Phase 30: Field 结构体增加 Type/Example 字段
- Status: complete (12062f1)
- Field 结构体增加 Type (string, yaml:"type") 和 Example (string, yaml:"example") 两个字段
- 添加 TestParseRouteConfig_TypeAndExample 测试验证 YAML 解析正确
- 未设置 type/example 的字段保持零值

### Phase 31: printTemplate 输出 type 和 example
- Status: complete (3777fb4)
- 拆分 printTemplate 为 printTemplate (写 stdout) + printTemplateTo (接受 io.Writer，方便测试)
- 输出增加 type 字段，未设置时默认 "string"
- 输出增加 example 字段，仅在有值时输出（条件输出）
- 添加 TestPrintTemplate_TypeAndExample 测试覆盖

### Phase 32: 更新 agent.yaml 为数值型参数补充 type
- Status: complete (7eb6870)
- list 路由: page, limit, enablePagination, platType 补充 type: int
- create 路由: botType, isSaleOnly, promptType 补充 type: int
- 全量测试通过 (42 个测试，无回归)

## 2026-03-26 CLI 重命名 (ckjr -> ckjr-cli)

### Phase 33: 更新测试断言
- Status: complete
- cmd/root_test.go: Use 字段断言从 "ckjr" 改为 "ckjr-cli"

### Phase 34: 更新 cobra 命令定义
- Status: complete
- cmd/root.go: Use 改为 "ckjr-cli"，Short 改为"创客匠人 CLI - 知识付费 SaaS 系统的命令行工具"
- cmd/root.go: createClient 错误提示改为 "ckjr-cli config init"

### Phase 35: 更新 config.go 错误提示
- Status: complete
- cmd/config.go: runConfigShow 错误提示改为 "ckjr-cli config init"

### Phase 36: 重命名入口目录
- Status: complete
- 创建 cmd/ckjr-cli/main.go（内容与 cmd/ckjr/main.go 一致）
- 需手动执行: git rm -r cmd/ckjr

### Phase 37: 更新构建与发布配置
- Status: complete
- .github/workflows/release.yml: BINARY_NAME=ckjr-cli, 构建路径改为 ./cmd/ckjr-cli, dist 目录名改为 ckjr-cli_
- install.sh: BINARY_NAME="ckjr-cli", go install 路径改为 cmd/ckjr-cli@latest

### Phase 38: 更新技能文件
- Status: complete
- skills/ckjr-agent/SKILL.md: 所有 ckjr 命令引用改为 ckjr-cli（约 20 处），go install 路径修复
- skills/ckjr-agent/README.md: 命令表格和描述中 ckjr 改为 ckjr-cli

### Phase 39: 更新项目 README.md
- Status: complete
- README.md: 项目描述改为"创客匠人 CLI"，所有命令引用改为 ckjr-cli（约 25 处）
- go install 路径和源码构建路径已更新

### Phase 40: 最终验证
- Status: pending (需手动执行)

## 2026-03-26 Request Body 日志 & Route 命令隐藏

### Phase 49: DoCtx 增加 request_body 日志
- Status: complete (pending commit)
- 将 json.Marshal(body) 提前到 api_request 日志之前执行
- api_request 日志新增 request_body 字段
- nil body 时输出空字符串（string(nil) == ""）
- 新增 TestDoCtx_LogsRequestBody、TestDoCtx_NilBody_LogsEmptyRequestBody 测试

### Phase 50: DoCtx 增加 response_body 日志
- Status: complete (pending commit)
- 在 7 个 api_response 日志点增加 response_body 字段
- 排除网络错误（无响应体）和读取失败（无响应体）两个日志点
- 新增 TestDoCtx_LogsResponseBody、TestDoCtx_LogsResponseBody_OnError 测试

### Phase 51: 隐藏 route 命令
- Status: complete (pending commit)
- 在 routeCmd 定义中设置 Hidden: true
- 新增 TestRouteCmd_IsHidden 测试

## 2026-03-26 Workflow YAML 实现

### Phase 52: workflow 包 - 数据结构与 Parse
- Status: complete (82d7fb8)
- 创建 internal/workflow 包
- 实现 Input/Step/Workflow/Config 数据结构
- 实现 Parse 函数解析 YAML
- 测试覆盖：InvalidYAML、ValidWorkflow、EmptyWorkflows

### Phase 53: Describe 函数
- Status: complete (4a8e82d)
- 实现 Describe 函数，输出 AI 可读的 workflow 文本描述
- 输出包含：Workflow 名称、Description、Inputs（必填/可选）、Steps（命令/参数/输出）、Summary
- 测试覆盖：Output、NoInputs

### Phase 54: workflow YAML 文件
- Status: complete (97dad85)
- 创建 cmd/workflows/agent.yaml
- 包含 create-agent 工作流（5 个 inputs: name/desc/avatar/instructions/greeting，3 个 steps: create/update/get-link）
- 测试覆盖：TestParse_AgentWorkflowFile

### Phase 55: workflow 命令 (list + describe)
- Status: complete (c4adea9)
- 创建 cmd/workflow.go 实现 workflow list/describe 子命令
- 使用 //go:embed 嵌入 workflows 目录
- 在 cmd/root.go 注册 workflow 命令
- 测试覆盖：TestWorkflowList、TestWorkflowDescribe、TestWorkflowDescribe_NotFound
- 全项目测试通过

### Phase 56: 更新 SKILL.md
- Status: complete (3914de0)
- 在 skills/ckjr-cli/SKILL.md 添加 workflow 优先策略
- 指导 AI 优先使用 workflow list/describe 发现多步骤任务
- 构建验证通过

### Phase 57: 安装并端到端验证
- Status: complete (final)
- go install 安装成功
- workflow list/describe/--help 验证通过
- 原有命令（agent/common）正常工作
- 全量测试通过（62 个测试，无回归）

## 2026-03-26 Routes Resource to Name

### Phase 58: 修改 RouteConfig 结构体
- Status: complete (4a801b5)
- internal/router/router.go: Resource → Name 字段重命名

### Phase 59: 更新 cmdgen 代码
- Status: complete (3a22b41)
- internal/cmdgen/cmdgen.go: cfg.Resource → cfg.Name

### Phase 60: 更新 yamlgen 代码
- Status: complete (1143330)
- internal/yamlgen/generate.go: CreateFile 参数名调整

### Phase 61: 更新 route 命令 CLI 参数
- Status: complete (955aab3)
- cmd/route.go: --resource-desc → --name-desc，移除 --resource 参数

### Phase 62: 更新 YAML 文件
- Status: complete (3d525ae)
- cmd/routes/common.yaml, agent.yaml: resource → name 字段

### Phase 63: 更新测试文件
- Status: complete (89daa09)
- router_test.go, route_test.go, generate_test.go, cmdgen_test.go: 断言更新

### Phase 64: 验证和集成测试
- Status: complete (final)
- 全量测试通过（66 个测试，无回归）
- 验收标准全部满足:
  - go run . agent describe 正常显示
  - route import --name-desc 正常工作
  - 生成的 YAML 文件使用 name 字段
  - 代码中无 cfg.Resource 引用

## 2026-03-26 Wiki 技术文档体系

### Phase 65-67: 创建 HOME.md / install.md / quickstart.md
- Status: complete
- 创建 wiki/HOME.md: 项目简介、学习路径图、文档目录表格
- 创建 wiki/install.md: 前置条件、三种安装方式（一键脚本/go install/源码构建）、常见问题
- 创建 wiki/quickstart.md: 配置初始化、第一个 API 调用（agent 模块）、全局选项、日志说明

### Phase 68-70: 创建 core-concepts.md / project-structure.md / extending.md
- Status: complete
- 创建 wiki/core-concepts.md: YAML 路由配置、模板系统、API 客户端、日志系统、Workflow YAML
- 创建 wiki/project-structure.md: 顶层目录、cmd/ 和 internal/ 详解、数据流图
- 创建 wiki/extending.md: 手写 YAML、curl 导入、编译验证、CI/CD 发布流程

### Phase 71-72: 创建 cli-skill.md / 验证文档完整性
- Status: complete
- 创建 wiki/cli-skill.md: Skill 介绍、安装方式、自发现机制、Workflow 优先策略
- 验证全部 13 个相对链接有效
- 验证关键命令可执行（--version, config --help, agent list --template, workflow list --help）

## 2026-03-26 Log Environment Modes

### Phase 73: logging 包新增 Environment 类型和辅助函数
- Status: complete (752d87c)
- 新增 Environment int 类型（Production=0, Development=1）
- 新增 ParseEnvironment() 函数，支持 "development"/"dev" 解析为 Development，其他默认 Production
- 新增 IsDev() 函数，返回 currentEnv == Development
- 新增 currentEnv 包级变量，默认 Production
- 10 个 ParseEnvironment 表驱动测试 + 1 个 IsDev 默认值测试

### Phase 74: 更新 Init 签名，支持环境感知的日志级别
- Status: complete (8e93967)
- Init 签名新增 env Environment 参数
- Development 模式设置 slog.LevelDebug，Production 模式设置 slog.LevelInfo
- Init 设置 currentEnv = env
- 更新 3 个现有测试的 Init 调用签名
- 新增 TestInit_DevLogLevel、TestInit_ProdLogLevel、TestIsDev_AfterInit 3 个测试

### Phase 75: api.Client 条件记录 request/response body
- Status: complete (a5316d2)
- DoCtx 的 api_request 日志：request_body 仅在 Development 模式记录
- DoCtx 的 7 处 api_response 日志：response_body 仅在 Development 模式记录
- 改用 []interface{} attrs 收集属性 + IsDev() 条件追加 body
- 更新 5 个现有 body 测试设置 Development 环境
- 新增 TestDoCtx_ProdOmitsBody 测试验证 Production 省略 body
- cmd/root.go logging.Init 临时传 Production（Task 4 正式修复）

### Phase 76: cmd/root.go 接入 Environment ldflags 变量
- Status: complete (d015f38)
- 新增 Environment = "production" 变量（可通过 ldflags 注入）
- initLogging 使用 logging.ParseEnvironment(Environment) 解析后传给 Init
- 编译验证通过

### Phase 77: 更新 CI release 构建注入 Environment
- Status: complete (a508f24)
- release.yml go build ldflags 新增 -X main.Environment=production

## 2026-03-26 Version Flag ldflags 注入修复

### Phase 78: 重构 cmd 包的 Version/Environment 为私有变量 + setter
- Status: complete (76eb962)
- cmd/root.go: Version/Environment 从导出变量改为私有变量 version/environment
- 新增 SetVersion(v) 和 SetEnvironment(e) setter 方法
- rootCmd.Version 初始化改为引用私有变量 version
- initLogging 中 Environment 引用改为 environment
- 新增 TestSetVersion、TestSetEnvironment 测试（先写失败测试，再实现）

### Phase 79: 在 main 包定义 Version/Environment 供 ldflags 注入
- Status: complete (00b5294)
- cmd/ckjr-cli/main.go 新增 Version/Environment 变量（默认 "dev"/"production"）
- init() 调用 cmd.SetVersion/cmd.SetEnvironment 传递给 cmd 包
- 验证: 不带 ldflags --version 输出 "dev"，带 ldflags -X main.Version=v9.9.9 输出 "v9.9.9"

### Phase 80: 补充版本默认值测试
- Status: complete (d1eb851)
- 新增 TestDefaultVersion 验证 version 默认值 "dev"
- 新增 TestDefaultEnvironment 验证 environment 默认值 "production"

## 2026-03-27 YAML 配置文件迁移到 config/

### Phase 81: 创建 internal/config/yaml 包
- Status: complete (c1cff6f)
- 创建 internal/config/yaml 包
- New() 接受 fs.FS，提供 LoadRoutes()/LoadWorkflows() 方法
- 5 个测试：routes 加载、空目录、不存在目录、workflows 加载、workflows 不存在目录

### Phase 82: 迁移 YAML 文件 + 创建 embed.go
- Status: complete (2d0060c)
- 复制 cmd/routes/*.yaml 和 cmd/workflows/*.yaml 到 cmd/ckjr-cli/config/
- 创建 cmd/ckjr-cli/embed.go（go:embed all:config）
- 由于 go:embed 限制，config 目录放在 cmd/ckjr-cli/ 下（非根目录）

### Phase 83: 修改 cmd/ 包使用 yaml.FS
- Status: complete (3a52918)
- cmd/root.go: 移除 //go:embed routes 和 routesFS，新增 yamlFS + SetYAMLFS()
- cmd/workflow.go: 移除 //go:embed workflows 和 workflowsFS，loadAllWorkflows() 改用 yamlFS
- cmd/ckjr-cli/main.go: init() 调用 cmd.SetYAMLFS(configyaml.New(configFS))
- cmd/embed_test.go: 测试辅助，使用 //go:embed + fs.Sub 初始化 yamlFS
- registerRouteCommands() 从 init() 移到 Execute()，解决测试初始化顺序问题
- 修复 TestWorkflowDescribe 预存断言 bug
- 全部 67 个测试通过

### Phase 84: 更新测试和文档路径
- Status: complete (d134455)
- internal/workflow/workflow_test.go: 路径更新为 cmd/ckjr-cli/config/workflows/agent.yaml
- wiki/core-concepts.md: cmd/routes/ -> cmd/ckjr-cli/config/routes/，cmd/workflows/ -> cmd/ckjr-cli/config/workflows/
- wiki/extending.md: cmd/routes/ -> cmd/ckjr-cli/config/routes/（4 处）
- wiki/project-structure.md: 更新目录结构和数据流描述

### Phase 85: 清理旧文件
- Status: complete (606c51c)
- 删除 cmd/routes/agent.yaml, cmd/routes/common.yaml, cmd/workflows/agent.yaml
- 删除空目录 cmd/routes/, cmd/workflows/
- 全量测试通过

## 2026-03-27 cmd 目录结构重组

### Phase 86: 更新 internal/config/yaml 加载路径
- Status: complete (c0a826f)
- yaml.go: LoadRoutes 路径从 config/routes 改为 routes，LoadWorkflows 从 config/workflows 改为 workflows
- yaml_test.go: MapFS key 从 config/routes/... 改为 routes/...

### Phase 87: 迁移 YAML 物理文件 + 更新 embed 指令
- Status: complete (6a6f9aa)
- 移动 cmd/ckjr-cli/config/routes/*.yaml 到 cmd/ckjr-cli/routes/
- 移动 cmd/ckjr-cli/config/workflows/*.yaml 到 cmd/ckjr-cli/workflows/
- embed.go: go:embed all:config 改为 go:embed all:routes all:workflows
- embed_test.go: go:embed all:ckjr-cli/config 改为 go:embed all:ckjr-cli/routes all:ckjr-cli/workflows
- 删除空 cmd/ckjr-cli/config/ 目录

### Phase 88: 更新 workflow_test.go 路径 + wiki 文档
- Status: complete (2aec040)
- internal/workflow/workflow_test.go: 路径更新为 cmd/ckjr-cli/workflows/agent.yaml
- wiki 文档: 所有 cmd/ckjr-cli/config/routes/ 和 config/workflows/ 路径更新

### Phase 89: 提取辅助函数到 internal/router
- Status: complete (8ad4d58)
- 创建 internal/router/infer.go: InferRouteName, InferNameFromPath (导出函数)
- 创建 internal/router/infer_test.go: 14 个表驱动测试

### Phase 90: 创建 cmd/config/ 子包
- Status: complete (f55d920)
- cmd/config/config.go: NewCommand() 工厂函数，包含 init/set/show 子命令
- cmd/config/config_test.go: 6 个测试（使用 internalconfig 别名避免包名冲突）

### Phase 91: 创建 cmd/route/ 子包
- Status: complete (cb9c866)
- cmd/route/route.go: NewCommand() 工厂函数，使用 router.InferRouteName
- cmd/route/route_test.go: 3 个测试

### Phase 92: 创建 cmd/workflow/ 子包
- Status: complete (c7a3663)
- cmd/workflow/workflow.go: NewCommand(yamlFS) 工厂函数
- cmd/workflow/workflow_test.go: 3 个测试（使用 MapFS mock）

### Phase 93: 重构 cmd/root.go 集成子包
- Status: complete (f3666d7)
- root.go: 移除 configCmd/routeCmd/workflowCmd 引用，改为 import 子包
- init() 注册 configcmd.NewCommand() 和 routecmd.NewCommand()
- Execute() 注册 workflowcmd.NewCommand(yamlFS)
- embed_test.go: TestMain 中显式注册 workflow 子命令

### Phase 94: 删除旧文件
- Status: complete (6b6f353)
- 删除 cmd/config.go, config_test.go, route.go, route_test.go, workflow.go, workflow_test.go (6 个文件)

### Phase 95: 最终验证
- Status: complete (final)
- 全量测试: 81 tests, 15 packages, ALL PASS
- go vet: 无警告
- go build: 编译成功
- 目录结构: 符合计划预期

## 2026-03-27 本地多平台构建与 GitHub Release 发布

### Task 1: 创建 Makefile 基础框架
- Status: complete (818678b)
- 创建 Makefile，包含 BINARY_NAME/BUILD_DIR/CMD_PATH/VERSION/LDFLAGS 等变量
- 实现 version 目标（输出 git tag）和 clean 目标（清理 bin/）
- 验证: make version 输出 v0.0.1，make clean 无错误

### Task 2: 添加 build-local 目标
- Status: complete (918fefd)
- 添加 build-local 目标，编译当前平台二进制到 bin/ckjr-cli
- 验证: 构建成功，bin/ckjr-cli --help 正常输出

### Task 3: 添加 build 目标（多平台交叉编译）
- Status: complete (9521072)
- 添加 build 目标支持 5 平台: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Windows 使用 zip 打包，其他使用 tar.gz
- 验证: 5 个压缩包生成，文件名兼容 install.sh 正则，解压后二进制可执行

### Task 4: 添加前置检查和 release 目标
- Status: complete (9ec4cc2)
- 添加 check-gh（gh CLI 认证）、check-clean（工作区干净）、check-github-remote（remote 已配置）
- 添加 release 目标: check-gh + check-clean + check-github-remote + tag + push + build + gh release create
- 验证: check-gh/check-github-remote 防护正常（未登录/未配置时报错退出）

### Task 5: 配置 GitHub remote 并端到端验证
- Status: complete (no commit)
- 添加 git remote github -> git@github.com:childelins/ckjr-cli.git
- check-github-remote 验证通过
- 端到端 make build VERSION=v0.0.3-test: 5 个平台全部成功

## 2026-03-27 install.sh 简化

### Phase 96: 简化 install.sh
- Status: complete (3ccd4ab)
- 删除 has_go() 函数（环境检测）
- 删除 install_via_go() 函数（Go 安装方式，含 GOPRIVATE/认证逻辑）
- 简化 main() 函数，移除交互式选择，直接调用 install_via_release
- bash -n 语法验证通过，无 go install 残留

### Phase 97: 更新 wiki/install.md
- Status: complete (4d37554)
- 修复 curl URL 分支名 main -> master
- 移除安装位置中的 go install 行
- 删除"方式二：go install"整节
- "方式三"改为"方式二"
- 简化常见问题表：移除 go install 失败行，简化 PATH 和 Release 描述

### Phase 98: 更新 README.md
- Status: complete (d128ba7)
- 修复 curl URL 分支名 main -> master
- 移除安装方式描述中的 go install 引用

### Phase 99: 最终验证
- Status: complete (final)
- grep -rn 'go install' 三个文件：无匹配
- bash -n install.sh：语法正确
- curl URL 一致性：README.md 和 wiki/install.md 均使用 master 分支
