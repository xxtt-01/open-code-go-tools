# open-code-go-tools

`ocgt` 是一个轻量级的 Claude API 兼容性代理工具，用于连接 Claude Code 和 OpenCode Go API。

## 架构概览

```text
Claude Code -> CC Switch -> ocgt 本地代理 -> OpenCode Go 官方 API
```

目标是让 Claude Code 能够使用 OpenCode Go 的订阅服务，同时保持用户设置的简单性。CC Switch 只需要配置一个 Anthropic 兼容的提供商指向 `ocgt`，`ocgt` 负责处理 OpenCode Go 的协议转换。

## 为什么需要这个工具

OpenCode Go 目前提供两种官方 API 接口：

- Anthropic Messages: `https://opencode.ai/zen/go/v1/messages`
- OpenAI 兼容的 Chat Completions: `https://opencode.ai/zen/go/v1/chat/completions`
- 模型列表: `https://opencode.ai/zen/go/v1/models`

Claude Code 使用 Anthropic Messages 协议。OpenCode Go 大部分模型使用 Chat Completions 协议，而 MiniMax M2.5/M2.7 是 Messages 协议模型。因此最有效的架构是本地兼容性代理：

- **MiniMax M2.5/M2.7**: 直接转发到 `/v1/messages`
- **Kimi, GLM, DeepSeek, Qwen, MiMo**: 将 Anthropic Messages 转换为 `/v1/chat/completions`
- 响应转换回 Claude Code 期望的 Anthropic Messages 格式
- 流式文本从 OpenAI SSE 块转换为 Anthropic SSE 事件
- 工具调用通过使用非流式上游调用桥接，然后本地发出 Anthropic SSE 事件
- DeepSeek V4 思考模式兼容性通过禁用转发调用的思考功能并回放 `reasoning_content` 来处理
- `GET /v1/models` 被规范化以供 Claude Code 和 CC Switch 模型发现

## 快速开始

### 1. 编译

```powershell
cd D:\Project\open-code-go-tools
go build -o .\bin\ocgt.exe .\cmd\ocgt
```

### 2. 初始化配置

```powershell
.\bin\ocgt.exe init
```

默认配置文件路径：`%USERPROFILE%\.ocgt\config.json`

### 3. 设置 API 密钥

```powershell
# 临时设置（当前 PowerShell 会话）
$env:OPENCODE_GO_API_KEY = "your-opencode-go-key"

# 永久设置（推荐）
.\bin\ocgt.exe key set "your-opencode-go-key"
```

永久设置后，重启 PowerShell 或执行以下命令加载：

```powershell
$env:OPENCODE_GO_API_KEY = [Environment]::GetEnvironmentVariable("OPENCODE_GO_API_KEY", "User")
```

### 4. 启动代理服务

```powershell
.\bin\ocgt.exe serve
```

本地 Anthropic 兼容的 Base URL：`http://127.0.0.1:8787`

### 5. 配置 Claude Code

在另一个终端中：

```powershell
.\bin\ocgt.exe claude-env
```

执行输出的 PowerShell 命令，然后启动 Claude Code：

```powershell
claude
```

关键环境变量：

```powershell
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8787"
$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"
$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: opencode-go"
$env:CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS = "1"
$env:CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY = "1"
$env:ANTHROPIC_MODEL = "kimi-k2.6"
```

## 配置说明

### 默认配置

```json
{
  "listen": "127.0.0.1:8787",
  "upstream": "https://opencode.ai/zen/go",
  "active_profile": "opencode-go",
  "profiles": {
    "opencode-go": {
      "api_key_env": "OPENCODE_GO_API_KEY",
      "default_model": "kimi-k2.6",
      "model_aliases": {
        "deepseek": "deepseek-v4-pro",
        "flash": "deepseek-v4-flash",
        "glm": "glm-5.1",
        "glm5": "glm-5",
        "hy3": "hy3-preview",
        "kimi": "kimi-k2.6",
        "kimi25": "kimi-k2.5",
        "mimo": "mimo-v2.5-pro",
        "mimo25": "mimo-v2.5",
        "minimax": "minimax-m2.7",
        "qwen35": "qwen3.5-plus",
        "qwen": "qwen3.6-plus"
      },
      "message_models": [
        "minimax-m2.5",
        "minimax-m2.7"
      ]
    }
  }
}
```

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `listen` | 本地代理监听地址 | `127.0.0.1:8787` |
| `upstream` | OpenCode Go API 上游地址 | `https://opencode.ai/zen/go` |
| `active_profile` | 默认使用的配置 profile | `opencode-go` |
| `api_key_env` | API Key 环境变量名称 | `OPENCODE_GO_API_KEY` |
| `default_model` | 默认模型 | `kimi-k2.6` |
| `model_aliases` | 模型别名映射 | 见上表 |
| `message_models` | 使用 Messages 端点的模型 | `minimax-m2.5`, `minimax-m2.7` |

## 命令列表

| 命令 | 说明 |
|------|------|
| `ocgt init` | 创建配置文件 |
| `ocgt serve` | 启动本地代理服务 |
| `ocgt profiles` | 列出已配置的 profiles |
| `ocgt models` | 显示本地模型别名或使用 `--remote` 查询官方模型列表 |
| `ocgt claude-env` | 打印 Claude Code 环境变量配置 |
| `ocgt ccswitch` | 打印 CC Switch 提供商配置片段 |
| `ocgt key set <key>` | 保存 API Key 到用户环境变量 |
| `ocgt key show` | 显示当前设置的 API Key |
| `ocgt version` | 显示版本号 |

## 支持的模型

