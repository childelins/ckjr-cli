---
name: CLI 自更新命令实现
project: ckjr-cli
created: 2026-03-28
tags: [Go, CLI, self-update, GitHub Release, Cobra]
---

# CLI 自更新命令实现

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 自更新策略 | 重命名-替换（.bak 备份 + 回滚） | 跨平台兼容，不依赖外部命令，原子性回滚 |
| 版本比较 | semver 数值比较，手写实现 | 项目零外部依赖风格，cobra + yaml.v3 已是唯二依赖 |
| 下载方式 | Go 标准库 net/http | 无需假设用户系统有 curl/wget |
| 网络调用测试 | httptest + 可覆盖 API URL | 无需 mock 框架，Go 标准库够用 |
| 版本注入 | SetVersion() 函数由 root.go 调用 | 与项目现有 SetYAMLFS/SetVersion 模式一致 |

## 坑点预警

- **Go init 执行顺序**: main.go init() 先于 cmd/root.go init() 执行，所以 SetVersion() 在 root.go 的 init() 中调用时 version 变量已被注入
- **os.Executable() 解析符号链接**: 需要用 filepath.EvalSymlinks() 获取真实路径，否则替换的是符号链接而非真实二进制
- **tar.gz 中的目录结构**: GitHub Release 的 tar.gz 可能包含子目录（如 ckjr-cli_v0.2.0_linux_amd64/ckjr-cli），解压时需要递归查找匹配文件名前缀的文件
- **Windows 文件锁定**: Windows 上运行中的 exe 无法直接覆盖，重命名-替换策略可绕过（ren 旧文件后复制新文件）

## 复用模式

```go
// 测试中覆盖 API URL 的模式（避免硬编码外部依赖）
var defaultAPIURL = "https://api.github.com/repos/owner/repo/releases/latest"

cmd.Flags().StringVar(&apiURL, "api-url", defaultAPIURL, "GitHub API URL")
_ = cmd.Flags().MarkHidden("api-url")

// 测试时使用 httptest.NewServer 提供的 URL
cmd.SetArgs("--api-url", ts.URL)
```

```go
// 二进制替换的备份回滚模式
func replaceBinary(currentPath, newBinaryPath string) error {
    bakPath := currentPath + ".bak"
    os.Rename(currentPath, bakPath)      // 步骤1: 备份
    err := os.WriteFile(currentPath, ...) // 步骤2: 写入
    if err != nil {
        os.Rename(bakPath, currentPath)   // 回滚
        return err
    }
    os.Remove(bakPath)                    // 步骤3: 清理
    return nil
}
```
