# 项目结构详解

## 顶层目录

```
ckjr-cli/
  main.go              # 程序入口，调用 cmd.Execute()
  go.mod               # Go 模块定义（Go 1.24.3）
  install.sh           # 一键安装脚本
  CLAUDE.md            # AI 开发规范
  cmd/                 # CLI 命令定义和入口
  internal/            # 内部库，不对外暴露
  docs/                # 开发文档
  skills/              # Skills（共享源，安装时自动分发到各平台）
  .github/             # CI/CD 配置
```

## cmd/ 目录详解

```
cmd/
  root.go              # 根命令，注册全局 flag，集成子包
  root_test.go         # 根命令测试
  embed_test.go        # 测试辅助，嵌入 YAML 配置并初始化 yamlFS
  config/
    config.go          # config init/set/show 子命令
    config_test.go
  route/
    route.go           # route import 命令（隐藏命令）
    route_test.go
  workflow/
    workflow.go        # workflow list/describe 子命令，init 隐藏命令
    workflow_test.go
  ckjr-cli/
    main.go            # 独立的 main 包，供 go install 使用
    embed.go           # go:embed all:routes all:workflows，嵌入 YAML 配置
    routes/
      agent.yaml       # 智能体模块（create/delete/get/list/update）
      common.yaml      # 公共接口模块（link）
    workflows/
      agent.yaml       # 智能体工作流（create-agent）
```

`cmd/root.go` 的核心逻辑：

```go
var yamlFS *configyaml.FS

func init() {
    rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
    rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
    cobra.OnInitialize(initLogging)

    // 注册静态子命令（子包工厂函数）
    rootCmd.AddCommand(configcmd.NewCommand())
    rootCmd.AddCommand(routecmd.NewCommand())
}

func Execute() {
    registerRouteCommands()  // 从 yamlFS 加载并生成动态路由命令
    rootCmd.AddCommand(workflowcmd.NewCommand(yamlFS))
    rootCmd.Execute()
}
```

`registerRouteCommands()` 通过 `yamlFS.LoadRoutes()` 读取嵌入的 YAML 文件，解析后调用 `cmdgen.BuildCommand()` 生成 cobra 命令。

## internal/ 目录详解

```
internal/
  router/      # YAML 路由配置解析
  cmdgen/      # 核心：RouteConfig 转 cobra.Command
  api/         # HTTP 客户端
  config/      # 配置管理（~/.ckjr/config.json）
  logging/     # 日志系统（requestId + 文件/终端双输出）
  output/      # JSON 输出格式化
  curlparse/   # curl 命令解析器
  yamlgen/     # YAML 路由配置生成器
  workflow/    # Workflow YAML 解析和描述生成
```

各模块一行说明：

| 模块 | 职责 |
|------|------|
| `router/` | 定义 RouteConfig/Route/Field 数据结构，解析 YAML |
| `cmdgen/` | 将 RouteConfig 转换为 cobra.Command，处理参数校验和 API 调用 |
| `api/` | HTTP 客户端，Bearer Token 认证，Dingo API Response 解析 |
| `config/` | 读写 `~/.ckjr/config.json`，API Key 脱敏 |
| `logging/` | requestId 生成（UUID v4），slog 日志，按日期滚动 |
| `output/` | JSON 输出，支持 pretty 格式化 |
| `curlparse/` | 解析 curl 命令提取 method/path/body |
| `yamlgen/` | 生成 YAML 路由配置，支持新建和追加到已有文件 |
| `workflow/` | 解析 Workflow YAML，生成 AI 可读的文本描述 |

## 数据流

从用户输入到 API 调用的完整数据流：

```
YAML 路由配置 (cmd/ckjr-cli/routes/*.yaml)
    |
    v  (go:embed -> configyaml.FS -> LoadRoutes())
router.Parse() -> RouteConfig
    |
    v
cmdgen.BuildCommand() -> cobra.Command
    |
    v  (用户执行 CLI 命令)
cmdgen.buildSubCommand()
    |-- 解析 JSON 参数
    |-- applyDefaults()      应用默认值
    |-- validateRequired()   校验必填字段
    |-- api.Client.DoCtx()   发送 HTTP 请求（含 requestId）
    |-- output.Print()       输出 JSON 结果
```

关键点：YAML 文件在编译时通过 `go:embed` 嵌入二进制文件，运行时无需文件系统依赖。

---

[上一步：核心概念](core-concepts.md) | 下一步：[扩展开发指南](extending.md)
