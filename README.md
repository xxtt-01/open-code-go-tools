# ocgt - Claude Code 本地代理与 GUI 控制面板

`ocgt`（OpenCode Go Tools）是面向 Claude Code 的本地兼容代理。它把 Claude Code 的 Anthropic Messages 请求转换为 OpenAI Chat Completions 风格上游可接收的请求，并提供一个 Wails 原生 GUI，用于保存 OpenCode Go API Key、配置模型映射、修复 Claude Code 环境变量、拉起已配置终端、查看实时请求日志。

当前 GUI 版本：`v0.1.7`。

## 这版解决了什么

- 修复 GUI 中 “Thinking Token 上限” 标签乱码问题，改为更不容易误配的“思考强度”下拉选项。
- 支持 Claude Code 的 `MAX_THINKING_TOKENS` / `CLAUDE_CODE_DISABLE_THINKING` 工作流。
- 透传并转换上游返回的 `reasoning_content`、`thinking`、`reasoning_details` 等思考字段，能显示则显示，不伪造思考过程。
- 为思考预算设置安全上限，默认 `512`，可选关闭、快速、均衡、深度、强力。
- 缩短 GUI 初始代理状态等待时间，避免打开后长时间空白等待。
- 增加 Wails 单实例锁，重复双击会唤起已有窗口，而不是开多个进程等半天才显示。
- 构建脚本在没有全局 `wails` CLI 时会自动使用 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`。
- Release workflow 从 tag 注入版本号，并上传真实构建产物。

## Claude Code 接入方式

推荐从 GUI 操作：

1. 下载并运行最新 Release 中的 `ocgt-windows-amd64.exe`。
2. 在“配置管理”里填写 OpenCode Go API Key，选择默认模型和 Sonnet/Haiku/Opus 映射。
3. 选择“思考强度”：
   - 快速：低延迟，适合简单问题。
   - 均衡：默认推荐。
   - 深度：适合复杂编码和分析。
   - 强力：适合重构、排错、长链路任务。
   - 关闭思考：设置 `CLAUDE_CODE_DISABLE_THINKING=1`。
4. 点击“一键修复 Claude Code 系统环境变量”，清理旧的 CC Switch / Claude Code 残留变量。
5. 在“终端启动”页拉起配置好的终端，进入终端后运行 `claude`。

GUI 会设置这些关键变量：

```text
ANTHROPIC_BASE_URL=http://127.0.0.1:8787
ANTHROPIC_API_KEY=ocgt-local-proxy
ANTHROPIC_MODEL=<当前默认模型>
ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile: <当前 profile>
MAX_THINKING_TOKENS=<思考强度对应预算>
```

关闭思考时额外设置：

```text
CLAUDE_CODE_DISABLE_THINKING=1
```

## 为什么有时看不到思考过程

Claude Code 是否显示思考过程，取决于三件事：

1. Claude Code 当前版本是否展示上游返回的 thinking / reasoning 内容。
2. 上游模型是否真的返回 reasoning 字段。
3. 代理是否能把上游字段转换为 Anthropic 流式 thinking block。

`ocgt` 现在做的是兼容转换：如果上游返回了真实 reasoning，就转成 Claude Code 能识别的 thinking；如果上游没有返回，`ocgt` 不会伪造思考内容。

## 快速开始

1. 到 [Releases](../../releases) 下载最新 GUI 可执行文件。
2. Windows 用户通常下载 `ocgt-windows-amd64.exe`。
3. 双击运行，程序默认监听 `127.0.0.1:8787`。
4. 在 GUI 中保存 API Key 和模型配置。
5. 点击环境变量修复和终端启动，然后在新终端运行 `claude`。

## 配置文件

默认路径：

```text
%USERPROFILE%\.ocgt\config.json
```

核心配置示例：

```json
{
  "listen": "127.0.0.1:8787",
  "upstream": "https://opencode.ai/zen/go",
  "request_timeout_seconds": 300,
  "max_thinking_budget_tokens": 512,
  "active_profile": "opencode-go",
  "profiles": {
    "opencode-go": {
      "api_key_env": "OPENCODE_GO_API_KEY",
      "default_model": "kimi-k2.6",
      "model_aliases": {
        "opus": "kimi-k2.6",
        "sonnet": "qwen3.6-plus",
        "haiku": "deepseek-v4-flash"
      }
    }
  }
}
```

`max_thinking_budget_tokens` 支持：

- `-1`：关闭思考。
- `0`：不主动设置预算。
- `1..8192`：设置最大思考 token 预算。

GUI 为了降低误配概率，只暴露固定档位。

## 开发与构建

需要 Go 1.22+。没有全局 Wails CLI 也可以直接使用仓库脚本：

```powershell
go test ./...
go build ./...
.\build.bat
```

输出示例：

```text
build\bin\ocgt_v0.1.7.exe
```

手动构建 GUI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails build -clean -ldflags "-X github.com/ethan-blue/open-code-go-tools/internal/version.Version=0.1.7"
```

## CLI

```powershell
ocgt init
ocgt serve
ocgt models
ocgt claude-env
ocgt ccswitch
ocgt version
```

`ocgt ccswitch` 会输出可导入 CC Switch 的 provider JSON。建议只保留一个 `ocgt-*` provider，删除旧的 astron / 其他历史 provider，避免 Claude Code 的 `/model` 菜单继续显示旧选项。

## Release

推送 `v*` tag 会触发 GitHub Actions：

- 构建 Windows、macOS、Linux GUI 二进制。
- 从 tag 注入运行时版本号。
- 上传平台产物和 `checksums.txt`。
- 自动创建 GitHub Release。

本地也可以用 GitHub CLI 手动创建 Release：

```powershell
gh release create v0.1.7 build\bin\ocgt_v0.1.7.exe --title "v0.1.7" --notes-file RELEASE_NOTES.md
```

## License

MIT
