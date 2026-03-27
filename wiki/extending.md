# 扩展开发指南

本文档介绍如何为 ckjr-cli 添加新的 API 模块。

## 方式一：手写 YAML 路由配置

### YAML 结构

```yaml
name: example                    # 资源名（CLI 子命令名，不可重名）
description: 示例资源             # 资源描述
routes:
    list:                        # 路由名（CLI 子子命令名）
        method: POST             # HTTP 方法
        path: /admin/example/list  # API 路径
        description: 获取列表
        template:                # 参数模板（可选）
            page:
                description: 页码
                required: false
                default: 1
                type: int
            name:
                description: 名称
                required: false
    create:
        method: POST
        path: /admin/example/create
        description: 创建资源
        template:
            name:
                description: 名称
                required: true
            desc:
                description: 描述
                required: false
```

### 真实示例

以下是 `cmd/ckjr-cli/routes/agent.yaml` 中 create 路由的完整定义：

```yaml
name: agent
description: AI智能体管理
routes:
    create:
        method: POST
        path: /admin/aiCreationCenter/createApp
        description: 创建智能体
        template:
            avatar:
                description: 头像URL
                required: true
            botType:
                description: 智能体类型, 99-自营智能体 100-Coze智能体
                required: false
                default: 99
                type: int
            desc:
                description: 描述
                required: true
            isSaleOnly:
                description: 是否支持售卖, 1-交付型 0-客服型
                required: false
                default: 1
                type: int
            name:
                description: 智能体名称
                required: true
            promptType:
                description: 提示词模板类型, 1-交付型 3-角色类/客服型
                required: false
                default: 3
                type: int
```

### 步骤

1. 在 `cmd/ckjr-cli/routes/` 目录下创建新的 YAML 文件（如 `example.yaml`）
2. 本地编译并测试（见下方编译验证）
3. `name` 不能与已有资源重名

## 方式二：从 curl 命令导入

适用于快速将现有 API 请求转换为 YAML 配置。

工作原理：

```
curl 命令 -> curlparse.Parse() -> yamlgen.GenerateRoute() -> 写入 YAML 文件
```

### 使用示例

新建文件：

```bash
ckjr-cli route import \
  --curl 'curl -X POST https://api.example.com/admin/example/create -d '"'"'{"name":"test"}'"'"'' \
  --file cmd/ckjr-cli/routes/example.yaml \
  --name-desc "示例资源管理"
```

追加到已有文件：

```bash
ckjr-cli route import \
  --curl 'curl -X POST https://api.example.com/admin/example/update -d '"'"'{"id":"1"}'"'"'' \
  --file cmd/ckjr-cli/routes/example.yaml \
  --name update
```

stdin 管道输入：

```bash
echo 'curl -X POST https://api.example.com/admin/example/list -d '"'"'{"page":1}'"'"'' \
  | ckjr-cli route import --file cmd/ckjr-cli/routes/example.yaml --name-desc "示例"
```

### 参数说明

| 参数 | 说明 |
|------|------|
| `--curl` | curl 命令字符串（可从 stdin 管道输入） |
| `--file` / `-f` | 目标 YAML 文件路径 |
| `--name` / `-n` | 路由名，默认从 URL 路径推导 |
| `--name-desc` | 资源描述，新建文件时必需 |

## 编译验证

本地测试新增的模块：

```bash
# 编译
go build -o ckjr-cli ./cmd/ckjr-cli

# 查看参数模板
./ckjr-cli example list --template

# 执行请求
./ckjr-cli example list --pretty
```

## 发布流程

### 推送 tag 触发自动发布

```bash
git tag v1.x.x
git push origin v1.x.x
```

CI/CD 流程（`.github/workflows/release.yml`）：
1. GitHub Actions 自动构建 linux/darwin/windows x amd64/arm64
2. 创建 GitHub Release 并上传二进制文件
3. 用户通过 `install.sh` 或直接下载使用

---

[上一步：项目结构详解](project-structure.md) | 下一步：[Claude Code Skill 集成](cli-skill.md)
