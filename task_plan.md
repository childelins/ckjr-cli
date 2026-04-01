# 路由路径参数替换 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-31-path-params.md

## 概述

在 YAML 路由的 template 中通过 `type: path` 声明路径参数，请求时自动替换 path 中的 `{xxx}` 占位符。

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---

## 历史记录

- [已完成] ckjr-cli 初始实现 (Phase 1-12, 2026-03-25)
- [已完成] API Client 错误处理改进 (Phase 13-17, 2026-03-25)
- [已完成] Request Logging (Phase 18-22, 2026-03-25)
- [已完成] ckjr-agent Skill 实现 (Phase 23-24, 2026-03-25)
- [已完成] 私有仓库安装分发方案 (Phase 25-29, 2026-03-25)
- [已完成] Field Type/Example 字段扩展 (Phase 30-32, 2026-03-26)
- [已完成] CLI 重命名 ckjr -> ckjr-cli (Phase 33-40, 2026-03-26)
- [已完成] curl-to-yaml 实现 (Phase 41-44, 2026-03-26)
- [已完成] Field omitempty 修复 (Phase 45, 2026-03-26)
- [已完成] Skill 自发现改造 (Phase 46-48, 2026-03-26)
- [已完成] Request Body 日志 & Route 命令隐藏 (Phase 49-51, 2026-03-26)
- [已完成] Workflow YAML 实现 (Phase 52-57, 2026-03-26)
- [已完成] Routes Resource to Name (Phase 58-64, 2026-03-26)
- [已完成] Wiki 技术文档体系 (Phase 65-72, 2026-03-26)
- [已完成] Log Environment Modes (Phase 73-77, 2026-03-26)
- [已完成] Version Flag ldflags 注入修复 (Phase 78-80, 2026-03-26)
- [已完成] YAML 配置文件迁移到 config/ (Phase 81-85, 2026-03-27)
- [已完成] cmd 目录结构重组 (Phase 86-95, 2026-03-27)
- [已完成] 本地多平台构建与 GitHub Release 发布 (Task 1-5, 2026-03-27)
- [已完成] install.sh 简化 (Phase 96-99, 2026-03-27)
- [已完成] Update 命令实现 (Phase 100-104, 2026-03-28)
- [已完成] Field 类型与约束校验 (Phase 105-111, 2026-03-28/29)
- [已完成] YAML 配置文件兜底测试验证 (Phase 112-113, 2026-03-29)
- [已完成] AI 友好错误处理 (Task 1-7, 2026-03-29)
- [已完成] 生产环境静默 HTTP 请求日志 (Phase 1-3, 2026-03-29)
- [已完成] Workflow YAML 快速创建 (Task 1-3, 2026-03-30)
- 完整历史详见 docs/superpowers/archive/

---

## Phase 1: IsPathParam + extractPlaceholders

- **Source**: Plan -> Task 1-2
- **Status**: complete (0654470)
- **Description**: 创建 pathparam.go，实现 IsPathParam 判断和 extractPlaceholders 提取占位符

---

## Phase 2: PathParamError + ReplacePath

- **Source**: Plan -> Task 3-4
- **Status**: complete (0654470)
- **Description**: 实现 PathParamError 错误类型和 ReplacePath 路径参数替换函数

---

## Phase 3: validate.go 修改

- **Source**: Plan -> Task 5-7
- **Status**: complete (6671e85)
- **Description**: validateTypes/validateRequiredErrors/validateConstraints 跳过 type: path 字段

---

## Phase 4: cmdgen.go 集成

- **Source**: Plan -> Task 8
- **Status**: complete (5aa0333)
- **Description**: buildSubCommand 中在 ValidateAll 之前调用 ReplacePath，使用 resolvedPath 发送请求

---

## Phase 5: YAML 更新

- **Source**: Plan -> Task 9
- **Status**: complete (0410d8c)
- **Description**: 更新 course.yaml，在 update 路由 template 中添加 courseId path 字段

