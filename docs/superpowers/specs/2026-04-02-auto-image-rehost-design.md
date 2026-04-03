# 路由模板自动图片转存设计文档

> Created: 2026-04-02
> Status: Draft

## 概述

在路由 YAML 的 template 字段中添加 `autoUpload: image` 标记，cmdgen 在执行命令时自动检测并转存外部图片 URL 到系统素材库。消除 workflow 中重复的 upload-avatar 步骤声明，同时覆盖直接调用 CLI 命令的场景。

## 背景

当前 agent.yaml 和 course.yaml 的 4 个 workflow 中都包含完全相同的 upload-avatar 步骤：
- 调用 `asset upload-image` 转存外部图片
- 将返回的 `imageUrl` 传递给后续步骤

这种模式是声明式 workflow 的样板代码，增加了维护负担。更根本的问题是：直接通过 CLI 调用 `agent create` 或 `course create` 时，外部图片 URL 不会被自动转存。

## 设计决策

| 决策项 | 选择 |
|--------|------|
| 实现层级 | cmdgen 命令执行层 + workflow 声明层 |
| 覆盖范围 | 所有场景（workflow + 直接 CLI 调用） |
| 失败策略 | 中止流程，报告错误 |
| 标记语法 | Field 新增 `autoUpload: image` |

## 架构

### 1. 路由模板层：标记字段

在 `router.Field` 结构体中新增 `AutoUpload` 字段：

```go
// internal/router/router.go
type Field struct {
    // ... 现有字段 ...
    AutoUpload string `yaml:"autoUpload,omitempty"` // "image" 表示自动转存图片
}
```

路由 YAML 标记语法：

```yaml
# cmd/ckjr-cli/routes/agent.yaml - create 路由
template:
    avatar:
        description: 头像URL
        required: true
        type: string
        autoUpload: image        # <-- 新增标记
        minLength: 1
        maxLength: 255
```

```yaml
# cmd/ckjr-cli/routes/course.yaml - create/update 路由
template:
    courseAvatar:
        description: 课程封面
        required: true
        type: string
        autoUpload: image        # <-- 新增标记
        minLength: 1
        maxLength: 255
```

### 2. cmdgen 层：自动转存

在 `buildSubCommand` 的执行管线中，ValidateAll 之后、API 请求之前，插入转存步骤：

```
解析 JSON -> applyDefaults -> ReplacePath -> ValidateAll -> [自动转存] -> API 请求
```

新增 `processAutoUpload` 函数：

```go
// internal/cmdgen/cmdgen.go

// processAutoUpload 扫描 template 中 autoUpload=image 的字段，
// 对外部 URL 执行转存，将 input 中对应值替换为转存后的 OSS URL
func processAutoUpload(ctx context.Context, input map[string]interface{},
    template map[string]router.Field, apiClient *api.Client) error {

    for name, field := range template {
        if field.AutoUpload != "image" {
            continue
        }

        val, exists := input[name]
        if !exists {
            continue
        }

        urlStr, ok := val.(string)
        if !ok || urlStr == "" {
            continue
        }

        // 已是系统内部 URL，跳过
        if !ossupload.IsExternalURL(urlStr) {
            continue
        }

        slog.InfoContext(ctx, "auto_upload_start",
            "field", name,
            "original_url", urlStr,
        )

        result, err := ossupload.Upload(ctx, apiClient, urlStr)
        if err != nil {
            return fmt.Errorf("字段 %s 图片转存失败: %w", name, err)
        }

        input[name] = result.ImageURL

        slog.InfoContext(ctx, "auto_upload_complete",
            "field", name,
            "new_url", result.ImageURL,
        )
    }
    return nil
}
```

在 `buildSubCommand` 的 Run 函数中插入调用，位置在 ValidateAll 之后、API 请求之前：

```go
// 校验参数
if errs := ValidateAll(input, route.Template); len(errs) > 0 {
    // ... 现有错误处理 ...
}

// 自动转存外部图片 URL
if err := processAutoUpload(ctx, input, route.Template, client); err != nil {
    output.PrintError(os.Stderr, err.Error())
    os.Exit(1)
}

// 执行 API 请求（现有代码）
```

### 3. --template 输出提示

在 `printTemplateTo` 中，对 autoUpload 字段添加 note 提示：

```go
if field.AutoUpload == "image" {
    entry["note"] = "外部图片URL将自动转存到系统素材库"
}
```

