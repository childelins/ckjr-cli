# config init 简化实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 从 `config init` 移除 base_url 交互输入，仅保留 api_key 配置

**Architecture:** 删除 `runConfigInit` 中 base_url 的 prompt/读取逻辑，保存时 BaseURL 留空，运行时由 `ResolveBaseURL()` 自动回退到 `DefaultBaseURL()`

**Tech Stack:** Go, Cobra

**Spec:** `docs/superpowers/specs/2026-04-02-config-init-simplify-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/config/config_test.go` | Modify | 新增 init 测试 |
| `cmd/config/config.go` | Modify | 删除 base_url 交互逻辑 |

---

### Task 1: 测试 config init 保存后 base_url 为空

**Files:**
- Modify: `cmd/config/config_test.go`

- [ ] **Step 1: 写失败测试**

在 `cmd/config/config_test.go` 末尾添加：

```go
func TestConfigInitSavesEmptyBaseURL(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// 模拟用户输入：只输入 api_key
	input := "test-api-key-12345\n"
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// 重定向 stdout 避免输出到终端
	oldStdout := os.Stdout
	_, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() {
		wOut.Close()
		os.Stdout = oldStdout
	}()

	cmd := NewCommand()
	cmd.SetArgs([]string{"init"})
	cmd.Execute()

	wOut.Close()
	os.Stdout = oldStdout

	// 验证保存后的配置
	loaded, err := internalconfig.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BaseURL != "" {
		t.Errorf("BaseURL = %q, want empty string", loaded.BaseURL)
	}
	if loaded.APIKey != "test-api-key-12345" {
		t.Errorf("APIKey = %q, want test-api-key-12345", loaded.APIKey)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./cmd/config/ -run TestConfigInitSavesEmptyBaseURL -v`
Expected: FAIL — 当前 init 读取 2 个输入，但 stdin 只提供 1 个，导致 base_url 消费了唯一的输入

- [ ] **Step 3: 实现修改 — 删除 base_url 交互逻辑**

修改 `cmd/config/config.go` 的 `runConfigInit` 函数，从：

```go
func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入 API 地址 (base_url): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	fmt.Println("\n请按以下步骤获取 API Key:")
	fmt.Println("1. 访问公司 SaaS 平台并登录")
	fmt.Println("2. 进入个人设置 -> API 密钥")
	fmt.Println("3. 复制 API Key")
	fmt.Print("\n请粘贴 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	cfg := &internalconfig.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
```

改为：

```go
func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("请按以下步骤获取 API Key:")
	fmt.Println("1. 访问公司 SaaS 平台并登录")
	fmt.Println("2. 进入个人设置 -> API 密钥")
	fmt.Println("3. 复制 API Key")
	fmt.Print("\n请粘贴 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	cfg := &internalconfig.Config{
		APIKey: apiKey,
	}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./cmd/config/ -run TestConfigInitSavesEmptyBaseURL -v`
Expected: PASS

- [ ] **Step 5: 运行全部测试确认无回归**

Run: `go test ./cmd/config/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 清理未使用的 import**

删除 `runConfigInit` 不再使用 `strings` 包的情况下，检查 import 是否仍需要。`runConfigSet` 中未使用 `strings`，但 `strings` 在其他地方也未使用 — 实际检查后，`strings` 仅在已删除的 `strings.TrimSpace(baseURL)` 和保留的 `strings.TrimSpace(apiKey)` 中使用，因此 import 保留不变。

- [ ] **Step 7: Commit**

```bash
git add cmd/config/config.go cmd/config/config_test.go
git commit -m "fix(config): remove base_url prompt from config init

base_url is now auto-resolved via ResolveBaseURL() using the
compile-time environment default. Users can still override with
config set base_url <value>."
```
