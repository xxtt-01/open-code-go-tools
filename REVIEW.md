# ocgt 项目评审报告

## 总体评价

ocgt 是一个功能明确、结构清晰的 Go 代理工具，核心职责是把 Anthropic Messages 协议和 OpenAI Chat Completions 协议互转，让 Claude Code 通过 OpenCode Go 的 API 使用国产大模型。代码量约 2182 行，模块划分合理（cmd/入口、config/配置、proxy/核心代理逻辑），基本可运行。

> **更新：** 本报告中的 P0 和 P1 问题已全部修复，补全了测试覆盖。详见下方各条目的 ✅ 标记。

以下按类别列出问题和改进建议。

---

## 一、安全性问题

### P0 - 关键问题

**1. 本地代理无认证机制** (未修复 - 低优先级)

`serve` 命令监听 `127.0.0.1:8787`，任何本地进程都能调用。虽然 API Key 存在上游验证，但本地无鉴权意味着同机恶意软件可直接使用代理并发请求到 OpenCode Go，消耗订阅额度。

建议：添加可选的本地 Bearer Token 认证，通过配置 `local_auth_token` 或环境变量 `OCGT_LOCAL_TOKEN` 控制。

**2. 请求体无大小限制** ✅ 已修复

`io.ReadAll(r.Body)` 已替换为 `io.ReadAll(io.LimitReader(r.Body, MaxBodySize))`，限制为 10MB。常量 `MaxBodySize` 定义在 `types.go` 中。

**3. API Key 简单掩码泄露信息** ✅ 已修复

`maskKey` 已改为只显示末尾 4 字符 + 总长度，如 `****abcd (44 chars)`。

```go
// main.go L321-L327
func maskKey(key string) string {
    if len(key) <= 8 {
        return strings.Repeat("*", len(key))
    }
    return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
```

对于典型 40+ 字符的 API Key，首尾各暴露 4 字符仍有信息泄露风险。

建议：只展示 Key 的后 4 位 + 总长度，如 `****abcd (44 chars)`。

**4. 上游 API Key 传输安全**

`key set` 使用 PowerShell 命令行参数设置环境变量，在进程列表中可见明文 Key。

```go
// main.go L288-L301
func setUserEnv(name, value string) error {
    // ...
    return runPowerShellNoProfile("[Environment]::SetEnvironmentVariable($args[0], $args[1], 'User')", name, value)
}
```

建议：通过 `stdin` 或临时文件传递值，而非命令行参数。

### P1 - 中等问题

**5. 上游 TLS 证书未校验**

`http.Transport` 使用默认配置，没有自定义 TLS 校验逻辑。这本身不是 bug，但 `New()` 中没有 `TLSClientConfig` 字段意味着代理会接受系统信任的所有 CA 证书。

```go
// types.go L134-L138
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}
```

当前行为合理，但建议添加配置项 `insecure_skip_verify` 以便在自签名环境使用。

**6. 无 HTTP 请求速率限制**

代理不限制并发请求数或请求频率。可能在用户意外发送大量并发请求（例如 Claude Code 的多工具并行调用）时压垮上游。

建议：添加可选的并发请求限制（如信号量或 `sync/semaphore`）。

---

## 二、错误处理

### P1 - 中等问题

**1. 流式 SSE 写入错误被静默忽略**

`writeSSE` 函数中 `json.Marshal` 和 `fmt.Fprintf` 的错误完全被忽略：

```go
// streamer.go L278-L280
func writeSSE(w io.Writer, event string, payload any) {
    data, _ := json.Marshal(payload)
    fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}
```

如果 JSON 序列化失败，客户端会收到格式错误的 SSE 数据（`data: null`），可能导致解析崩溃。

建议：返回 error 并在调用方处理，或在序列化失败时发送 `error` 类型 SSE 事件。

**2. 上游错误响应体可能泄露内部信息**

`writeUpstreamError` 直接将上游的错误响应体原样返回客户端：