### OpenCode Go 官方模型

- `glm-5`
- `glm-5.1`
- `kimi-k2.5`
- `kimi-k2.6`
- `mimo-v2.5`
- `mimo-v2.5-pro`
- `minimax-m2.5`
- `minimax-m2.7`
- `qwen3.5-plus`
- `qwen3.6-plus`
- `deepseek-v4-pro`
- `deepseek-v4-flash`
- `hy3-preview` (如果 OpenCode Go 返回)

### 模型别名

| 别名 | 映射模型 |
|------|----------|
| `deepseek` | `deepseek-v4-pro` |
| `flash` | `deepseek-v4-flash` |
| `glm` | `glm-5.1` |
| `glm5` | `glm-5` |
| `hy3` | `hy3-preview` |
| `kimi` | `kimi-k2.6` |
| `kimi25` | `kimi-k2.5` |
| `mimo` | `mimo-v2.5-pro` |
| `mimo25` | `mimo-v2.5` |
| `minimax` | `minimax-m2.7` |
| `qwen35` | `qwen3.5-plus` |
| `qwen` | `qwen3.6-plus` |

## CC Switch 配置

生成提供商配置片段：

```powershell
.\bin\ocgt.exe ccswitch --profile opencode-go
```

在 CC Switch 中使用以下值：

- **Provider type**: Anthropic
- **Base URL**: `http://127.0.0.1:8787`
- **API key**: `ocgt-local-proxy`
- **Model**: `kimi-k2.6`, `glm-5.1`, `deepseek-v4-pro`, `qwen3.6-plus`, `minimax-m2.7` 等
- **Custom header**: `X-Ocgt-Profile: opencode-go`

### 推荐的 CC Switch 映射

| 角色 | 菜单名称 | 实际模型 |
|------|----------|----------|
| Opus | Kimi K2.6 | `kimi-k2.6` |
| Opus | Kimi K2.5 | `kimi-k2.5` |
| Sonnet | GLM-5.1 | `glm-5.1` |
| Sonnet | GLM-5 | `glm-5` |
| Sonnet | Hy3 Preview | `hy3-preview` |
| Sonnet | Qwen3.6 Plus | `qwen3.6-plus` |
| Sonnet | Qwen3.5 Plus | `qwen3.5-plus` |
| Sonnet | DeepSeek V4 Pro | `deepseek-v4-pro` |
| Haiku | DeepSeek V4 Flash | `deepseek-v4-flash` |
| Sonnet | MiMo V2.5 Pro | `mimo-v2.5-pro` |
| Sonnet | MiMo V2.5 | `mimo-v2.5` |
| Sonnet | MiniMax M2.7 | `minimax-m2.7` |
| Haiku | MiniMax M2.5 | `minimax-m2.5` |

除非 OpenCode Go 明确记录支持 1M 上下文，否则不要勾选 1M 声明。

## 使用限制

OpenCode Go 使用基于美元价值窗口的使用限制：

- 5 小时限制
- 周限制
- 月限制

官方文档说明当前使用情况在 OpenCode 控制台中追踪。在此项目更新时，没有公开的剩余配额 API，因此 `ocgt` 不会模拟余额端点。

## 兼容性说明

`ocgt` 旨在覆盖所有官方 OpenCode Go API 协议：

- `/v1/models`
- `/v1/messages`
- `/v1/messages/count_tokens`
- `/v1/chat/completions`（在适配器后面）

某些 Claude Code 的测试版功能可能不受非 Anthropic 模型端点支持。生成的环境设置了 `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1` 以保持请求更接近公共协议层面。

## 项目结构

```
open-code-go-tools/
├── cmd/
│   └── ocgt/
│       └── main.go          # 主程序入口
├── internal/
│   ├── config/
│   │   ├── config.go        # 配置管理
│   │   └── config_test.go   # 配置测试
│   └── proxy/
│       ├── types.go         # 类型定义和 Server 构造
│       ├── converter.go     # Anthropic↔OpenAI 协议转换
│       ├── streamer.go      # SSE 流式响应处理
│       ├── handler.go       # HTTP 路由和处理逻辑
│       ├── helpers.go       # 工具函数
│       └── proxy_test.go    # 代理测试
├── bin/
│   └── ocgt.exe             # 编译后的可执行文件
├── go.mod                   # Go 模块定义
├── Makefile                 # 构建和开发命令
├── LICENSE                  # MIT 许可证
└── README.md                # 项目说明文档
```

## 技术细节

### 协议转换流程

1. **Claude Code 请求** -> Anthropic Messages 格式
2. **ocgt 代理** -> 根据模型类型决定路由：
   - Messages 模型 -> 直接转发到 `/v1/messages`
   - Chat Completions 模型 -> 转换为 OpenAI 格式并转发到 `/v1/chat/completions`
3. **OpenCode Go API** -> 返回响应
4. **ocgt 代理** -> 将响应转换回 Anthropic Messages 格式
5. **Claude Code** <- 接收标准 Anthropic 响应

### 支持的特性

- ✅ 流式响应 (Streaming)
- ✅ 工具调用 (Tool Calling)
- ✅ 多模型支持
- ✅ 模型别名映射
- ✅ 自定义 Headers
- ✅ 环境变量配置
- ✅ 配置文件管理

## 参考文档

- [OpenCode Go 文档](https://dev.opencode.ai/docs/go/)
- [Claude Code 环境变量](https://code.claude.com/docs/en/env-vars)
- [Claude Code 模型配置](https://code.claude.com/docs/en/model-config)
- [CC Switch](https://cc-switch.cc/en)

## 许可证

MIT License
