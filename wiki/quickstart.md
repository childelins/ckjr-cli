# 快速开始

本指南帮助你完成配置初始化并执行第一个 API 调用。

## 配置初始化

### 交互式初始化

```bash
ckjr-cli config init
```

按提示输入 API 地址和 API Key。配置文件保存到 `~/.ckjr/config.json`。

### 手动设置

```bash
# 设置 API 地址
ckjr-cli config set base_url https://your-api.example.com

# 设置 API Key
ckjr-cli config set api_key your-api-key

# 查看配置
ckjr-cli config show
```

使用 `--pretty` 格式化输出：

```bash
ckjr-cli config show --pretty
```

## 第一个 API 调用

以智能体（agent）模块为例。

### 查看参数模板

不确定需要哪些参数时，使用 `--template` 查看：

```bash
ckjr-cli agent list --template
```

输出示例：

```json
{
  "enablePagination": {
    "description": "是否分页返回, 1-是 0-否",
    "required": false,
    "default": 0,
    "type": "int"
  },
  "limit": {
    "description": "每页数量",
    "required": false,
    "default": 10,
    "type": "int"
  },
  "name": {
    "description": "按名称搜索",
    "required": false
  },
  "page": {
    "description": "页码",
    "required": false,
    "default": 1,
    "type": "int"
  }
}
```

### 列表查询

```bash
# 使用默认参数（page=1, limit=10）
ckjr-cli agent list

# 自定义参数
ckjr-cli agent list '{"page":1,"limit":20}'

# stdin 管道输入
echo '{"page":1}' | ckjr-cli agent list -
```

### 详情查询

```bash
ckjr-cli agent get '{"aikbId":"xxx"}'
```

### 创建智能体

```bash
ckjr-cli agent create '{"name":"my-agent","avatar":"https://example.com/avatar.png","desc":"一个测试智能体"}'
```

## 全局选项

| 选项 | 说明 |
|------|------|
| `--pretty` | 格式化 JSON 输出，便于阅读 |
| `--verbose` | 显示详细调试信息（HTTP 请求/响应详情） |
| `--version` | 显示版本号 |
| `--help` | 显示帮助信息 |

## 请求日志

每次 API 调用自动生成日志，便于排查问题。

- 日志位置：`~/.ckjr/logs/YYYY-MM-DD.log`
- 每次请求生成唯一的 requestId（UUID v4）
- 使用 `--verbose` 实时在终端查看请求详情

```bash
# 查看 agent list 的详细请求过程
ckjr-cli agent list --verbose
```

---

[上一步：安装指南](install.md) | 下一步：[核心概念](core-concepts.md)