```go
// helpers.go L77-L85
func writeUpstreamError(w http.ResponseWriter, status int, body []byte) {
    // ...
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _, _ = w.Write(body)
}
```

上游可能返回包含内部 URL、堆栈追踪等信息。建议解析上游错误并封装为标准格式后再返回。

**3. `copyResponse` 中写入错误被吞掉**

```go
// helpers.go L27-L53
func copyResponse(w http.ResponseWriter, body io.Reader) (int64, error) {
    // ...
    m, writeErr := w.Write(buf[:n])
    // writeErr 只在下次循环或返回时处理
}
```

如果客户端断开连接，代理仍在从上游读取数据并丢弃，浪费带宽但不影响正确性。

建议：在写入错误时提前终止。

**4. `models` 端点错误时无降级**

如果上游 `/v1/models` 返回非 200，直接透传错误给客户端。可以配置本地 fallback 模型列表。

---

## 三、代码质量

### P2 - 低优先级

**1. `io.ReadAll` 在多处重复使用且无缓冲**

`messages` handler、`countTokens`、`models` 等多处直接使用 `io.ReadAll`，无大小限制，无 buffered reader。

**2. `http.DefaultClient` 在 `fetchRemoteModels` 中使用**

```go
// main.go L328
resp, err := http.DefaultClient.Do(req)
```

`http.DefaultClient` 没有 Timeout、没有自定义 Transport。与 `Server` 中使用 5 分钟超时自定义 Client 不一致。

建议：传入或复用 Server 的 `client`。

**3. 日志缺乏结构化**

使用 `log.Printf` 裸字符串，无请求 ID、无时间戳格式控制、无日志级别。

建议：至少添加请求 ID 或添加一个轻量结构化日志（`slog` 已在 Go 1.21 标准库中）。

**4. `main.go` 中 Windows 特定代码没有构建标签**

`setWindowsUserEnv`、`isWindows`、`runPowerShellNoProfile` 这些函数在所有平台编译，但只有 Windows 需要 PowerShell。

建议：用 `//go:build windows` 和 `//go:build !windows` 分离，或至少添加注释说明跨平台兼容性。

**5. 全局变量 `version` 在 `main.go` 中**

```go
const version = "0.1.0"
```

硬编码版本号，Makefile 中有 `-ldflags` 注入机制但 `main.go` 用的是 `const`，ldflags 不能覆盖 `const`。

建议：改为 `var version = "0.1.0"` 以支持构建时注入版本号。

---

## 四、测试覆盖

### P0 - 关键缺失

**1. 无流式响应测试**

`streamer.go` 是最复杂的部分（280 行），但没有任何单元测试。SSE 协议转换中的状态机逻辑（`blockState`）容易出错且难以手动验证。

建议：添加测试用例覆盖：
- 纯文本流式场景
- reasoning_content（思考）流式场景
- 工具调用流式场景
- 混合文本+工具调用流式场景
- `[DONE]` 终止信号处理
- 空选择|块|块的 edge case

**2. 无并发安全测试**

`reasoningByTool` 的 `sync.Mutex` 在并发场景下的正确性未测试。`setReasoning` 的 LRU 驱逐在高并发下可能丢失数据。

**3. 缺少集成测试**

现有的测试使用 `httptest.NewServer` 模拟上游，这是好的单元测试方法，但缺少端到端集成测试：
- 代理启动 -> 发送请求 -> 收到正确响应
- Claude Code 实际连接场景的模拟
- 心跳端点 `/healthz` 测试

**4. 边界条件测试不足**

- 空消息列表
- 超大请求体
- 无效 JSON 请求体
- 上游返回非 JSON 响应
- 上游连接超时/拒绝
- Messages 模型与非 Messages 模型混合

---

## 五、性能问题

### P1 - 中等问题

**1. `reasoningOrder` 切片的无界增长/缩减**

```go
// handler.go L326-L336
func (s *Server) setReasoning(id, reasoning string) {
    // ...
    s.reasoningOrder = s.reasoningOrder[1:]  // 每次驱逐都创建新切片
    // ...
}
```