当字段同时有 autoUpload 和 type=path/date 时，autoUpload 的 note 优先（autoUpload=image 的字段类型总是 string，不会和 path/date 冲突）。

### 4. Workflow 层简化

标记了 `autoUpload: image` 的字段在 workflow 中不再需要 upload-avatar 步骤。

**agent.yaml workflow 变更前**：
```yaml
steps:
  - id: upload-avatar
    description: 转存外部图片
    command: asset upload-image
    params:
      url: "{{inputs.avatar}}"
    output:
      imageUrl: "response.imageUrl"
  - id: create
    params:
      avatar: "{{steps.upload-avatar.imageUrl}}"
```

**变更后**：
```yaml
steps:
  - id: create
    params:
      avatar: "{{inputs.avatar}}"  # 直接使用原始值，cmdgen 会自动转存
```

**workflow Describe 输出增强**：

在 Describe 输出的"需要收集的信息"部分，对 autoUpload 字段自动追加提示：

```
1. avatar (必填): 头像URL
   提示: 外部图片URL将自动转存到系统素材库
```

实现方式：workflow 的 Input 新增 `AutoUpload` 字段，Describe 中检测到该字段时追加提示。但更简单的方式是：workflow YAML 的 inputs 不需要声明 autoUpload（这是路由层的事），而是在 cmdgen 的 `--template` 输出中体现。AI 读取 `agent create --template` 的输出时已经能看到 note。

**因此 workflow 层不需要代码改动，只需删除 upload-avatar 步骤和简化参数引用即可。**

## 受影响文件清单

### 新增/修改字段
- `internal/router/router.go` - Field 新增 AutoUpload 字段

### cmdgen 核心逻辑
- `internal/cmdgen/cmdgen.go` - 新增 processAutoUpload 函数，修改 buildSubCommand 调用链
- `internal/cmdgen/cmdgen.go` - printTemplateTo 添加 autoUpload note

### 路由 YAML（添加标记）
- `cmd/ckjr-cli/routes/agent.yaml` - avatar 字段添加 autoUpload: image（create 和 update）
- `cmd/ckjr-cli/routes/course.yaml` - courseAvatar 字段添加 autoUpload: image（create 和 update）

### Workflow YAML（简化）
- `cmd/ckjr-cli/workflows/agent.yaml` - 移除 upload-avatar 步骤，简化 avatar 参数引用
- `cmd/ckjr-cli/workflows/course.yaml` - 移除 3 个 workflow 的 upload-avatar 步骤，简化 courseAvatar 引用

### 不需要修改
- `internal/ossupload/` - 完整复用现有实现，无需改动
- `internal/workflow/` - 结构无变化
- `cmd/workflow/` - 逻辑无变化

## 测试策略

### 单元测试

1. **router.Field 解析测试** - 验证 autoUpload 字段能正确从 YAML 解析
2. **processAutoUpload 函数测试**：
   - 外部 URL 被转存，input 值被替换为 OSS URL
   - 内部 URL（aliyuncs.com/ckjr001.com）跳过转存
   - 空字符串跳过
   - 转存失败时返回错误
   - 非 string 类型跳过
   - 无 autoUpload 标记的字段跳过
3. **printTemplateTo 测试** - 验证 autoUpload=image 字段的 note 输出
4. **buildSubCommand 集成测试** - 完整流程：带 autoUpload 标记的命令自动转存

### 测试方法

利用 httptest.Server 模拟：
- 外部图片服务器
- OSS 服务器
- API 服务器（imageSign + addImgInAsset）
- 验证最终 API 请求的 input 中图片 URL 已被替换为 OSS URL

## 实现注意事项

1. **最小改动原则**：processAutoUpload 是纯函数，接收 input map 和 template，返回 error。不引入新的抽象或接口。

2. **日志可观测性**：转存开始和完成都通过 slog 记录，包含字段名、原始 URL、新 URL，便于排查问题。

3. **向后兼容**：autoUpload 是可选字段，不标记的字段行为完全不变。已有的内部 URL 调用不受影响。

4. **autoUpload 值设计**：当前只有 `image` 一种值。未来如需支持音频/视频转存，可扩展为 `audio`/`video`，processAutoUpload 根据 field.AutoUpload 值选择不同的转存逻辑。

5. **错误信息友好**：转存失败时，错误信息包含字段名和原始原因，方便 AI 和用户定位问题。
