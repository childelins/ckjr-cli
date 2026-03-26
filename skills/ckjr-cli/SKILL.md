---
name: ckjr-cli
description: 创客匠人 SaaS 平台 CLI，管理智能体、订单等业务模块
triggers:
  - command: /ckjr-cli
  - intent: 创客匠人、智能体、SaaS平台操作、ckjr
allowed-tools:
  - Bash
---

# ckjr-cli Skill

创客匠人 SaaS 平台命令行工具。通过 ckjr-cli 管理平台业务模块。

## 前置条件

1. 安装 CLI:
   ```bash
   go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
   ```

2. 初始化配置:
   ```bash
   ckjr-cli config init
   ```
   按提示设置 API 地址和 API Key。

## 命令发现

CLI 支持自描述，按以下步骤发现可用命令和参数:

1. **查看所有模块**: `ckjr-cli --help`
2. **查看模块子命令**: `ckjr-cli <module> --help`
3. **查看命令参数**: `ckjr-cli <module> <command> --template`

`--template` 输出 JSON 格式的参数结构，包含字段名、描述、类型、是否必填、默认值。

## 任务执行策略

对于多步骤任务（如创建智能体、配置智能体等），优先使用 workflow：

1. **匹配工作流**: `ckjr-cli workflow list` 查看是否有匹配的工作流
2. **获取流程**: `ckjr-cli workflow describe <name>` 获取完整流程定义
3. **收集信息**: 根据 workflow 的 inputs 一次性向用户收集所需信息
4. **按步执行**: 按 steps 顺序逐步执行原子命令，注意步骤间的数据传递
5. **汇报结果**: 按 summary 模板汇报执行结果

对于简单的单步操作（如查看列表、删除），直接使用命令发现流程。

## 使用规则

1. **先发现再执行**: 不确定参数时，先执行 `--template` 查看参数结构
2. **JSON 参数**: 所有命令参数使用单引号包裹的 JSON 字符串传递
   ```bash
   ckjr-cli <module> <command> '{"field1":"value1","field2":"value2"}'
   ```
3. **全局选项**:
   - `--pretty` 格式化 JSON 输出
   - `--verbose` 显示请求详情

## 错误处理

- 命令未找到 -> 执行 `--help` 确认可用命令
- 未找到配置 -> 执行 `ckjr-cli config init`
- 认证失败 -> 提示用户更新 API Key
- 参数错误 -> 执行 `--template` 检查参数结构