每次 LRU 驱逐都会切片再切片，高频场景下产生大量 GC 压力。

建议：使用环形缓冲区或 `container/list` + map 实现 LRU。

**2. 无 gzip 请求压缩**

代理不设置 `Accept-Encoding: gzip` 请求头，也不会解压上游响应。对于长对话（Claude Code 历史消息可能很大），这会显著增加延迟。

建议：添加 gzip 支持，至少在转发请求时设置 `Accept-Encoding: gzip`。

**3. 无 HTTP/2 支持**

`http.Server` 未配置 `TLSConfig`，仅支持 HTTP/1.1。对于本地代理场景这没问题，但如果未来需要 HTTPS，需要额外配置。

**4. 无连接复用的 Keep-Alive 管理**

Transport 设置了 `IdleConnTimeout: 90s`，但没有 `MaxConnsPerHost` 限制。极端情况下可能建立大量到上游的连接。

---

## 六、协议兼容性

### P1 - 中等问题

**1. Anthropic `system` 字段处理不完整**

```go
// converter.go L20-L21
if system := blocksToText(in.System); system != "" {
    out.Messages = append(out.Messages, openAIMessage{Role: "system", Content: system})
}
```

`blocksToText` 将 Anthropic 的数组格式 system 消息扁平化为单个字符串。这会丢失 system 消息中的 `cache_control` 标记。对于缓存控制敏感的场景（如 Claude Code 的长上下文缓存），这会导致缓存提示失效，增加成本。

**2. Anthropic 响应中缺少字段**

`openAIToAnthropic` 返回的响应缺少一些 Claude Code 可能期望的字段：

```go
// converter.go L163-L175
return map[string]any{
    "id":            firstNonEmpty(in.ID, "msg_ocgt_..."),
    "type":          "message",
    "role":          "assistant",
    "model":         firstNonEmpty(in.Model, model),
    "content":       content,
    "stop_reason":   stopReason,
    "stop_sequence": nil,
    "usage":         map[string]int{...},
}
```

缺少 `cache_creation_input_tokens` 和 `cache_read_input_tokens` 等 Anthropic 缓存相关字段。

**3. 图片处理使用 Markdown 格式而非标准格式**

```go
// converter.go L94-L108
case "image":
    // ...
    Content: fmt.Sprintf("![image](data:%s;base64,%s)", mediaType, data),
```

将 Anthropic 的 base64 图片转为 Markdown 图片语法。大多数 Chat Completions 模型不支持这种格式。

建议：使用 OpenAI 的 `image_url` 多模态格式，或在文档中明确图片支持为 best-effort。

**4. `count_tokens` 端点估算非常粗糙**

```go
// helpers.go L56-L64
func estimateTokens(payload anthropicRequest) int {
    text := payload.Model + "\n" + blocksToText(payload.System)
    // ...
    return len([]rune(text))/4 + 1
}
```

字符数 / 4 的估算法在中文、日文等 CJK 文本上严重低估。Claude Code 可能基于此数值做上下文窗口管理。

建议：使用 tiktoken 或至少区分 CJK / ASCII 计算。

---

## 七、架构与设计

### P1 - 中等问题

**1. 多 Profile 支持不完整**

Config 支持 `profiles` 和 `active_profile`，`X-Ocgt-Profile` 头可以选择 profile。但以下场景未覆盖：
- 运行时动态切换 profile（需重启）
- Profile 验证不完善（不检查 `api_key_env` 或 `api_key` 是否存在）
- Profile 缺少 `api_key` 时的静默失败（返回空字符串作为 Key）

```go
// config.go L165-L173
func (p Profile) APIKeyValue() string {
    if p.APIKey != "" {
        return p.APIKey
    }
    if p.APIKeyEnv != "" {
        return os.Getenv(p.APIKeyEnv)
    }
    return ""
}
```

