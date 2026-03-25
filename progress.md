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
