# 环境配置默认 base_url 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 根据编译时 Environment 变量自动选择对应环境的默认 base_url，用户无需手动输入

**Architecture:** config 包新增 envBaseURLs map 和 ResolveBaseURL() 方法，cmd/root.go 的 SetEnvironment 转发给 config 包，createClient 调用 ResolveBaseURL()

**Tech Stack:** Go, TDD

**Spec:** docs/superpowers/specs/2026-04-02-environment-config-design.md

---

### Task 1: config 包新增 DefaultBaseURL 和 ResolveBaseURL

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: 写失败测试 — DefaultBaseURL 和 ResolveBaseURL**

在 `internal/config/config_test.go` 末尾追加：

```go
func TestDefaultBaseURL_Development(t *testing.T) {
	SetEnvironment("development")
	got := DefaultBaseURL()
	want := "https://kpapi-cs.ckjr001.com/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestDefaultBaseURL_Production(t *testing.T) {
	SetEnvironment("production")
	got := DefaultBaseURL()
	want := "https://kpapi0.kw.ckjr.cn/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestDefaultBaseURL_UnknownFallback(t *testing.T) {
	SetEnvironment("unknown")
	got := DefaultBaseURL()
	want := "https://kpapi-cs.ckjr001.com/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestResolveBaseURL_WithBaseURL(t *testing.T) {
	cfg := &Config{BaseURL: "https://custom.example.com/api"}
	got := cfg.ResolveBaseURL()
	if got != cfg.BaseURL {
		t.Errorf("ResolveBaseURL() = %s, want %s", got, cfg.BaseURL)
	}
}

func TestResolveBaseURL_EmptyBaseURL(t *testing.T) {
	SetEnvironment("production")
	cfg := &Config{BaseURL: ""}
	got := cfg.ResolveBaseURL()
	want := "https://kpapi0.kw.ckjr.cn/api"
	if got != want {
		t.Errorf("ResolveBaseURL() = %s, want %s", got, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config/ -run "TestDefaultBaseURL|TestResolveBaseURL" -v`
Expected: FAIL — SetEnvironment/DefaultBaseURL/ResolveBaseURL 未定义

- [ ] **Step 3: 实现 envBaseURLs、SetEnvironment、DefaultBaseURL、ResolveBaseURL**

在 `internal/config/config.go` 中 `ErrConfigNotFound` 定义之前追加：

```go
// envBaseURLs 各环境对应的默认 base_url
var envBaseURLs = map[string]string{
	"development": "https://kpapi-cs.ckjr001.com/api",
	"production":  "https://kpapi0.kw.ckjr.cn/api",
}

// environment 由 main 包通过 SetEnvironment 注入
var environment string

// SetEnvironment 注入编译时的 Environment 值
func SetEnvironment(env string) {
	environment = env
}

// DefaultBaseURL 根据当前 environment 返回对应的默认 base_url
func DefaultBaseURL() string {
	if u, ok := envBaseURLs[environment]; ok {
		return u
	}
	return envBaseURLs["development"]
}

// ResolveBaseURL 返回最终使用的 base_url
// 优先级: config.json 中的 base_url > DefaultBaseURL()
func (c *Config) ResolveBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL()
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/config/ -run "TestDefaultBaseURL|TestResolveBaseURL" -v`
Expected: PASS

- [ ] **Step 5: 运行全量 config 测试**

Run: `go test ./internal/config/ -v`
Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add DefaultBaseURL and ResolveBaseURL for environment-based base_url"
```

---

### Task 2: cmd/root.go 接入 ResolveBaseURL

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: cmd.SetEnvironment 转发给 config 包**

在 `cmd/root.go` 的 `SetEnvironment` 函数中追加 `internalconfig.SetEnvironment(e)` 调用：

```go
func SetEnvironment(e string) {
	environment = e
	internalconfig.SetEnvironment(e)
}
```

- [ ] **Step 2: createClient 使用 ResolveBaseURL**

将 `cmd/root.go:120` 的：
```go
return api.NewClient(cfg.BaseURL, cfg.APIKey), nil
```
改为：
```go
return api.NewClient(cfg.ResolveBaseURL(), cfg.APIKey), nil
```

- [ ] **Step 3: 编译验证**

Run: `go build ./cmd/ckjr-cli/`
Expected: 编译成功

- [ ] **Step 4: 运行全量测试**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/root.go
git commit -m "feat(cmd): use ResolveBaseURL for environment-based default base_url"
```