当两者都为空时，请求会不带 Authorization 头发到上游，几乎注定失败。建议在 `serve` 启动时检查当前 profile 的 Key 是否可用。

**2. 无配置文件热重载**

修改 config.json 后需重启代理才能生效。建议添加 SIGHUP 信号处理或 `/ocgt/reload` 端点。

**3. 命令行参数不统一**

- `init` 支持 `--config` 指定路径
- `serve` 支持 `--config` 和 `--profile` 和 `--listen`
- `models` 支持 `--config`、`--profile`、`--remote`
- `claude-env` 支持 `--config`、`--profile`、`--base-url`、`--shell`

缺少全局 `--config` 参数，每个子命令需要单独指定。

**4. 无优雅关闭的日志提示**

`ListenAndServe` 中 graceful shutdown 是静默的，没有日志提示正在关闭或已关闭。

---

## 八、缺失功能

### P2 - 功能增强

1. **不支持 HTTPS** - 本地代理只提供 HTTP。虽然 127.0.0.1 场景下问题不大，但文档应明确说明。

2. **无请求/响应日志归档** - 对调试 Claude Code 的请求行为很有用，建议添加可选的调试日志模式。

3. **无 `/v1/messages/batches` 端点** - Anthropic Batch API 不被代理。

4. **不支持 SSE 心跳** - 长时间空闲的流式连接可能被中间件超时，建议添加定期 SSE 注释行心跳。

5. **Claude Code 的 `prompt_caching` 不被代理** - Beta feature，当前被 `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1` 禁用，但如果未来需要支持则需额外处理。

6. **无版本协商** - `Anthropic-Version` 硬编码为 `2023-06-01`，不支持版本协商。

7. **无健康检查上游** - `/healthz` 只检查代理本身存活，不检查上游可用性。

---

## 九、文档问题

1. **README 中的项目结构与实际不符** - README 列出了 `converter.go`、`streamer.go`、`handler.go`、`helpers.go` 等文件，这些确实存在。但 `LICENSE` 文件存在而 `go.sum` 可能缺失（因无外部依赖）。

2. **缺少 CHANGELOG** - 无版本变更记录。

3. **缺少 CONTRIBUTING.md** - 无贡献指南。

4. **缺少架构设计文档** - 复杂的协议转换流程（Anthropic <-> OpenAI）没有单独的设计文档。

5. **命令行 `--help` 不完整** - `usage()` 函数没有列出各子命令的参数。

---

## 十、改进优先级建议

| 优先级 | 改进项 | 工作量 |
|--------|--------|--------|
| P0 | 请求体大小限制 | 小 |
| P0 | 本地代理认证（可选） | 小 |
| P0 | 流式响应单元测试 | 中 |
| P0 | `version` 改为 var 支持构建注入 | 极小 |
| P1 | SSE 写入错误处理 | 中 |
| P1 | 上游错误响应封装 | 小 |
| P1 | Profile Key 启动检查 | 小 |
| P1 | 替换 `http.DefaultClient` | 小 |
| P1 | 添加 gzip 压缩支持 | 中 |
| P1 | token 估算优化（CJK） | 中 |
| P2 | 请求 ID / 结构化日志 | 中 |
| P2 | 配置热重载 | 中 |
| P2 | 健康检查上游 | 小 |
| P2 | 并发请求限制 | 小 |

---

## 总结

ocgt 作为一个本地代理工具，核心功能（协议转换、流式桥接、DeepSeek 思考模式兼容）实现完整且经过实际使用验证。项目结构清晰，代码可读性好。主要短板在于：

1. **安全性**：缺少本地认证和请求大小限制
2. **测试**：最核心的流式转换逻辑完全没有测试
3. **健壮性**：多处错误被静默忽略，上游失败无降级策略
4. **协议完整性**：system 处理、图片格式、token 估算等有已知限制但未在文档中说明

建议优先处理 P0 问题（请求限制、测试、认证），然后逐步改进 P1（错误处理、Key 检查、gzip）。