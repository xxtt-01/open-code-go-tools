# ocgt - Claude Code Native GUI Control Panel & Proxy

> 🌐 **[简体中文版 (Chinese Version)](../README.md)**

`ocgt` (OpenCode Go Tools) is a native desktop control center custom-built for **Claude Code** and the **OpenCode Go** (opencode.ai) subscription service. It integrates an ultra-low latency local compatibility proxy (converting Anthropic-format requests to OpenAI Chat Completions protocol) and provides a clean, intuitive native GUI control panel with **one-click English/Chinese switching**.

Developers do not need to manually configure tedious environment variables or modify system hosts in the command line. Simply double-click to open the client, and the agent environment will be automatically hosted with one-click terminal invocation.

---

## 🖥️ Core Features & Showcases

### 📊 System Status Dashboard
![System Status](../assets/2026-05-30_213807.png)
* **Real-time Monitoring**: Monitor local proxy listening ports (default `127.0.0.1:8787`) and upstream API node status.
* **Quick Access**: Visualize local configuration file paths and open the configuration directory with one click.

### ⚙️ Premium Configuration Management
![Configuration Settings](../assets/2026-05-30_213821.png)
* **Optimized for OpenCode Go**: Enter your API Key and hit save to apply hot-reloaded configurations in seconds.
* **Model Alias Mapping**: Custom-map Claude Sonnet, Haiku, and Opus models to upstream alternatives seamlessly.
* **Reasoning Intensity**: Provide fixed Reasoning (Thinking) Budget settings (Fast, Slow, Deep, Geek, Off) to eliminate configuration mistakes.

### 💻 One-Click Terminal Activation
![Terminal Activation](../assets/2026-05-30_213831.png)
* **Automatic Injection**: Select your preferred console type (PowerShell / Bash / CMD) and click **"Launch Pre-configured Terminal"** to spawn a native shell session with all proxy variables injected.
* **Instant Coding**: Directly type `claude` and press Enter to start your AI coding session!
* **External Integration**: Provide quick environment variables and CC Switch JSON configurations for existing terminal sessions or IDE windows.

### 📡 Traffic Radar Logs
* **Real-time Capture**: Real-time capture and visualization of API request logs, latency, methods, and status codes originating from the Claude Code client, summarized with success rates and average response latency.

---

## 🚦 Three-Step Quick Start

1. **Download & Launch**: Go to [Releases](../../releases) to download the native executable for your system (e.g. `ocgt_v2.0.0.exe` for Windows) and double-click to run.
2. **Save Settings**: In the **"Configuration"** page, fill in your **OpenCode Go API Key**, choose a default model and reasoning strength, and click **"Save & Hot-Reload"**.
3. **Launch Terminal**: Under the **"Terminal"** tab, select your shell, click **"Launch Pre-configured Terminal"**, and type:
   ```bash
   claude
   ```

*(Note: You only need to choose and start one shell type, no need to configure all of them.)*

---

## 📁 Configurations & Hot Reload

Configuration preferences are persisted locally:
```text
%USERPROFILE%\.ocgt\config.json
```
Thanks to the background **Hot Reload** mechanism, manually editing this JSON file externally will automatically reload the configuration in the local proxy server within 2.5 seconds without needing a client restart.

---

## 💻 Advanced CLI Reference

Although using the visual GUI is highly recommended, `ocgt` also provides basic CLI options for command-line convenience:

```powershell
ocgt init       # Initialize the default configuration file
ocgt serve      # Run the local proxy silently in the background
ocgt claude-env # Print proxy env variables for the current active profile
ocgt ccswitch   # Output provider JSON for import into CC Switch routers
ocgt version    # Print the current running version
```

---

## 🛠️ Build & Development

Requires Go 1.22+ and Wails CLI for compilation.

**Install Wails CLI**:
```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

**Run Development Mode**:
```powershell
wails dev
```

**Build Production Executable**:
```powershell
.\build.bat
```

---

## 📄 License

This project is licensed under the **MIT License**.
