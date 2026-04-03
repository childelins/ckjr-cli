# OSS 图片上传设计文档

> Created: 2026-04-02
> Status: Draft

## 概述

为 ckjr-cli 引入 OSS 图片上传能力，将外部图片 URL 转存到系统素材库。支持两种使用模式：独立 CLI 命令直接调用，以及 workflow 中自动检测外部 URL 并转存（用户无感）。

完整流程：获取 OSS 上传签名 -> 下载外部图片 -> 直传到阿里云 OSS -> OSS 回调通知服务端 -> 保存到素材库。

## 背景

课程创建等场景要求图片 URL 指向系统素材库（OSS 域名下的资源）。用户可能提供外部图片 URL（如第三方图床），需要在提交前自动转存。

前端 Web 端的直传流程已有成熟实现，CLI 需要模拟同样的 3 步 API 调用，其中 OSS 直传使用 multipart/form-data 发送到阿里云 OSS 域名，与现有 JSON API 调用模式不同。

## 架构

### 新增组件

```
internal/
  ossupload/           # 新增：OSS 图片上传核心逻辑
    upload.go          # Upload 函数及数据结构
    upload_test.go

cmd/
  upload.go            # 新增：asset upload 子命令
```

### 与现有系统的关系

```
                    +-- 独立 CLI 命令模式 --+
                    |                       |
用户/Workflow ------>  cmd/upload.go  ------+
                    |                       |
                    +-- Workflow 工具函数 --+
                                            |
                                            v
                                  internal/ossupload/upload.go
                                            |
                              +-------------+-------------+
                              |             |             |
                              v             v             v
                         api.Client    http.Client    api.Client
                        (imageSign)   (OSS 直传)   (addImgInAsset)
```

- `ossupload` 包是纯功能包，不依赖 cobra 命令框架
- CLI 命令和 Workflow 都通过调用 `ossupload.Upload()` 完成转存
- OSS 直传需要直接使用 `http.Client`（不走 `api.Client.DoCtx`，因为目标是外部 OSS 域名且使用 multipart/form-data）

## 数据流

### 完整转存流程

```
外部图片 URL
    |
    v
Step 1: GET /admin/assets/imageSign?type=2
    |   请求: api.Client (JSON, Bearer auth)
    |   响应: { key, policy, OSSAccessKeyId, signature, callback,
    |           success_action_status, origin, host, ... }
    v
Step 2: GET <外部图片URL> -> 下载图片字节流
    |
    v
Step 3: POST https://knowledge-payment.oss-cn-beijing.aliyuncs.com/
    |   请求: multipart/form-data (key, policy, OSSAccessKeyId,
    |          signature, callback, file=<图片字节>)
    |   目标: 阿里云 OSS（非系统 API，无 Bearer auth）
    |   响应: OSS 自动回调 imageCallBack 通知服务端
    v
Step 4: POST /admin/assets/addImgInAsset
    |   请求: api.Client (JSON, Bearer auth)
    |   body: { data: [{ fId, level, parentId, size, height,
    |                    width, suffix, name, imageUrl }] }
    |   响应: 素材库记录
    v
返回素材库图片 URL
```

### 关键数据结构

**imageSign 响应**（推测结构，需实际验证）：

```json
{
  "data": {
    "key": "lj7l/resource/imgs/eeb49984/admin-fe_lj7l_uploadBox_xxx.png",
    "policy": "<base64>",
    "OSSAccessKeyId": "LTAIEooZEnvlRbrb",
    "signature": "xxx",
    "callback": "<base64>",
    "success_action_status": "200",
    "origin": "0",
    "host": "https://knowledge-payment.oss-cn-beijing.aliyuncs.com/"
  }
}
```

**OSS 直传 multipart 字段**：

| 字段 | 来源 | 说明 |
|------|------|------|
| key | imageSign 响应 | OSS 对象路径 |
| policy | imageSign 响应 | 签名策略 |
| OSSAccessKeyId | imageSign 响应 | OSS AccessKey |
| success_action_status | imageSign 响应 | 固定 "200" |
| callback | imageSign 响应 | 回调配置（含 width/height 自动提取） |
| signature | imageSign 响应 | 签名 |
| origin | imageSign 响应 | 固定 "0" |
| name | 文件名 | 原始文件名（不含扩展名） |
| x:realname | 文件名 | 同 name |
| file | 图片字节 | 实际文件内容 |

**addImgInAsset 请求体**：

```json
{
  "data": [{
    "fId": -1,
    "level": 1,
    "parentId": 0,
    "size": 0.11,
    "height": "512",
    "width": "512",
    "suffix": ".png",
    "name": "avatar_1",
    "imageUrl": "https://knowledge-payment.oss-cn-beijing.aliyuncs.com/lj7l/resource/imgs/..."
  }]
}
```

**OSS callback 机制**：OSS 直传成功后，阿里云会自动调用 `imageCallBack` 接口，回调 body 中包含 `${imageInfo.height}` 和 `${imageInfo.width}`，由 OSS 自动从图片中提取。这意味着服务端可通过回调获取真实尺寸，CLI 端不需要自行解析图片尺寸。

