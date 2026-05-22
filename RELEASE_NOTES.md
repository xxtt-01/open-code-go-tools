# ocgt v0.1.8

这版重点修复 Claude Code 接入体验、思考过程兼容、启动等待和发布流程。

## 重点变化

- GUI 中的 “Thinking Token 上限” 改为“思考强度”下拉选项，避免乱码和误配。
- 支持 `MAX_THINKING_TOKENS` 与 `CLAUDE_CODE_DISABLE_THINKING`，和 Claude Code 的现有工作流保持一致。
- 将上游真实返回的 `reasoning_content`、`thinking`、`reasoning_details` 等字段转换为 Anthropic thinking stream block。
- 默认 thinking budget 收敛到 `512`，并支持关闭、快速、均衡、深度、强力档位。
- 缩短 GUI 初始代理状态等待时间，减少打开后长时间空白。
- 增加 Wails 单实例锁，重复双击会唤起已有窗口。
- 去掉外部字体依赖，减少首次打开时的网络等待。
- 修复用户可见乱码，包括 GUI 标签、产品名、README 和评审文档。
- 修复 release workflow：固定 Wails CLI 版本、从 tag 注入版本号、按实际构建产物上传。

## 验证

- `go test ./...`
- `go build ./...`
- `.\build.bat`

Windows 本地产物：

```text
build\bin\ocgt_v0.1.8.exe
```
