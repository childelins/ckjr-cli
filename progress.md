# Progress

## 历史摘要

- 2026-03-25: ckjr-cli 初始实现、API 错误处理、Request Logging、ckjr-agent Skill、私有仓库安装
- 2026-03-26: Field Type/Example、CLI 重命名、curl-to-yaml、Skill 自发现、Request Body 日志、Workflow YAML、Routes Resource→Name、Wiki 文档、Log Environment Modes、Version Flag ldflags
- 2026-03-27: YAML 迁移到 config/、cmd 目录重组、本地构建发布、install.sh 简化
- 2026-03-28: Update 命令 (Phase 100-104)、Field 类型与约束校验 (Phase 105-111)
- 2026-03-29: YAML 兜底测试、AI 友好错误处理、生产环境静默日志
- 2026-03-30: Workflow YAML 快速创建 (Task 1-3, commit a70bb2e)
- 2026-03-31: 路由路径参数替换 (Phase 1-5, pathparam.go/validate.go/cmdgen.go 集成, commits 0654470-0410d8c)

## 2026-04-01 Response Filter 实现计划

### Task 1: Route 结构扩展 -- 新增 ResponseFilter
- Status: complete (93853df)
- router.go 新增 ResponseFilter 结构体 (Fields/Exclude)，Route 结构新增 Response *ResponseFilter 字段
- router_test.go 新增 3 个测试: Unmarshal/Exclude/Nil
- cmdgen 包 80+ 测试全量通过

### Task 2: filterByFields -- 白名单过滤函数
- Status: complete (f9d1684)
- cmdgen/filter.go 新建，实现 filterByFields 纯函数
- filter_test.go 新增 4 个测试: AllMatch/PartialMatch/NoneMatch/PreservesNested

### Task 3: filterByExclude -- 黑名单过滤函数
- Status: complete (3f90d36)
- filter.go 追加 filterByExclude 纯函数
- filter_test.go 追加 3 个测试: AllMatch/PartialMatch/NoneMatch

### Task 4: FilterResponse -- 顶层过滤入口函数
- Status: complete (c127fd0)
- filter.go 追加 FilterResponse 入口函数，支持 fields 优先于 exclude
- filter_test.go 追加 10 个测试: NilFilter/NonMap/Slice/FieldsOnly/ExcludeOnly/FieldsAndExclude/EmptyFields/EmptyExclude/FieldNotFound/EmptyResult
- cmdgen 包 80+ 测试全量通过，无回归
- 发现: Go slice 不可用 == 比较，需拆分测试用例

## 2026-04-02 OSS 图片上传实现

### Task 1: 数据结构与 IsExternalURL
- Status: complete (029ddb2)
- 创建 internal/ossupload 包
- 定义 ImageSignResponse、AssetImage 数据结构
- 实现 IsExternalURL 辅助函数（检查 URL 是否为外部图片）

### Task 2: 下载外部图片辅助函数
- Status: complete (f80864b)
- 实现 downloadImage 函数：支持 Content-Type 校验、大小限制（10MB）

### Task 3: 文件名与扩展名解析辅助函数
- Status: complete (fa8deb8)
- 实现 parseFileName/isKnownImageExt/extFromContentType
- 发现: mime.ExtensionsByType("image/jpeg") 返回 [.jpe .jpeg .jpg]，需优先选择 .jpg

### Task 4: OSS 直传函数
- Status: complete (5a54148)
- 实现 uploadToOSS multipart/form-data 直传函数

### Task 5: Upload 总入口函数
- Status: complete (1e99ade)
- 实现 Upload 函数编排完整 4 步流程：imageSign -> download -> uploadToOSS -> addImgInAsset

### Task 6: asset upload-image 子命令
- Status: complete (be3dc3a)
- 创建 cmd/upload.go，注册 asset upload-image 子命令
- 在 registerRouteCommands 中为 asset 命令额外添加 upload-image 子命令

### Task 7: 更新 course workflow
- Status: complete (c1d49ce)
- course.yaml 三个工作流添加 upload-avatar 步骤
- allowed-routes 添加 asset
- yaml_validate_test.go 添加手动注册命令白名单