---

# Response Filter 实现计划

> Source plan: docs/superpowers/plans/2026-04-01-response-filter.md

## 概述

在 route YAML 中支持 `response` 字段定义，通过 fields(白名单) 或 exclude(黑名单) 过滤 API 响应的顶层字段输出。

---

## Task 1: Route 结构扩展 -- 新增 ResponseFilter

- **Source**: Plan -> Task 1
- **Status**: complete (93853df)
- **Description**: 在 router.go 新增 ResponseFilter 结构体，Route 结构新增 Response 字段

---

## Task 2: filterByFields -- 白名单过滤函数

- **Source**: Plan -> Task 2
- **Status**: complete (f9d1684)
- **Description**: 在 cmdgen/filter.go 实现 filterByFields，仅保留白名单顶层 key

---

## Task 3: filterByExclude -- 黑名单过滤函数

- **Source**: Plan -> Task 3
- **Status**: complete (3f90d36)
- **Description**: 在 cmdgen/filter.go 实现 filterByExclude，移除黑名单顶层 key

---

## Task 4: FilterResponse -- 顶层过滤入口函数

- **Source**: Plan -> Task 4
- **Status**: complete (c127fd0)
- **Description**: 实现 FilterResponse 入口函数，支持 fields/exclude，处理边界情况

---

## Task 5: 集成 FilterResponse 到 cmdgen 输出前

- **Source**: Plan -> Task 5
- **Status**: complete (6372250)
- **Description**: 在 buildSubCommand 的 output.Print 之前调用 FilterResponse(result, route.Response)，新增 3 个集成测试

---

## Task 6: 更新 course.yaml get 路由 response fields

- **Source**: Plan -> Task 6
- **Status**: complete (d754346)
- **Description**: 为 course get 路由添加 response.fields 白名单 (7 个核心字段)

---

# Response Filter 自动数组穿透 - 任务计划

> Source plan: docs/superpowers/plans/2026-04-01-array-traversal.md

## 概述

增强 response filter 的 dot notation 路径解析，遇到 `[]interface{}` 自动穿透数组，对每个元素应用剩余路径，支持分页列表场景。

---

## Phase 1: deepCopyMap 数组支持

- **Source**: Plan -> Task 1
- **Status**: complete (6568e7d)
- **Description**: deepCopyMap 增加 deepCopyValue 递归处理数组内 map 的深拷贝

---

## Phase 2: getNestedValue 数组穿透

- **Source**: Plan -> Task 2
- **Status**: complete (683cf69)
- **Description**: getNestedValue 遍历路径时遇到 []interface{} 自动穿透数组，对每个元素递归取值

---

## Phase 3: deleteNestedPath 数组穿透

- **Source**: Plan -> Task 3
- **Status**: complete (0a7f662)
- **Description**: deleteNestedPath 穿透数组对每个元素递归删除目标字段

---

## Phase 4: filterByExclude 数组穿透验证

- **Source**: Plan -> Task 4
- **Status**: complete (83fb32b)
- **Description**: filterByExclude 无需改代码，验证底层增强后自动获得穿透能力

---

## Phase 5: filterByFields 重构为 applyFieldPath

- **Source**: Plan -> Task 5
- **Status**: complete (181678d)
- **Description**: filterByFields 从 get-then-set 重构为 applyFieldPath 递归构建模式，支持数组穿透

---

## Phase 6: FilterResponse 集成测试

- **Source**: Plan -> Task 6
- **Status**: complete (0141369)
- **Description**: 端到端集成测试验证 FilterResponse 在 list 场景下的 fields/exclude 数组穿透

---

## Phase 7: course.yaml list 路由配置

- **Source**: Plan -> Task 7
- **Status**: complete (4389f52)
- **Description**: 更新 course.yaml list 路由添加 response.fields 白名单

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|