## 组件设计

### 1. ossupload 包 (`internal/ossupload/upload.go`)

```go
package ossupload

// ImageSignResponse imageSign 接口响应
type ImageSignResponse struct {
    Key                string `json:"key"`
    Policy             string `json:"policy"`
    OSSAccessKeyId     string `json:"OSSAccessKeyId"`
    Signature          string `json:"signature"`
    Callback           string `json:"callback"`
    SuccessActionStatus string `json:"success_action_status"`
    Origin             string `json:"origin"`
    Host               string `json:"host"`
}

// AssetImage 素材库图片信息
type AssetImage struct {
    ImageURL string  // OSS 上的完整 URL
    Name     string  // 文件名（不含扩展名）
    Suffix   string  // 扩展名（含点，如 .png）
    Size     float64 // 文件大小（MB）
    Width    string  // 宽度（可为空，OSS callback 会处理）
    Height   string  // 高度（可为空，OSS callback 会处理）
}

// Upload 将外部图片 URL 转存到素材库
//
// 完整流程：imageSign -> 下载外部图片 -> 直传 OSS -> addImgInAsset
// apiClient: 系统API客户端（JSON 请求用）
// imageURL: 外部图片URL
// type_: imageSign 的 type 参数，固定为 2
func Upload(ctx context.Context, apiClient *api.Client, imageURL string) (*AssetImage, error)
```

#### 实现要点

**Step 1: 获取签名**

```go
var signResp ImageSignResponse
err := apiClient.DoCtx(ctx, "GET", "/admin/assets/imageSign?type=2", nil, &signResp)
```

通过现有 `api.Client.DoCtx` 发起请求，正常 JSON 响应处理。

**Step 2: 下载外部图片**

```go
resp, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
// 读取全部 body 到 []byte
// 提取 Content-Type 确定文件类型
// 从 URL 路径提取文件名
// 计算文件大小（MB）
```

使用标准 `http.Client`，不经过 `api.Client`。需要：
- 设置合理的 User-Agent
- 设置超时（建议 30 秒）
- 验证 Content-Type 是图片类型
- 限制最大文件大小（10MB，与 OSS policy 中的 1048576000 字节限制一致）

**Step 3: 直传 OSS**

```go
// 构造 multipart/form-data 请求
body := &bytes.Buffer{}
writer := multipart.NewWriter(body)

// 按顺序写入字段（与前端 curl 一致）
writer.WriteField("key", signResp.Key)
writer.WriteField("policy", signResp.Policy)
writer.WriteField("OSSAccessKeyId", signResp.OSSAccessKeyId)
writer.WriteField("success_action_status", signResp.SuccessActionStatus)
writer.WriteField("callback", signResp.Callback)
writer.WriteField("signature", signResp.Signature)
writer.WriteField("origin", signResp.Origin)
writer.WriteField("name", fileName)
writer.WriteField("x:realname", fileName)

// 写入文件
part, _ := writer.CreateFormFile("file", fileName+suffix)
part.Write(imageBytes)
writer.Close()

// 发送到 OSS（非系统 API，不需要 Bearer auth）
req, _ := http.NewRequestWithContext(ctx, "POST", signResp.Host, body)
req.Header.Set("Content-Type", writer.FormDataContentType())
```

注意：目标 URL 是 `signResp.Host`（阿里云 OSS 域名），不经过 `api.Client`。

**Step 4: 保存到素材库**

```go
ossURL := signResp.Host + "/" + signResp.Key
payload := map[string]interface{}{
    "data": []map[string]interface{}{
        {
            "fId":      -1,
            "level":    1,
            "parentId": 0,
            "size":     fileSizeMB,
            "height":   "",  // 宽高留空，OSS callback 已处理
            "width":    "",
            "suffix":   suffix,
            "name":     fileName,
            "imageUrl": ossURL,
        }
    },
}
var result interface{}
err := apiClient.DoCtx(ctx, "POST", "/admin/assets/addImgInAsset", payload, &result)
```

### 2. CLI 命令 (`cmd/upload.go`)

作为 asset 命令的子命令注册：

```
ckjr-cli asset upload-image '{"url": "https://example.com/avatar.png"}'
```

实现方式：不通过 YAML 路由定义（因为 OSS 直传不走标准 JSON API 流程），而是手动注册一个 Cobra 子命令到 asset 命令下。

在 `registerRouteCommands` 中，asset 路由命令构建完成后，额外添加 `upload-image` 子命令：

```go
// registerRouteCommands 中，构建完 asset 命令后：
if cfg.Name == "asset" {
    cmd.AddCommand(newUploadImageCmd(createClient))
}
```

`newUploadImageCmd` 返回一个 `*cobra.Command`，其 Run 函数：
1. 解析 JSON 参数获取 `url`
2. 调用 `ossupload.Upload(ctx, client, url)`
3. 输出结果（素材库图片 URL）

