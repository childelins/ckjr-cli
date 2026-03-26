# ckjr-cli Wiki

ckjr-cli 是创客匠人知识付费 SaaS 系统的命令行工具，通过 YAML 路由配置自动生成 CLI 子命令，支持智能体管理等业务模块。

## 适合谁读

- 新加入团队的成员，需要快速了解项目
- 想了解 CLI 架构设计的开发者
- 需要扩展新 API 模块的贡献者
- 需要集成 Claude Code Skill 的 AI 开发者

## 学习路径

```
install -> quickstart -> core-concepts -> project-structure -> extending -> cli-skill
```

## 文档目录

| 文档 | 描述 | 预计时间 |
|------|------|---------|
| [安装指南](install.md) | 环境准备、三种安装方式、常见问题 | 5 min |
| [快速开始](quickstart.md) | 配置初始化、第一个 API 调用、全局选项 | 10 min |
| [核心概念](core-concepts.md) | YAML 路由配置、模板系统、API 客户端、日志系统 | 15 min |
| [项目结构详解](project-structure.md) | 目录结构、模块职责、数据流 | 10 min |
| [扩展开发指南](extending.md) | 手写 YAML、curl 导入、编译发布流程 | 10 min |
| [Claude Code Skill 集成](cli-skill.md) | Skill 安装、使用方式、自发现机制 | 5 min |

## 快速参考

```bash
# 安装后初始化
ckjr-cli config init

# 查看可用模块
ckjr-cli --help

# 查看命令参数模板
ckjr-cli agent list --template

# 执行 API 调用
ckjr-cli agent list '{"page":1,"limit":20}' --pretty
```
