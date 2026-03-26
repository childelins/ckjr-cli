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