### 3. Workflow 集成

在课程创建 workflow 的 steps 中添加前置步骤：

```yaml
# course.yaml 修改示例
workflows:
  create-video-course:
    steps:
      - id: upload-avatar
        description: 转存外部图片URL到素材库
        command: asset upload-image
        params:
          url: "{{inputs.courseAvatar}}"
        output:
          imageUrl: "response.imageUrl"
      - id: create
        description: 创建视频课程
        command: course create
        params:
          courseAvatar: "{{steps.upload-avatar.imageUrl}}"
          # ... 其他参数不变
```

Workflow 是声明式知识文件，AI 执行时会读取描述后自行调用 `ckjr-cli asset upload-image` 命令。**不需要实现 runtime 检测外部 URL 的逻辑** -- AI 通过 workflow describe 输出理解应该在创建课程前先转存图片。

### 外部 URL 检测逻辑

AI 在 workflow 执行时根据 workflow 定义决定是否调用 upload-image。但如果需要 CLI 层面的自动检测（更可靠），可在 `ossupload` 包中提供一个辅助函数：

```go
// IsExternalURL 检查 URL 是否为外部图片（非系统 OSS 域名）
func IsExternalURL(imageURL string) bool
```

判断规则：如果 URL 不以系统 API 的 base_url 域名或 OSS 域名开头，则认为是外部 URL。此函数供 AI 或未来自动化场景使用。

## 错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| imageSign 接口失败 | 透传 api.Client 错误（已有的 Unauthorized/APIError 体系） |
| 外部图片下载失败 | 返回明确错误：`下载外部图片失败: <url>: <reason>` |
| 图片过大 | 下载前无法预知大小；下载后检查 Content-Length，超过 10MB 返回错误 |
| 非图片 Content-Type | 返回错误：`不支持的内容类型: <ct>，仅支持图片文件` |
| OSS 直传失败 | 返回错误：`OSS 上传失败: HTTP <status>` |
| OSS 直传返回非 200 | 检查响应状态码，非 200 时返回错误 |
| addImgInAsset 失败 | 透传 api.Client 错误 |

所有错误使用 `fmt.Errorf` 包装，保持与现有错误处理风格一致。

## 测试策略

### 单元测试 (`internal/ossupload/upload_test.go`)

使用 `httptest.Server` 模拟各步骤的 HTTP 服务：

1. **TestUpload_Success** -- 完整流程成功，返回素材库图片信息
2. **TestUpload_ImageSignFails** -- imageSign 返回错误
3. **TestUpload_DownloadFails** -- 外部图片下载失败（超时、404）
4. **TestUpload_InvalidContentType** -- 非图片类型拒绝
5. **TestUpload_OSSTransferFails** -- OSS 直传返回非 200
6. **TestUpload_AddAssetFails** -- addImgInAsset 返回错误
7. **TestIsExternalURL** -- 外部 URL 检测逻辑

### 集成测试

8. **TestUploadImageCmd** -- CLI 子命令端到端（httptest 模拟）

### 测试要点

- 模拟 imageSign 响应时需包含完整的签名字段
- 模拟 OSS 直传时需验证 multipart/form-data 格式正确
- 模拟 addImgInAsset 时需验证请求体结构

## 实现注意事项

### 关键实现细节

1. **OSS 直传不走 api.Client** -- 目标是外部 OSS 域名，使用 multipart/form-data，不需要 Bearer auth。直接使用 `http.Client`。

2. **文件名处理** -- 从 URL 路径中提取文件名和扩展名。如果 URL 无扩展名（如 `https://example.com/abc123`），使用 Content-Type 推断扩展名（`image/png` -> `.png`）。

3. **OSS 回调与尺寸** -- OSS callback 机制会自动将 `imageInfo.height` 和 `imageInfo.width` 回传给服务端。CLI 在 addImgInAsset 中传空字符串即可，不需要自行解析图片尺寸。但如果需要返回尺寸信息给调用方，可考虑用 `image.DecodeConfig` 从下载的字节流中获取。

4. **type 参数** -- imageSign 接口的 `type=2` 含义需要确认。目前硬编码为 2（与前端一致）。如果未来需要支持音频/视频上传，可暴露为参数。

5. **并发安全** -- `ossupload.Upload` 是无状态的纯函数，每次调用创建独立的 HTTP 请求，天然并发安全。

### 实现顺序

1. `internal/ossupload/upload.go` -- 核心上传逻辑 + 数据结构
2. `internal/ossupload/upload_test.go` -- 单元测试
3. `cmd/upload.go` -- CLI 子命令注册
4. 修改 `cmd/root.go` -- 在 asset 命令下注册 upload-image
5. 更新 `cmd/ckjr-cli/workflows/course.yaml` -- 添加 upload-avatar 步骤
6. 手动端到端测试

### 不在本期实现

- 音频/视频文件上传（只做图片）
- 本地文件上传（只做 URL 转存）
- 批量上传
- 上传进度显示
- 图片处理（压缩、裁剪、水印）
