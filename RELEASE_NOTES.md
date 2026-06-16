# Release Notes

## 🌐 语言选择 / Language
* [简体中文 (Simplified Chinese)](#-ocgt-v205---v205-)
* [English](#-ocgt-v205---release-notes)

---

# 🇨🇳 ocgt v2.0.5

## 修复

- **修复鉴权头回归（严重）**：v2.0.4 的 `applyAnthropicAuth` 会无条件删除上游请求的 `Authorization: Bearer` 头、仅替换为 `X-Api-Key` 头。而默认上游 `opencode.ai/zen/go`（OpenAI 兼容网关）依赖 `Authorization: Bearer` 认证，导致所有 `/v1/models`、`/v1/messages`、`/v1/messages/count_tokens`、`/v1/chat/completions` 请求返回 401
- **新增配置驱动的鉴权方式**：`Profile` 新增 `auth_mode` 字段，取值 `bearer`（默认）/ `x-api-key` / `both`。`applyAnthropicAuth` 按配置选择鉴权头策略：
  - `bearer`（默认）：保留 `Authorization: Bearer`，不发送 `X-Api-Key`，适配 OpenAI 兼容网关（如 opencode.ai/zen/go）
  - `x-api-key`：删除 Bearer，发送 `X-Api-Key` + `Anthropic-Version`，适配真正的 Anthropic 官方 API / new-api 风格网关
  - `both`：两者都发（兼容兜底，少数场景）
- **向后兼容**：老配置无 `auth_mode` 字段时默认按 `bearer` 处理，行为与 v2.0.4 之前一致，零迁移
- 参考了 [fuergaosi233/claude-code-proxy](https://github.com/fuergaosi233/claude-code-proxy)、[Calcium-Ion/new-api](https://github.com/Calcium-Ion/new-api)、[LiteLLM](https://github.com/BerriAI/litellm/issues/19618) 等业界项目的鉴权头处理方案
- **新增测试覆盖**：`TestApplyAnthropicAuthModes`（6 子用例，覆盖三种模式 + 默认值 + 大小写归一化 + 非法值回退）、`TestEffectiveAuthMode`（7 子用例）

---

# 🇺🇸 ocgt v2.0.5 - Release Notes

## Fixes

- **Fixed auth header regression (critical)**: `applyAnthropicAuth` in v2.0.4 unconditionally deleted the upstream `Authorization: Bearer` header and replaced it with only `X-Api-Key`. Since the default upstream `opencode.ai/zen/go` (an OpenAI-compatible gateway) authenticates via `Authorization: Bearer`, all `/v1/models`, `/v1/messages`, `/v1/messages/count_tokens`, and `/v1/chat/completions` requests returned 401
- **Added config-driven auth mode**: `Profile` now has an `auth_mode` field accepting `bearer` (default) / `x-api-key` / `both`. `applyAnthropicAuth` selects the header strategy accordingly:
  - `bearer` (default): keep `Authorization: Bearer`, do not send `X-Api-Key` — for OpenAI-compatible gateways (e.g. opencode.ai/zen/go)
  - `x-api-key`: drop Bearer, send `X-Api-Key` + `Anthropic-Version` — for the genuine Anthropic API / new-api style gateways
  - `both`: send both (compatibility fallback, rare)
- **Backward compatible**: old configs without `auth_mode` default to `bearer`, behaving exactly as before v2.0.4 — no migration needed
- Referenced auth-header handling from [fuergaosi233/claude-code-proxy](https://github.com/fuergaosi233/claude-code-proxy), [Calcium-Ion/new-api](https://github.com/Calcium-Ion/new-api), and [LiteLLM](https://github.com/BerriAI/litellm/issues/19618)
- **New test coverage**: `TestApplyAnthropicAuthModes` (6 sub-cases: three modes + default + case normalization + unknown fallback), `TestEffectiveAuthMode` (7 sub-cases)

---

# 历史版本 / Previous Releases

## v2.0.4

### 修复 / Fixes
- **`/v1/models` 端点缺少认证头**：查询模型列表时未附带 `X-Api-Key` / `Anthropic-Version` 头，导致部分上游网关返回 401
- **`/v1/chat/completions` 转发缺少 `X-Api-Key`**：通过 chat/completions 路径转发请求时仅携带 `Authorization: Bearer`，未附加 `X-Api-Key`
- ⚠️ 注意：v2.0.4 的实现存在回归（无条件删除 Bearer），已在 v2.0.5 修复

---

# 🇨🇳 ocgt v2.0.4

## 修复

- **`/v1/models` 端点缺少认证头**：查询模型列表时未附带 `X-Api-Key` / `Anthropic-Version` 头，导致部分上游网关返回 401。修复后与 `/v1/messages` 端点认证行为一致
- **`/v1/chat/completions` 转发缺少 `X-Api-Key`**：通过 chat/completions 路径转发请求时仅携带 `Authorization: Bearer`，未附加 `X-Api-Key`。部分上游（如 opencode.ai 网关）要求 `X-Api-Key` 头，缺失会导致 401 认证失败。修复后同时发送 `X-Api-Key` 和 `Anthropic-Version` 头
- **新增测试覆盖**：补充 `TestChatCompletionsEndpointUsesAnthropicAuth` 测试，确保 chat/completions 路径的认证头行为正确

---

# 🇺🇸 ocgt v2.0.4 - Release Notes

## Fixes

- **Missing auth headers on `/v1/models`**: Model list queries were not sending `X-Api-Key` / `Anthropic-Version` headers, causing 401 errors on some upstream gateways. Fixed to match `/v1/messages` auth behavior
- **Missing `X-Api-Key` on `/v1/chat/completions`**: Requests forwarded via chat/completions path only carried `Authorization: Bearer` without `X-Api-Key`. Some upstreams (e.g., opencode.ai gateway) require `X-Api-Key`, resulting in 401 auth failures. Fixed to send both `X-Api-Key` and `Anthropic-Version` headers
- **New test coverage**: Added `TestChatCompletionsEndpointUsesAnthropicAuth` to verify correct auth header behavior on chat/completions path

---

# 历史版本 / Previous Releases

## v2.0.3

### 修复 / Fixes
- **费用估算双重计费（严重）**：`EstimateCost` 对缓存读取的 tokens 既按全价计费又按缓存价计费，导致有缓存的请求费用虚高约 2-23 倍
- **`extractUsageFromAnthropicStream` 缺字段**：只解析 `message_delta` 事件，遗漏了 `message_start` 中的 `input_tokens` / cache 字段
- **重试导致请求次数虚高**：每次重试失败都写入历史记录，导致 1 次用户请求在统计中最多被计为 6 次
- **`modelBreakdown` 缓存命中率误含写入**：命中率分子使用了 `CacheRead + CacheCreation`，修复后只使用 `CacheRead`
- **流式 `message_delta` 缺 `input_tokens`**：OpenAI → Anthropic 协议转换的合成 `message_delta` 未包含 `input_tokens`
- **流量界面选择"今日"时间窗口错误**：使用 `time.Now()` 导致显示为近 24h 而非当日数据

## v2.0.2 — 流量监控 / 额度看板 / 客户端集成 / 多巴胺配色

## v2.0.1 — ccswitch / claude-desktop-env CLI 增强

## v2.0.0 — 原生双语控制面板发布 / Premium Bilingual Desktop Control Panel
