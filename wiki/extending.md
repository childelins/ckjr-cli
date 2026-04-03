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
                min: 1           # 最小值（适用于 int/float）
                max: 1000        # 最大值（适用于 int/float）
            keyword:
                description: 搜索关键词
                required: false
                type: string
                minLength: 1     # 最小长度（适用于 string）
                maxLength: 100   # 最大长度（适用于 string）
    create:
        method: POST
        path: /admin/example/create
        description: 创建资源
        template:
            name:
                description: 名称
                required: true
                type: string
                minLength: 1
                maxLength: 50
            email:
                description: 邮箱
                required: true
                type: string
                pattern: "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"  # 正则约束
            score:
                description: 评分
                required: false
                type: float
                min: 0.0
                max: 10.0
            desc:
                description: 描述
                required: false
```

### response 字段（响应过滤）

可选配置，用于限制 API 响应的输出字段。`response` 直接使用列表格式，支持纯字符串和带描述的对象两种格式混合使用：

```yaml
    get:
        method: GET
        path: /admin/example/detail
        description: 获取详情
        template:
            id:
                description: ID
                required: true
                type: path
        response:
            - data.id                              # 纯字符串
            - path: data.status                    # 带描述的对象格式
              description: "状态, 1-上架 2-下架"
            - data.name
```

| 属性 | 类型 | 说明 |
|------|------|------|
| `response` | list | 白名单字段列表，支持纯字符串（`- data.id`）和对象格式（`- path: ... description: ...`）混合 |

带 `description` 的字段会在 `--template` 输出的 `response` 部分展示描述信息，帮助 AI 理解返回字段含义。

未配置 `response` 时全量输出。

### template 字段完整属性

| 属性 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `description` | string | 是 | 字段描述 |
| `required` | bool | 是 | 是否必填 |
| `default` | any | 否 | 默认值 |
| `type` | string | 否 | 类型：string/int/float/bool/array/path/date。`path` 用于替换 URL 路径占位符 `{xxx}`，`date` 格式为 `YYYY-MM-DD HH:MM:SS` |
| `example` | string | 否 | 示例值 |
| `min` | number | 否 | 最小值（适用于 type: int/float） |
| `max` | number | 否 | 最大值（适用于 type: int/float） |
| `minLength` | number | 否 | 最小长度（适用于 type: string） |
| `maxLength` | number | 否 | 最大长度（适用于 type: string） |
| `pattern` | string | 否 | 正则表达式（适用于 type: string） |
| `autoUpload` | string | 否 | 自动转存标记。`image` 表示外部图片 URL 在 API 请求前自动转存到素材库 |

约束与 type 不匹配时会被静默忽略（如对 string 设置 min），方便后续调整 type 时保留约束。

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
                type: string
                autoUpload: image
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

### 路径参数

当 API 路径包含动态参数时（如 `/admin/courses/{courseId}`），需要在 template 中用 `type: path` 声明路径参数：

```yaml
routes:
    update:
        method: PUT
        path: /admin/courses/{courseId}
        description: 更新课程
        template:
            courseId:
                description: 课程ID
                required: true
                type: path       # 路径参数：替换路径中的 {courseId}
            courseType:
                description: 课程类型
                required: true
                type: int
            name:
                description: 课程名称
                required: true
                type: string
```

行为规则：
- `type: path` 的字段值用于替换 `path` 中的对应占位符
- 替换后从请求 body 中移除，不会发送给服务端
- 不参与类型校验和约束校验，必填检查由替换逻辑负责
- 路径参数缺失时立即报错，阻止请求发送

多路径参数示例：

```yaml
routes:
    update-chapter:
        method: PUT
        path: /courses/{courseId}/chapters/{chapterId}
        template:
            courseId:
                description: 课程ID
                required: true
                type: path
            chapterId:
                description: 章节ID
                required: true
                type: path
            title:
                description: 章节标题
                required: true
                type: string
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

## 方式三：创建工作流骨架

快速创建模块的 workflow YAML 骨架文件：

```bash
ckjr-cli workflow init example
# 已创建: cmd/ckjr-cli/workflows/example.yaml
```

生成结果：

```yaml
name: example
description: example
workflows:
  workflow-name:
    description: 工作流描述
    triggers: []
    inputs: []
    steps: []
```

然后编辑该文件，填充具体的 triggers、inputs、steps 等内容。

## 编译验证

本地测试新增的模块：

```bash
# 编译
make build-local

# 查看参数模板
bin/ckjr-cli example list --template

# 执行请求
bin/ckjr-cli example list --pretty
```

## 发布流程

```bash
# 一键发布：tag + 构建 + 上传到 GitHub Release
make release VERSION=vX.Y.Z
```

发布流程：前置检查（gh 登录、工作区干净、github remote）-> 创建 tag -> 推送 tag -> 多平台构建 -> 创建 GitHub Release 上传二进制文件。

---

[上一步：项目结构详解](project-structure.md) | 下一步：[Claude Code Skill 集成](cli-skill.md)
