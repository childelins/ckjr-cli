# install.sh 简化实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 简化 install.sh，移除 Go 安装方式，仅保留 GitHub Release 下载；同步修复文档中的分支名 bug。

**Architecture:** 删除 install.sh 中 Go 相关函数和交互逻辑，保留 install_via_release() 作为唯一安装路径。同步更新 wiki/install.md 和 README.md。

**Tech Stack:** Bash, Markdown

---

### Task 1: 简化 install.sh

**Files:**
- Modify: `install.sh:39-63` (删除 has_go + install_via_go)
- Modify: `install.sh:167-182` (简化 main 函数)

- [ ] **Step 1: 删除 has_go() 函数（L39-41）**

删除以下代码：
```bash
# 检测 Go 环境
has_go() {
    command -v go &> /dev/null
}
```

- [ ] **Step 2: 删除 install_via_go() 函数（L43-63）**

删除以下代码：
```bash
# Go install 方式安装
install_via_go() {
    ...整个函数...
}
```

- [ ] **Step 3: 简化 main() 函数（L167-184）**

将 main 函数从：
```bash
main() {
    info "Installing $BINARY_NAME from $REPO"
    if has_go; then
        info "Go environment detected"
        read -p "Use 'go install' method? (y/n, default: y): " use_go
        if [ -z "$use_go" ] || [ "$use_go" = "y" ]; then
            install_via_go
            exit 0
        fi
    fi
    install_via_release
}
```

简化为：
```bash
main() {
    info "Installing $BINARY_NAME from $REPO"
    install_via_release
}
```

- [ ] **Step 4: 验证 install.sh 语法正确**

Run: `bash -n install.sh`
Expected: 无输出（语法正确）

- [ ] **Step 5: 验证无 Go 残留引用**

Run: `grep -n 'has_go\|install_via_go\|go install' install.sh`
Expected: 无匹配

- [ ] **Step 6: 提交**

```bash
git add install.sh
git commit -m "refactor: remove go install method from install.sh, keep release-only"
```

### Task 2: 更新 wiki/install.md

**Files:**
- Modify: `wiki/install.md`

- [ ] **Step 1: 修复 curl URL 分支名 main → master（L19）**

```diff
-curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
+curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

- [ ] **Step 2: 简化安装位置说明（L11-12）**

移除 `go install：$GOPATH/bin/ckjr-cli` 行，仅保留：
```
- 一键脚本：`~/.local/bin/ckjr-cli`
```

- [ ] **Step 3: 删除 "方式二：go install" 整节（L26-30）**

删除：
```markdown
## 方式二：go install（Go 开发者）

适用于已安装 Go 环境的开发者。

\```bash
go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
\```
```

- [ ] **Step 4: "方式三" 改为 "方式二"（L32）**

```diff
-## 方式三：从源码构建（贡献者）
+## 方式二：从源码构建（贡献者）
```

- [ ] **Step 5: 简化常见问题表**

移除 `go install 失败` 行，简化 Release 下载失败的描述（移除 "或切换到 go install" 建议）：

```markdown
| 问题 | 解决方案 |
|------|---------|
| `command not found` | 检查 PATH 是否包含 `~/.local/bin` |
| Release 下载失败 | 检查网络连接，或切换到源码构建方式 |
```

- [ ] **Step 6: 验证文档无 go install 残留**

Run: `grep -n 'go install' wiki/install.md`
Expected: 无匹配（源码构建中的 go build 不算）

- [ ] **Step 7: 提交**

```bash
git add wiki/install.md
git commit -m "docs: update install guide, remove go install method, fix branch name"
```

### Task 3: 更新 README.md

**Files:**
- Modify: `README.md:10,13`

- [ ] **Step 1: 修复 curl URL 分支名（L10）**

```diff
-curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
+curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

- [ ] **Step 2: 更新安装方式描述（L13）**

```diff
-更多安装方式（go install / 源码构建 / Fork 自定义）见 [安装指南](wiki/install.md)。
+更多安装方式（源码构建 / Fork 自定义）见 [安装指南](wiki/install.md)。
```

- [ ] **Step 3: 提交**

```bash
git add README.md
git commit -m "docs: fix install URL branch name, remove go install reference"
```

### Task 4: 最终验证

- [ ] **Step 1: 全局验证无 go install 残留**

Run: `grep -rn 'go install' install.sh wiki/install.md README.md`
Expected: 无匹配

- [ ] **Step 2: 验证 install.sh 语法**

Run: `bash -n install.sh`
Expected: 无输出

- [ ] **Step 3: 验证 curl URL 一致性**

确认 README.md 和 wiki/install.md 中的 curl URL 都使用 `master` 分支。