### Task 8: 全量测试与编译验证
- Status: complete
- 全部 18 个包测试通过，无回归
- go build ./... 编译通过
- go run ./cmd/ckjr-cli asset upload-image --help 命令注册验证通过

### Task 5: 集成 FilterResponse 到 cmdgen 输出前
- Status: complete (6372250)
- cmdgen.go 在 output.Print 之前插入 `result = FilterResponse(result, route.Response)`
- cmdgen_test.go 新增 3 个集成测试: ResponseFilter(fields)/ResponseFilter(exclude)/NoResponseFilter
- 测试使用 os.Pipe 捕获 stdout（因 output.Print 直接写 os.Stdout）
- 17 个包全部通过，无回归

### Task 6: 更新 course.yaml get 路由 response fields
- Status: complete (d754346)
- course.yaml get 路由添加 response.fields 白名单: courseId/name/courseType/status/price/payType/courseAvatar
- 构建和全量测试通过

## 2026-04-01 Response Filter 自动数组穿透

### Phase 1: deepCopyMap 数组支持
- Status: complete (6568e7d)
- 新增 deepCopyValue 递归函数，处理 map/array/原始值深拷贝
- deepCopyMap 改为使用 deepCopyValue，支持数组内 map 的独立深拷贝

### Phase 2: getNestedValue 数组穿透
- Status: complete (683cf69)
- getNestedValue 增加 []interface{} type switch 分支
- 遇到数组时对每个元素递归调用，收集结果为 []interface{}
- 4 个新子测试 + 5 个原有子测试全部通过

### Phase 3: deleteNestedPath 数组穿透
- Status: complete (0a7f662)
- 重构为 deleteNestedPath + deleteNestedParts 递归模式
- 增加 []interface{} 分支，穿透数组对每个 map 元素递归删除
- 3 个新子测试 + 3 个原有子测试全部通过

### Phase 4: filterByExclude 数组穿透验证
- Status: complete (83fb32b)
- 无代码修改，仅新增测试验证底层增强后 filterByExclude 自动获得穿透能力
- deepCopyMap 正确深拷贝数组内 map + deleteNestedPath 穿透删除 = 原始数据不被修改

### Phase 5: filterByFields 重构为 applyFieldPath
- Status: complete (181678d)
- 核心重构：filterByFields 从 get-then-set 模式改为 applyFieldPath 递归构建
- applyFieldPath 处理 map/array/leaf 三种情况，遇到数组对每个元素分别构建
- 修复: 不存在的嵌套路径不应创建空的中间 map 结构
- 3 个新数组测试 + 所有 12 个 filterByFields 原有测试全部通过，无回归

### Phase 6: FilterResponse 集成测试
- Status: complete (0141369)
- 新增 TestFilterResponse_ListWithFields: 完整分页列表 fields 场景端到端
- 新增 TestFilterResponse_ListWithExclude: 排除数组内字段端到端
- cmdgen 包 80+ 测试全部通过

### Phase 7: course.yaml list 路由配置
- Status: complete (4389f52)
- course.yaml list 路由添加 response.fields 白名单: 8 个 list.data 字段 + list.total/current_page/per_page
- 构建通过

## 2026-04-01 Response Field Descriptions

### Phase 1: ResponseField 类型 + 自定义 UnmarshalYAML
- Status: complete (9a1fb4e)
- router.go: 新增 ResponseField 结构体 (Path+Description)，ResponseFilter.Fields 从 []string 改为 []ResponseField
- 自定义 UnmarshalYAML 支持纯字符串和 path+description 对象两种 YAML 格式
- FieldPaths() 方法提取纯路径列表
- 2 个新测试 (MixedFieldFormats + BackwardCompat) + 原有测试全部通过

### Phase 2: 迁移 FilterResponse 使用 FieldPaths
- Status: complete (db8ae4c)
- filter.go: FilterResponse 改用 FieldPaths() 提取路径列表
- filter_test.go + cmdgen_test.go: 所有 ResponseFilter 构造从 []string 迁移为 []ResponseField
- 全量 80+ 测试通过，无回归

