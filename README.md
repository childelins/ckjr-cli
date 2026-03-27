# ckjr-cli

创客匠人 CLI - 知识付费 SaaS 系统的命令行工具。

通过 YAML 路由配置自动生成 CLI 子命令，无需手写 cobra 代码即可扩展新的 API 资源。

## 快速安装

```bash
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

更多安装方式（源码构建 / Fork 自定义）见 [安装指南](wiki/install.md)。

## 快速开始

```bash
ckjr-cli config init                    # 初始化配置
ckjr-cli agent list                     # 查询智能体列表
ckjr-cli agent get '{"aikbId":"xxx"}'  # 查询详情
```

完整使用教程见 [快速开始](wiki/quickstart.md)。

## 文档

| 文档 | 说明 |
|------|------|
| [安装指南](wiki/install.md) | 安装、认证配置、常见问题 |
| [快速开始](wiki/quickstart.md) | 5 分钟上手 |
| [核心概念](wiki/core-concepts.md) | YAML 路由、模板系统、API 客户端 |
| [项目结构](wiki/project-structure.md) | 目录详解、模块职责、数据流 |
| [扩展开发](wiki/extending.md) | 手写 YAML / curl 导入、编译发布 |
| [Claude Code Skill](wiki/cli-skill.md) | AI Skill 安装与使用 |

## 开发

```bash
make build-local          # 当前平台编译
make build VERSION=v0.1.0 # 多平台交叉编译
make test                 # 运行测试
make release VERSION=v0.1.0 # 一键发布到 GitHub Release
```
