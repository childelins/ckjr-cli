# install.sh 简化设计文档

> Created: 2026-03-27
> Status: Draft

## 概述

将 install.sh 从双模式安装（Go install + Release 下载）简化为单模式（仅 Release 下载）。仓库已从私有转为公共，Go install 和 GITHUB_TOKEN 认证不再是必要的安装路径。同时修复 curl 安装 URL 中的分支名错误（main -> master）。

## 变更范围

### 1. install.sh 修改

**删除的代码：**

| 行号 | 内容 | 原因 |
|------|------|------|
| 39-41 | `has_go()` 函数 | 不再需要检测 Go 环境 |
| 43-63 | `install_via_go()` 函数 | 移除 Go 安装路径 |
| 167-182 | `main()` 中的 Go 检测和交互式提问逻辑 | 不再需要 |

**简化的代码：**

| 位置 | 变更 | 原因 |
|------|------|------|
| `install_via_release()` 中 GITHUB_TOKEN 相关逻辑 | 保留但简化注释 | 公共仓库不需要 token，但保留支持可避免 API 限流 |
| `main()` 函数 | 直接调用 `install_via_release` | 无需分支判断 |

**简化后的 main() 函数：**

```bash
main() {
    info "Installing $BINARY_NAME from $REPO"
    install_via_release
}
```

**简化后的 install.sh 结构：**

```
配置变量（REPO, BINARY_NAME, INSTALL_DIR）
颜色输出函数（info, warn, error）
detect_os()
detect_arch()
install_via_release()   # 唯一安装方式
main()                  # 直接调用 install_via_release
```

### 2. wiki/install.md 修改

**删除：**
- "方式二：go install（Go 开发者）" 整节
- 前置条件表中 "go install：`$GOPATH/bin/ckjr-cli`" 行
- 常见问题中 "`go install` 失败" 行
- 常见问题中 "Release 下载失败" 行的 "或切换到 go install" 建议

**修改：**
- "方式三：从源码构建" 改为 "方式二：从源码构建"
- curl URL 分支名：`main` -> `master`（修复 bug，当前默认分支为 master）
- 安装位置说明简化为仅列出 `~/.local/bin/ckjr-cli`

**修改后的文档结构：**

```
前置条件
方式一：一键安装脚本（推荐）
方式二：从源码构建（贡献者）
验证安装
常见问题
Fork 自定义
```

### 3. README.md 修改

**修改：**
- curl URL 分支名：`main` -> `master`
- 安装方式引用文字：移除 "go install /" 部分

## 分支名 Bug 修复

当前 GitHub remote 的默认分支为 `master`（通过 `git branch -r` 确认：`github/master`），但文档中的 raw URL 使用了 `main`：

```
# 当前（错误）
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash

# 修正后
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

影响文件：
- `README.md` 第 10 行
- `wiki/install.md` 第 19 行

## GITHUB_TOKEN 处理策略

保留 `install_via_release()` 中的 GITHUB_TOKEN 支持，原因：
1. GitHub API 对匿名请求有 60 次/小时的限流
2. 带 token 请求限流为 5000 次/小时
3. CI/CD 环境或频繁安装场景可能触发限流
4. 代码量很小，保留不增加复杂度

但移除 `install_via_go()` 中的 GITHUB_TOKEN 认证逻辑（整个函数删除）。

## 错误处理

无新增错误场景。现有 `install_via_release()` 的错误处理已覆盖：
- 不支持的 OS/架构 -> `error()` 退出
- curl/wget 都不存在 -> `error()` 退出
- Release 不存在或网络错误 -> `error()` 退出
- 归档中未找到二进制 -> `error()` 退出

## 测试策略

| 测试项 | 验证方式 |
|--------|---------|
| install.sh 语法正确 | `bash -n install.sh` |
| 删除后无残留引用 | grep 确认无 `has_go`、`install_via_go`、`go install` 引用 |
| curl URL 可达 | `curl -fsSL -o /dev/null -w "%{http_code}" <URL>` 返回 200 |
| 实际安装 | 在干净环境中执行 curl 管道安装，验证二进制可运行 |
| 文档一致性 | 检查 README.md 和 wiki/install.md 中的 URL 和安装方式描述一致 |

## 实现注意事项

1. **变更量小**：主要是删除代码，风险低
2. **向后兼容**：已通过 curl 安装的用户不受影响，重新安装使用新脚本
3. **执行顺序**：先改 install.sh，再改文档，最后验证
4. **不涉及 Makefile**：构建和发布流程不变