### Phase 3: --template 输出 request/response 结构
- Status: complete (f3b6144)
- cmdgen.go: printTemplateTo 输出结构从扁平改为 { "request": {...}, "response": {...} }
- 新增 2 个测试 (WithResponse + WithoutResponse)，更新 3 个已有测试的解析逻辑
- 全量测试通过

### Phase 4: 为 course.yaml 添加响应字段描述
- Status: complete (6c95f9b)
- course.yaml list 路由: 6 个字段添加描述 (courseType/status/isSaleOnly/payType/contentAuditStatus/name)
- course.yaml get 路由: 5 个字段添加描述 (courseType/status/isSaleOnly/payType/playMode/articleType)
- 构建通过

## 2026-04-01 date 类型支持

### Phase 1: date 类型校验
- Status: complete (229cb4b)
- validate.go: import "time"，validateType 新增 case "date" 分支
- time.Parse 校验 "2006-01-02 15:04:05" 格式和日期合法性
- 10 个测试 (8 个 table-driven + 2 个错误信息断言) 全部通过

### Phase 2: --template 输出 date note
- Status: complete (4bfd67c)
- cmdgen.go: printTemplateTo 中 date 类型添加 note "日期格式: YYYY-MM-DD HH:MM:SS"
- 1 个新测试 (DateFieldNote) + 全量 90+ 测试通过，无回归

### Phase 3: 更新文档
- Status: complete (407464b)
- core-concepts.md: 类型表新增 date 行
- extending.md: type 属性说明补充 date 类型

## 2026-04-02 OSS 图片上传实现

### Task 1: 数据结构与 IsExternalURL
- Status: complete (029ddb2)
- 创建 internal/ossupload 包，定义 ImageSignResponse/AssetImage 结构体
- 实现 IsExternalURL 辅助函数，识别 aliyuncs.com/ckjr001.com 域名

### Task 2: 下载外部图片辅助函数
- Status: complete (f80864b)
- 实现 downloadImage 函数，支持 Content-Type 校验、大小限制（10MB）

### Task 3: 文件名与扩展名解析辅助函数
- Status: complete (fa8deb8)
- 实现 parseFileName/isKnownImageExt/extFromContentType
- 修复 mime.ExtensionsByType("image/jpeg") 返回 .jpe 而非 .jpg 的问题

### Task 4: OSS 直传函数
- Status: complete (5a54148)
- 实现 uploadToOSS multipart/form-data 直传函数

### Task 5: Upload 总入口函数
- Status: complete (1e99ade)
- 实现 Upload 函数编排完整 4 步流程：imageSign -> download -> uploadToOSS -> addImgInAsset

### Task 6: asset upload-image 子命令
- Status: complete (be3dc3a)
- 创建 cmd/upload.go，注册 upload-image 子命令到 asset 命令

### Task 7: 更新 course workflow
- Status: complete (c1d49ce)
- course.yaml 三个工作流添加 upload-avatar 步骤
- allowed-routes 添加 asset
- yaml_validate_test.go 添加手动注册命令白名单

### Task 8: 全量测试与编译验证
- Status: complete
- 全部 18 个包测试通过，无回归
- go build ./... 编译通过
- go run ./cmd/ckjr-cli asset upload-image --help 命令注册验证通过

## 2026-04-02 环境配置默认 base_url 实现

### Phase 1: config 包新增 DefaultBaseURL 和 ResolveBaseURL
- Status: complete (57c49d4)
- config.go: 新增 envBaseURLs map (development/production)、environment 变量、SetEnvironment/DefaultBaseURL/ResolveBaseURL
- config_test.go: 新增 5 个测试 (DefaultBaseURL_Development/Production/UnknownFallback, ResolveBaseURL_WithBaseURL/EmptyBaseURL)
- 全量 config 包 9 个测试通过

### Phase 2: cmd/root.go 接入 ResolveBaseURL
- Status: complete (c6eb88b)
- cmd.SetEnvironment 转发给 internalconfig.SetEnvironment
- createClient 改用 cfg.ResolveBaseURL() 替代 cfg.BaseURL
- 全量 18 个包测试通过，编译通过
