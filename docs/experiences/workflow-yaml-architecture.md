---
name: routes-resource-to-name
project: ckjr-cli
created: 2026-03-26
tags: [refactor, breaking-change, yaml-consistency]
---

# Routes Resource to Name 重构经验

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 向后兼容 | 否 | 简化代码，一次性迁移 |
| CLI 参数名 | --name | 与 YAML 字段保持一致 |
| Go 结构体字段 | Name | 简洁直接，无命名冲突 |

## 实施顺序

**TDD 驱动的四阶段策略：**
1. **代码修改** - 先改结构体定义，让测试失败
2. **依赖更新** - 逐个更新使用该字段的代码
3. **YAML 文件** - 最后更新配置文件
4. **测试更新** - 更新测试断言，确保通过

## 影响范围

| 类型 | 数量 | 文件 |
|------|------|------|
| Go 结构体 | 1 | router.RouteConfig |
| 使用该字段的代码 | 5 | cmdgen, yamlgen, route, 测试文件 |
| CLI 参数 | 2 | --resource, --resource-desc |
| YAML 文件 | 2 | agent.yaml, common.yaml |

## 坑点预警

1. **yamlgen.CreateFile 参数顺序调整**
   - 原: `resource, resourceDesc, name`
   - 新: `name, nameDesc, routeName`
   - 调用方需同步更新参数顺序

2. **route.go 新增 inferNameFromPath 函数**
   - 移除 --resource 参数后，新建文件需从路径推导 name
   - 新增 `inferNameFromPath` 函数从文件路径推导资源名

## 复用模式

```go
// 1. 结构体字段重命名
type RouteConfig struct {
    Name string `yaml:"name"`  // 原: Resource
}

// 2. 更新所有引用点
cfg.Name  // 原: cfg.Resource

// 3. 更新 YAML 文件
name: common  # 原: resource: common
```

## 验收标准

- 所有测试通过
- `go run . agent describe` 正常工作
- `route import --name-desc` 正常工作
- 生成的 YAML 文件使用 `name` 字段
- 代码中无 `cfg.Resource` 引用
