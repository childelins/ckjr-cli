# Wiki 技术文档体系 设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

为 ckjr-cli 项目建立 wiki 目录下的分步技术文档体系，引导新人从零开始理解和使用项目。文档覆盖安装配置、快速上手、核心架构理解、扩展开发全流程。

## 目标受众

- 刚接触项目的新团队成员
- 需要了解 ckjr-cli 架构的开发者
- 想要扩展新 API 资源的贡献者

## 文档清单

### 1. HOME.md -- Wiki 首页和导航

定位：wiki 入口，提供完整文档目录和学习路径。

内容要点：
- 项目一句话简介
- 学习路径图（按顺序阅读的建议）
- 文档目录（带简要描述和链接）

### 2. install.md -- 安装指南

定位：详细的安装步骤，覆盖不同场景和常见问题。

内容要点：
- 前置条件（操作系统、Go 环境 - 可选）
- 三种安装方式详解：
  - 一键安装脚本（推荐非开发者）
  - go install（Go 开发者）
  - 从源码构建（贡献者）
- 私有仓库认证配置（GITHUB_TOKEN / SSH）
- 验证安装成功
- Fork 自定义安装
- 常见安装问题排查

### 3. quickstart.md -- 快速开始

定位：5 分钟内让新人完成第一次 API 调用。

内容要点：
- 第一步：初始化配置 (`config init`)
- 第二步：查看配置 (`config show`)
- 第三步：使用智能体命令（list / get / create）
- 第四步：体验 stdin 管道输入
- 全局选项说明（--pretty, --verbose）
- 请求日志查看

### 4. core-concepts.md -- 核心概念

定位：解释项目的关键设计思想和术语。

内容要点：
- YAML 路由配置：什么是路由 YAML，为什么用它而不是手写 Cobra 命令
- 资源（Resource）与命令的映射关系：YAML 中的 name 字段 -> CLI 子命令 -> HTTP API
- API 客户端：统一认证、错误处理、响应格式（Dingo API Response）
- 请求日志：requestId 追踪机制
- 模板系统：--template 查看参数结构、默认值、必填校验
- Workflow YAML：多步骤工作流的概念（可选，高级）

### 5. project-structure.md -- 项目结构详解

定位：逐文件/目录解释项目组织，让新人知道东西在哪。

内容要点：
- 顶层目录说明（cmd/, internal/, docs/, skills/, .github/）
- cmd/ 详解：root.go 入口逻辑、config.go/workflow.go/route.go 各自职责
- cmd/routes/ 和 cmd/workflows/：embed 嵌入的 YAML 配置文件
- internal/ 详解：
  - router/ - YAML 解析
  - cmdgen/ - 命令生成
  - api/ - HTTP 客户端
  - config/ - 配置管理
  - logging/ - 日志系统
  - output/ - 输出格式化
  - curlparse/ - curl 命令解析
  - yamlgen/ - YAML 生成
  - workflow/ - 工作流解析
- 数据流图：从 YAML 到 CLI 命令到 API 请求的完整路径

### 6. extending.md -- 扩展开发指南

定位：教新人如何为项目添加新的 API 资源。

内容要点：
- 场景：后端新增了一个 API，需要 CLI 支持
- 方式一：手写 YAML 路由配置
  - YAML 结构详解（name, description, routes, method, path, template）
  - template 字段说明（description, required, default, type, example）
  - 完整示例
- 方式二：从 curl 命令导入（route import）
  - curl 导入的工作原理（curlparse + yamlgen）
  - 命令用法和示例
- 重新编译和测试
- CI/CD 发布流程（tag -> GitHub Actions -> Release）

### 7. cli-skill.md -- Claude Code Skill 集成

定位：介绍 AI Agent 集成能力。

内容要点：
- 什么是 ckjr-cli Skill
- 安装方式
- 自描述发现机制
- 新增模块无需修改 Skill 的设计

## 文件结构

```
wiki/
  HOME.md              # 首页和导航
  install.md           # 安装指南
  quickstart.md        # 快速开始
  core-concepts.md     # 核心概念
  project-structure.md # 项目结构详解
  extending.md         # 扩展开发指南
  cli-skill.md         # Claude Code Skill 集成
```

## 文档间依赖关系

```
HOME.md
  -> install.md
    -> quickstart.md
      -> core-concepts.md
        -> project-structure.md
          -> extending.md
            -> cli-skill.md
```

## 与现有文档的关系

- `README.md`：保持不变，作为项目根目录的概览。wiki 文档是对 README 的细化和教学化展开
- `docs/experiences/`：已有的经验文档，wiki 中通过链接引用
- `docs/superpowers/`：AI 开发流程文档，不在 wiki 学习路径中

## 实现注意事项

1. wiki 目录放在项目根目录，与 README.md 同级
2. 所有文档使用中文，代码标识符保持英文
3. 每份文档控制在 200-300 行以内
4. 代码示例从实际项目代码中提取，确保可运行
5. 文档中引用代码使用相对路径（从项目根目录出发）
6. 链接使用相对路径，确保离线可访问

## 测试策略

1. 确认所有文档间链接有效
2. 文档中的命令示例可实际执行
3. 由未接触过项目的人阅读并反馈
