import os

def replace_in_file(path, replacements):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    for old, new in replacements:
        content = content.replace(old, new)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)

replace_in_file('README.md', [
    ('gui_status.png', '2026-05-30_213807.png'),
    ('gui_config.png', '2026-05-30_213821.png'),
    ('gui_terminal.png', '2026-05-30_213831.png'),
    ('ocgt_v0.2.0.exe', 'ocgt_v2.0.0.exe')
])

# RELEASE_NOTES.md needs to be updated with new version content.
with open('RELEASE_NOTES.md', 'r', encoding='utf-8') as f:
    rn_content = f.read()

new_rn = """# Release Notes - v2.0.0

## 🌐 语言选择 / Language
* [简体中文 (Simplified Chinese)](#-ocgt-v200---重大更新版本发布)
* [English](#-ocgt-v200---major-update-release)

---

# 🇨🇳 ocgt v2.0.0 - 重大更新版本发布

本次 2.0.0 版本包含全面的系统升级，完整的 MD 文档更新与最新的控制面板截图。
支持多系统（Windows, macOS, Linux）架构。
- 移除了旧版的 AI 生成界面截图，全面替换为最新的实机截图。
- 同步最新的 OpenCode Go 工作流。
- 更多底层稳定性改进。

---

# 🇺🇸 ocgt v2.0.0 - Major Update Release

This 2.0.0 release includes comprehensive system upgrades, fully updated MD documentation, and the latest control panel screenshots.
Supports multi-OS (Windows, macOS, Linux) architectures.
- Removed old AI-generated screenshots, replaced with the latest actual screenshots.
- Synced with the newest OpenCode Go workflow.
- More underlying stability improvements.

---

""" + rn_content.replace('v0.1.9', 'v2.0.0')

with open('RELEASE_NOTES.md', 'w', encoding='utf-8') as f:
    f.write(new_rn)
