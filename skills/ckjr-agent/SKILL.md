---
name: ckjr-agent
description: 管理公司 SaaS 平台的 AI 智能体，支持增删改查操作
triggers:
  - command: /ckjr-agent
  - intent: 智能体管理、创建智能体、查看智能体列表、AI助手操作
allowed-tools:
  - Bash
---

# ckjr-agent Skill

使用 ckjr-cli 操作公司 SaaS 平台的 AI 智能体。

## 前置条件

1. 安装 CLI：
   ```bash
   go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
   ```

2. 初始化配置：
   ```bash
   ckjr-cli config init
   ```
   按提示设置 API 地址和 API Key。

## 可用命令

### 查看帮助

```bash
ckjr-cli --help
ckjr-cli agent --help
```

### 智能体列表

```bash
# 查看所有智能体
ckjr-cli agent list

# 带筛选条件
ckjr-cli agent list '{"name":"助手","page":1,"limit":20}'

# 查看参数模板
ckjr-cli agent list --template
```

### 智能体详情

```bash
ckjr-cli agent get '{"aikbId":"xxx"}'
```

### 创建智能体

```bash
# 查看必填参数
ckjr-cli agent create --template

# 创建
ckjr-cli agent create '{"name":"销售助手","avatar":"https://...","desc":"帮助销售团队"}'
```

### 更新智能体

```bash
ckjr-cli agent update --template
ckjr-cli agent update '{"aikbId":"xxx","name":"新名称"}'
```

### 删除智能体

```bash
ckjr-cli agent delete '{"aikbId":"xxx"}'
```

## 使用规则

1. **先查看模板**: 不确定参数时，先执行 `--template` 查看参数结构
2. **JSON 格式**: 所有参数使用 JSON 格式
3. **脱敏显示**: API Key 在 `config show` 时会脱敏
4. **日志追踪**: 每次请求生成 requestId，日志在 `~/.ckjr/logs/`

## 错误处理

| 错误 | 原因 | 解决 |
|------|------|------|
| 未找到配置文件 | 未执行 config init | 执行 `ckjr-cli config init` |
| API Key 过期 | 认证失败 | 重新获取 API Key |
| 参数校验失败 | 必填字段缺失 | 使用 `--template` 检查参数 |

## 全局选项

- `--pretty`: 格式化 JSON 输出
- `--verbose`: 显示请求日志
- `--version`: 显示版本号
