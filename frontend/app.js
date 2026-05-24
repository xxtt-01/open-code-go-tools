// ── ocgt Dashboard ──
// Refactored: modular structure, bug fixes, proper i18n coverage

// ══════════════════════════════════════════════════════
// §1 — Constants & Global State
// ══════════════════════════════════════════════════════

const APP_VERSION = 'v0.1.9';
const DEFAULT_CLOSE_BEHAVIOR = 'prompt';
const CLOSE_BEHAVIORS = new Set(['prompt', 'minimize', 'exit']);
const ALLOWED_THINKING_BUDGETS = ['256', '512', '1024', '2048', '-1'];

let API_BASE = 'http://127.0.0.1:8787';
let systemStatus = null;
let currentShell = 'powershell';
let proxyReady = false;
let currentLang = localStorage.getItem('lang') || 'zh';
let originalSettingsValues = {};
let LOCAL_AUTH_TOKEN = '';
let isLoadingDashboard = true;
let isInitializing = false;

// ══════════════════════════════════════════════════════
// §2 — i18n Dictionary
// ══════════════════════════════════════════════════════

const i18n = {
    zh: {
        nav_dashboard: "系统状态",
        nav_settings: "配置管理",
        nav_terminal: "终端启动",
        nav_history: "流量监控",
        status_running: "代理运行中",
        status_connecting: "代理连接中",
        status_online: "代理已连接",
        status_offline: "代理未连接",
        status_api_key_configured: "已配置",
        status_api_key_not_configured: "未配置",
        status_model_unset: "未设定",
        status_not_configured: "未配置",
        status_saving: "保存中...",
        status_success: "已保存 ✓",
        service_normal: "服务正常",
        service_connecting: "服务连接中",
        service_offline: "服务离线",
        title_dashboard: "系统状态监控",
        subtitle_dashboard: "查看当前代理服务运行指标与后台状态",
        title_settings: "一键配置管理中心",
        subtitle_settings: "快速设置您的 API 密钥与高阶 Claude 模型代理映射",
        title_terminal: "一键控制台激活",
        subtitle_terminal: "一键启动已载入代理环境变量的原生命令行窗口",
        title_history: "流量雷达监控",
        subtitle_history: "实时捕获并通过仪表盘统计来自 Claude Code 的 API 请求日志",
        lbl_listen: "监听地址",
        lbl_upstream: "上游 API 节点",
        lbl_timeout: "请求超时",
        lbl_api_key: "API Key 状态",
        lbl_profile: "当前活跃 Profile",
        lbl_model: "默认解析模型",
        lbl_config_path: "本地配置文件路径",
        lbl_last_updated: "刚刚更新",
        btn_open_folder: "打开所在文件夹",
        sett_title: "一键配置管理中心",
        sett_section_api: "API Configuration",
        sett_section_model: "Model Settings",
        sett_section_prefs: "Application Preferences",
        sett_profile: "当前配置 Profile",
        sett_default_model: "全局默认模型 (Default Model)",
        sett_api_key: "OpenCode Go API Key (代理密钥)",
        placeholder_api_key: "请输入您的 sk-... 密钥",
        sett_timeout: "请求超时检查 (秒，1-3600)",
        sett_thinking: "思考强度（支持模型生效）",
        opt_thinking_256: "快速 · 低延迟",
        opt_thinking_512: "慢速 · 强力",
        opt_thinking_1024: "深度 · 复杂任务",
        opt_thinking_2048: "极客 · 强力重构",
        opt_thinking_off: "关闭思考",
        sett_mapping_title: "Claude 模型映射设置 (Real-time Alias Mapping)",
        sett_mapping_sonnet: "Sonnet 映射目标",
        sett_mapping_haiku: "Haiku 映射目标",
        sett_mapping_opus: "Opus 映射目标",
        opt_custom: "-- 自定义模型 --",
        btn_save_config: "保存并热重载配置",
        btn_repair_env: "一键修复 Claude Code 系统环境变量",
        btn_reset_defaults: "重置为默认值",
        btn_about_app: "关于 ocgt 控制面板 (About App)",
        btn_clear_history: "清除历史记录",
        hint_save: "保存后同步更新 config.json、热重载 Go 服务；思考强度会按模型能力转换，不支持的模型不会发送额外 thinking 字段",
        hint_tip: "💡 提示：在\"终端启动\"页中只需选择并一键启动任意一种您习惯的命令行窗口即可，无需重复配置多种终端。",
        hint_changes_detected: "检测到未保存的更改",
        term_title: "一键唤醒代理控制台",
        term_shell_type: "目标命令行类型",
        btn_launch_term: "一键拉起配置终端 (Launch)",
        btn_persistent_env: "修复以后所有新终端环境变量",
        hint_launch: "一键注入当前 Profile 代理变量并打开原生 shell。直接打 <code>claude</code> 即可开始运行！",
        guide_title: "💡 快捷运行极简指南",
        guide_1: "在上方选项卡选择您常用的命令终端。",
        guide_2: "点击 <b>\"一键拉起配置终端\"</b>，系统会自动唤醒控制台。",
        guide_3: "直接在拉起的窗口中键入 <code>claude</code> 即可启动 AI 代码对话。",
        guide_4: "（可选）若要在已有终端中工作，可点击右侧的复制按钮导入配置。",
        guide_5: "<b>提示</b>：终端类型只需选择并一键启动任意一个即可，无需全部配置或启动。",
        code_env_title: "Claude Code 环境变量 (Env Setup)",
        code_ccswitch_title: "CC Switch 提供商配置 (JSON Import)",
        btn_copy: "复制",
        btn_copied: "已复制 ✓",
        traf_total: "总吞吐请求量",
        traf_rate: "请求成功率",
        traf_latency: "平均响应延时",
        th_time: "时间",
        th_method: "方法",
        th_path: "路由路径",
        th_model: "解析模型",
        th_status: "状态码",
        th_duration: "耗时",
        th_error: "错误原因",
        traf_empty: "暂无流量记录。请使用一键终端或在其他 Shell 中向代理发送请求...",
        traf_listening: "实时流量雷达持续监听中",
        opt_model_kimi_26: "kimi-k2.6 (推荐 - Kimi 旗舰)",
        opt_model_qwen_36: "qwen3.6-plus (通义千问高效)",
        opt_model_deepseek_pro: "deepseek-v4-pro (深度求索通用)",
        opt_model_deepseek_flash: "deepseek-v4-flash (极速模型)",
        opt_model_glm_51: "glm-5.1 (智谱旗舰)",
        opt_model_hy3_preview: "hy3-preview (混元预览)",
        opt_mapping_sonnet_default: "qwen3.6-plus (推荐平替)",
        opt_mapping_haiku_default: "deepseek-v4-flash (推荐极速)",
        opt_mapping_opus_default: "kimi-k2.6 (推荐长文本)",
        sett_close_behavior: "关闭窗口时的行为 (Close Window Behavior)",
        opt_close_prompt: "每次询问我 (Prompt Every Time)",
        opt_close_minimize: "直接最小化到系统托盘 (Minimize to Tray)",
        opt_close_exit: "彻底退出程序 (Exit Directly)",
        close_dialog_title: "关闭窗口",
        close_dialog_msg: "选择您希望的操作方式：",
        close_dialog_exit: "彻底退出程序",
        close_dialog_minimize: "最小化到系统托盘",
        close_dialog_cancel: "取消",
        about_desc: "专为 Claude Code 与 OpenCode Go 打造的极简桌面控制面板与代理",
        about_author: "作者",
        about_license: "许可证",
        about_project: "项目地址",
        about_close: "关闭",
        err_api_key_required: "请输入 API Key",
        err_timeout_range: "超时必须在 1-3600 秒之间",
        toast_saved: "配置已保存并热重载",
        toast_save_failed: "保存失败",
        toast_env_repaired: "环境变量已修复并写入系统",
        toast_env_repair_failed: "环境变量修复失败",
        toast_copy_success: "已复制到剪贴板",
        toast_copy_failed: "复制失败",
        toast_profile_changed: "Profile 已切换",
        toast_launch_failed: "终端启动失败",
        toast_launch_success: "终端已成功启动",
        toast_history_cleared: "历史记录已清除",
        toast_validation_error: "请检查表单中的错误",
        toast_custom_model_prompt: "请输入自定义模型名称",
        toast_reset_confirm: "确定要重置所有设置为默认值吗？",
        toast_reset_done: "设置已重置为默认值",
        toast_confirm: "确认重置",
        // Terminal launch states
        term_launching: "启动中...",
        term_launched: "已启动终端 ✓",
        // Desktop-only warnings
        warn_desktop_only_launch: "一键启动终端仅在桌面版 app 客户端可用，请在桌面端中点击使用！",
        warn_desktop_only_env: "该功能仅桌面端可用。当前浏览器模式请复制右侧环境变量手动执行。",
        warn_desktop_only_folder: "该功能仅在桌面客户端可用。您的配置文件夹通常在您的个人用户目录下的 .ocgt 文件夹中。",
        // Env repair states
        env_repairing: "修复中...",
        env_repaired_hint: "已修复，重新打开终端生效",
        // Connection status with unconfigured key
        status_connected_no_key: "代理已连接，密钥未配置",
        // Open folder errors
        err_open_folder: "打开失败",
        err_open_folder_generic: "无法打开文件夹",
        // Footer
        footer_text: "ocgt \u00A9 2026 \u00B7 MIT Licensed \u00B7 Official OpenCode Go Companion Center",
        // Preferences popover
        pref_title: "偏好设置",
        pref_language: "界面语言",
        pref_appearance: "外观",
        pref_appearance_desc: "主题模式与界面语言",
        pref_theme: "主题模式",
        pref_theme_light: "浅色",
        pref_theme_dark: "深色",
        pref_theme_system: "跟随系统",
        pref_behavior: "行为",
        pref_behavior_desc: "关闭窗口与系统交互",
        pref_danger: "重置与关于",
        pref_danger_desc: "恢复默认设置或查看版本信息"
    },
    en: {
        nav_dashboard: "Status",
        nav_settings: "Configuration",
        nav_terminal: "Terminal",
        nav_history: "Traffic Radar",
        status_running: "Proxy Running",
        status_connecting: "Connecting...",
        status_online: "Connected",
        status_offline: "Disconnected",
        status_api_key_configured: "Configured",
        status_api_key_not_configured: "Unconfigured",
        status_model_unset: "Unset",
        status_not_configured: "Not configured",
        status_saving: "Saving...",
        status_success: "Saved ✓",
        service_normal: "Normal",
        service_connecting: "Connecting...",
        service_offline: "Offline",
        title_dashboard: "System Status Radar",
        subtitle_dashboard: "Monitor real-time proxy metrics and server status",
        title_settings: "Configuration Center",
        subtitle_settings: "Manage your upstream API keys, timeouts, and Claude model aliases",
        title_terminal: "One-Click Terminal Activator",
        subtitle_terminal: "Launch pre-configured shell terminals with proxy environments loaded",
        title_history: "Traffic Monitoring Radar",
        subtitle_history: "Real-time capture of API logs and metrics received from Claude Code",
        lbl_listen: "Listen Address",
        lbl_upstream: "Upstream Node",
        lbl_timeout: "Request Timeout",
        lbl_api_key: "API Key Status",
        lbl_profile: "Active Profile",
        lbl_model: "Default Model",
        lbl_config_path: "Local Config Path",
        lbl_last_updated: "Updated just now",
        btn_open_folder: "Open Directory",
        sett_title: "Easy Configuration Center",
        sett_section_api: "API Configuration",
        sett_section_model: "Model Settings",
        sett_section_prefs: "Application Preferences",
        sett_profile: "Current Profile",
        sett_default_model: "Global Default Model",
        sett_api_key: "OpenCode Go API Key",
        placeholder_api_key: "Enter your OpenCode sk-... API Key",
        sett_timeout: "Request Timeout (Seconds, 1-3600)",
        sett_thinking: "Reasoning Intensity (Supported Models)",
        opt_thinking_256: "Fast · Low Latency",
        opt_thinking_512: "Slow · Powerful",
        opt_thinking_1024: "Deep · Complex Tasks",
        opt_thinking_2048: "Geek · Heavy Refactoring",
        opt_thinking_off: "Disable Reasoning",
        sett_mapping_title: "Claude Model Alias Mapping",
        sett_mapping_sonnet: "Claude Sonnet Mapping",
        sett_mapping_haiku: "Claude Haiku Mapping",
        sett_mapping_opus: "Claude Opus Mapping",
        opt_custom: "-- Custom Model --",
        btn_save_config: "Save & Hot-Reload",
        btn_repair_env: "One-click Repair Claude Code System Env",
        btn_reset_defaults: "Reset to defaults",
        btn_about_app: "About ocgt Dashboard",
        btn_clear_history: "Clear history",
        hint_save: "Saves config and hot-reloads the proxy; reasoning controls are only sent to models with compatible request schemas",
        hint_tip: "💡 Tip: Just select and launch any terminal shell of your choice. No need to repeatedly configure all shells.",
        hint_changes_detected: "Unsaved changes detected",
        term_title: "Spawn Pre-configured Console",
        term_shell_type: "Target Shell / Console Type",
        btn_launch_term: "Launch Pre-configured Terminal",
        btn_persistent_env: "Repair System Env (Persistent for future shells)",
        hint_launch: "Injects proxy environment variables and spawns a native shell. Directly run <code>claude</code> to begin!",
        guide_title: "💡 Quick Start Guide",
        guide_1: "Select your preferred shell type in the tabs above.",
        guide_2: "Click \"Launch Pre-configured Terminal\" to summon the console.",
        guide_3: "Directly type <code>claude</code> and press Enter inside the shell to start coding!",
        guide_4: "(Optional) Copy env variables on the right if using an existing IDE terminal.",
        guide_5: "Note: You only need to choose and start one shell type, no need to configure all of them.",
        code_env_title: "Claude Code Env Variables",
        code_ccswitch_title: "CC Switch Provider Config (JSON Import)",
        btn_copy: "Copy",
        btn_copied: "Copied ✓",
        traf_total: "Total Requests",
        traf_rate: "Success Rate",
        traf_latency: "Average Latency",
        th_time: "Time",
        th_method: "Method",
        th_path: "Request Path",
        th_model: "Resolved Model",
        th_status: "Status",
        th_duration: "Duration",
        th_error: "Error Details",
        traf_empty: "No traffic captured yet. Launch a terminal or make API requests through the proxy...",
        traf_listening: "Live Traffic Radar Active & Listening",
        opt_model_kimi_26: "kimi-k2.6 (Recommended - Kimi Flagship)",
        opt_model_qwen_36: "qwen3.6-plus (Qwen High Efficiency)",
        opt_model_deepseek_pro: "deepseek-v4-pro (DeepSeek Universal)",
        opt_model_deepseek_flash: "deepseek-v4-flash (Flash Speed)",
        opt_model_glm_51: "glm-5.1 (GLM Flagship)",
        opt_model_hy3_preview: "hy3-preview (Hunyuan Preview)",
        opt_mapping_sonnet_default: "qwen3.6-plus (Recommended)",
        opt_mapping_haiku_default: "deepseek-v4-flash (Recommended)",
        opt_mapping_opus_default: "kimi-k2.6 (Recommended Long Context)",
        sett_close_behavior: "Close Window Behavior",
        opt_close_prompt: "Prompt Every Time",
        opt_close_minimize: "Minimize to System Tray",
        opt_close_exit: "Exit Application",
        close_dialog_title: "Close Window",
        close_dialog_msg: "How would you like to close the app?",
        close_dialog_exit: "Exit Application",
        close_dialog_minimize: "Minimize to System Tray",
        close_dialog_cancel: "Cancel",
        about_desc: "Premium native companion for Claude Code & OpenCode Go",
        about_author: "Author",
        about_license: "License",
        about_project: "Project",
        about_close: "Close",
        err_api_key_required: "API Key is required",
        err_timeout_range: "Timeout must be 1-3600 seconds",
        toast_saved: "Configuration saved & hot-reloaded",
        toast_save_failed: "Save failed",
        toast_env_repaired: "Environment variables written to system",
        toast_env_repair_failed: "Environment repair failed",
        toast_copy_success: "Copied to clipboard",
        toast_copy_failed: "Copy failed",
        toast_profile_changed: "Profile switched",
        toast_launch_failed: "Terminal launch failed",
        toast_launch_success: "Terminal launched successfully",
        toast_history_cleared: "History cleared",
        toast_validation_error: "Please check form errors",
        toast_custom_model_prompt: "Enter custom model name",
        toast_reset_confirm: "Reset all settings to defaults?",
        toast_reset_done: "Settings reset to defaults",
        toast_confirm: "Confirm Reset",
        term_launching: "Launching...",
        term_launched: "Terminal Launched ✓",
        warn_desktop_only_launch: "One-click launch is only available in the desktop app!",
        warn_desktop_only_env: "Only available in desktop app. Please copy the env variables manually on the right.",
        warn_desktop_only_folder: "Only available in the desktop client. Config is typically under ~/.ocgt directory.",
        env_repairing: "Repairing...",
        env_repaired_hint: "Repaired! Reopen terminals to apply",
        status_connected_no_key: "Connected, API Key Unconfigured",
        err_open_folder: "Open failed",
        err_open_folder_generic: "Cannot open folder",
        footer_text: "ocgt \u00A9 2026 \u00B7 MIT Licensed \u00B7 Official OpenCode Go Companion Center",
        pref_title: "Preferences",
        pref_language: "Language",
        pref_appearance: "Appearance",
        pref_appearance_desc: "Theme mode and interface language",
        pref_theme: "Theme",
        pref_theme_light: "Light",
        pref_theme_dark: "Dark",
        pref_theme_system: "System",
        pref_behavior: "Behavior",
        pref_behavior_desc: "Window close and system interaction",
        pref_danger: "Reset & About",
        pref_danger_desc: "Reset defaults or view version info"
    }
};

// ══════════════════════════════════════════════════════
// §3 — Utility Helpers
// ══════════════════════════════════════════════════════

/** Get the current language dictionary */
function t(key) {
    const dict = i18n[currentLang];
    return (dict && dict[key]) || key;
}

/** Safely access the Wails App binding. Returns null when not in desktop mode. */
function getWailsApp() {
    return (window.go && window.go.main && window.go.main.App) || null;
}

/** Call a Wails App method if available, returns null otherwise. */
async function callWails(method, ...args) {
    const app = getWailsApp();
    if (!app || typeof app[method] !== 'function') return null;
    return app[method](...args);
}

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function delay(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }

async function apiFetch(path, options, timeoutMs) {
    options = options || {};
    timeoutMs = timeoutMs || 8000;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    try {
        // Add auth token header if available
        const headers = options.headers || {};
        if (LOCAL_AUTH_TOKEN) {
            headers['X-Ocgt-Local-Token'] = LOCAL_AUTH_TOKEN;
        }
        return await fetch(`${API_BASE}${path}`, {
            ...options,
            headers,
            signal: controller.signal
        });
    } finally {
        clearTimeout(timeout);
    }
}

function normalizeCloseBehavior(value) {
    return CLOSE_BEHAVIORS.has(value) ? value : DEFAULT_CLOSE_BEHAVIOR;
}

function padTwo(n) { return n.toString().padStart(2, '0'); }
// ══════════════════════════════════════════════════════

// Lazily cached DOM references (populated in bootstrap)
const dom = {};

function cacheDom() {
    // Dashboard
    dom.elListen = document.getElementById('status-listen');
    dom.elUpstream = document.getElementById('status-upstream');
    dom.elProfile = document.getElementById('status-profile');
    dom.elModel = document.getElementById('status-model');
    dom.elConfigPath = document.getElementById('status-config-path');
    dom.elTimeout = document.getElementById('status-timeout');
    dom.elApiKey = document.getElementById('status-api-key');
    dom.dashboardSkeletons = document.getElementById('dashboard-skeletons');
    dom.dashboardContent = document.getElementById('dashboard-content');

    // Settings
    dom.selectProfile = document.getElementById('profile-select');
    dom.inputApiKey = document.getElementById('api-key-input');
    dom.inputTimeout = document.getElementById('timeout-input');
    dom.inputThinkingBudget = document.getElementById('thinking-budget-input');
    dom.inputDefaultModel = document.getElementById('default-model-input');
    dom.inputSonnetMapping = document.getElementById('mapping-sonnet-input');
    dom.inputHaikuMapping = document.getElementById('mapping-haiku-input');
    dom.inputOpusMapping = document.getElementById('mapping-opus-input');
    dom.inputCloseBehavior = document.getElementById('close-behavior-input');
    dom.btnSaveAllConfig = document.getElementById('save-all-config-btn');
    dom.btnInstallEnv = document.getElementById('install-env-btn');
    dom.btnInstallEnvTerminal = document.getElementById('install-env-terminal-btn');
    dom.btnToggleVisibility = document.getElementById('toggle-key-visibility');
    dom.settingsForm = document.getElementById('settings-form');
    dom.configActions = document.getElementById('config-actions');
    dom.resetDefaultsBtn = document.getElementById('reset-defaults-btn');
    dom.btnAboutApp = document.getElementById('about-app-btn');

    // Terminal
    dom.btnLaunchTerminal = document.getElementById('launch-terminal-btn');
    dom.shellTabs = document.getElementById('shell-tabs');
    dom.codeEnv = document.getElementById('env-code-block');
    dom.codeCCSwitch = document.getElementById('ccswitch-code-block');
    dom.btnCopyEnv = document.getElementById('copy-env-btn');
    dom.btnCopyCCSwitch = document.getElementById('copy-ccswitch-btn');

    // History
    dom.tbodyHistory = document.getElementById('history-tbody');
    dom.clearHistoryBtn = document.getElementById('clear-history-btn');

    // Header & footer
    dom.statusPill = document.getElementById('statusPill');
    dom.uptimeBadge = document.querySelector('.uptime-badge');
    dom.lastUpdated = document.getElementById('lastUpdated');
    dom.toastContainer = document.getElementById('toastContainer');
    dom.footerText = document.getElementById('footer-text');

    // Preferences trigger
    dom.prefsToggleBtn = document.getElementById('prefsToggleBtn');
    dom.prefLangSelect = document.getElementById('pref-lang-select');

    // Version stamps
    dom.appVersion = document.getElementById('app-version');
    dom.aboutVersion = document.getElementById('about-version');

    // Modals
    dom.closeDialogOverlay = document.getElementById('closeDialogOverlay');
    dom.closeDialogExit = document.getElementById('closeDialogExit');
    dom.closeDialogMinimize = document.getElementById('closeDialogMinimize');
    dom.closeDialogCancel = document.getElementById('closeDialogCancel');
    dom.aboutDialogOverlay = document.getElementById('aboutDialogOverlay');
    dom.aboutDialogClose = document.getElementById('aboutDialogClose');
}
// ══════════════════════════════════════════════════════

const TOAST_ICONS = {
    success: '<svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>',
    error: '<svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
    warning: '<svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
    info: '<svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
};

function toast(message, type, options) {
    type = type || 'info';
    options = options || {};
    const duration = options.duration || (type === 'error' ? 5000 : 3500);
    const actionCallback = options.actionCallback || null;
    const actionLabel = options.actionLabel || '';

    const el = document.createElement('div');
    el.className = `toast toast-${type}`;

    let html = TOAST_ICONS[type] || TOAST_ICONS.info;
    html += `<span class="toast-msg">${escapeHtml(message)}</span>`;

    if (actionCallback) {
        html += `<button class="toast-action">${escapeHtml(actionLabel)}</button>`;
    }
    html += `<button class="toast-close" aria-label="Close notification"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>`;

    el.innerHTML = html;

    let activeTimer = null;
    const dismiss = () => {
        if (el.classList.contains('toast-out')) return;
        if (activeTimer) { clearTimeout(activeTimer); activeTimer = null; }
        el.classList.add('toast-out');
        el.addEventListener('animationend', () => { if (el.parentNode) el.remove(); }, { once: true });
    };

    el.querySelector('.toast-close').addEventListener('click', dismiss);
    const actionBtn = el.querySelector('.toast-action');
    if (actionBtn && actionCallback) {
        actionBtn.addEventListener('click', () => { actionCallback(); dismiss(); });
    }

    // Timer management: properly cancel previous timer on re-enter
    activeTimer = setTimeout(dismiss, duration);
    el.addEventListener('mouseenter', () => {
        if (activeTimer) { clearTimeout(activeTimer); activeTimer = null; }
    });
    el.addEventListener('mouseleave', () => {
        activeTimer = setTimeout(dismiss, 2000);
    });

    dom.toastContainer.appendChild(el);
    return el;
}

function toastI18n(key, type, options) {
    return toast(t(key), type, options);
}
// ══════════════════════════════════════════════════════

function showModal(overlayEl) {
    if (!overlayEl) return;
    overlayEl.classList.add('active');
    overlayEl.setAttribute('aria-hidden', 'false');
}
function hideModal(overlayEl) {
    if (!overlayEl) return;
    overlayEl.classList.remove('active');
    overlayEl.setAttribute('aria-hidden', 'true');
}
// ══════════════════════════════════════════════════════

function setProxyConnectionState(state) {
    const meta = {
        connecting: { text: t('status_connecting'), className: 'connecting' },
        online:     { text: t('status_online'),     className: 'online' },
        offline:    { text: t('status_offline'),     className: 'offline' }
    }[state];
    if (!meta) return;

    [dom.statusPill, dom.uptimeBadge].forEach(el => {
        if (!el) return;
        el.classList.remove('online', 'offline', 'connecting');
        el.classList.add(meta.className);
        const textSpan = el.querySelector('span:last-child');
        if (textSpan) textSpan.textContent = meta.text;
    });
}

function showDashboardContent() {
    if (dom.dashboardSkeletons) dom.dashboardSkeletons.classList.add('hidden');
    if (dom.dashboardContent) dom.dashboardContent.classList.remove('hidden');
    isLoadingDashboard = false;
}

async function resolveApiBase() {
    try {
        const addr = await callWails('GetListenAddress');
        if (addr) API_BASE = `http://${addr}`;
    } catch (err) { console.error('Wails GetListenAddress error:', err); }
}

async function waitForProxyReady(timeoutMs) {
    timeoutMs = timeoutMs || 2500;
    const started = Date.now();
    while (Date.now() - started < timeoutMs) {
        try {
            const resp = await apiFetch('/healthz', { cache: 'no-store' }, 700);
            if (resp.ok) return true;
        } catch (_) { /* retry */ }
        await delay(350);
    }
    return false;
}

async function initializeApp() {
    if (isInitializing) return;
    isInitializing = true;
    setProxyConnectionState('connecting');
    await resolveApiBase();

    // Fetch local auth token from Wails (silently fails in browser mode)
    try { const t = await callWails('GetLocalToken'); if (t) LOCAL_AUTH_TOKEN = t; } catch (_) {}

    proxyReady = await waitForProxyReady();
    if (!proxyReady) {
        setProxyConnectionState('offline');
        showDashboardContent();
        isInitializing = false;
        return;
    }
    setProxyConnectionState('online');
    try {
        await Promise.all([loadStatus(), loadProfiles(), loadHistory(), loadPreferences()]);
    } catch (err) {
        console.error('Error during initial load:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
    } finally {
        isInitializing = false;
    }
}

function updateLastUpdated() {
    if (!dom.lastUpdated) return;
    const now = new Date();
    const timeStr = `${padTwo(now.getHours())}:${padTwo(now.getMinutes())}:${padTwo(now.getSeconds())}`;
    const span = dom.lastUpdated.querySelector('span:last-child');
    if (span) span.textContent = `${t('lbl_last_updated')}: ${timeStr}`;
}
// ══════════════════════════════════════════════════════

function setSelectValue(selectEl, value) {
    if (!selectEl) return;
    if (!value) { selectEl.selectedIndex = 0; return; }
    for (let i = 0; i < selectEl.options.length; i++) {
        if (selectEl.options[i].value === value) { selectEl.value = value; return; }
    }
    // Value not found — add it before the last option (custom)
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = value;
    selectEl.insertBefore(opt, selectEl.lastElementChild);
    selectEl.value = value;
}

function setThinkingBudgetValue(value) {
    if (!dom.inputThinkingBudget) return;
    if (ALLOWED_THINKING_BUDGETS.includes(value)) {
        dom.inputThinkingBudget.value = value;
        return;
    }
    let opt = Array.from(dom.inputThinkingBudget.options).find(o => o.value === value);
    if (!opt) {
        opt = document.createElement('option');
        opt.value = value;
        opt.textContent = `${value} · ${t('opt_custom')}`;
        dom.inputThinkingBudget.insertBefore(opt, dom.inputThinkingBudget.lastElementChild);
    }
    dom.inputThinkingBudget.value = value;
}

async function loadStatus() {
    try {
        const resp = await apiFetch('/ocgt/api/status');
        if (!resp.ok) throw new Error('Failed');
        systemStatus = await resp.json();

        dom.elListen.textContent = systemStatus.listen;
        dom.elUpstream.textContent = systemStatus.upstream;
        dom.elProfile.textContent = systemStatus.active_profile;

        // Model
        if (systemStatus.default_model) {
            dom.elModel.textContent = systemStatus.default_model;
            dom.elModel.classList.remove('not-configured');
        } else {
            dom.elModel.textContent = t('status_not_configured');
            dom.elModel.classList.add('not-configured');
        }

        // Config path
        if (systemStatus.config_path) {
            dom.elConfigPath.textContent = systemStatus.config_path;
            dom.elConfigPath.classList.remove('not-configured');
        } else {
            dom.elConfigPath.textContent = t('status_not_configured');
            dom.elConfigPath.classList.add('not-configured');
        }

        // API Key
        if (dom.elApiKey) {
            const configured = systemStatus.api_key_configured !== false;
            dom.elApiKey.textContent = configured ? t('status_api_key_configured') : t('status_api_key_not_configured');
            dom.elApiKey.style.color = configured ? 'var(--green)' : 'var(--yellow)';
        }

        // Timeout
        if (dom.elTimeout) {
            const seconds = Number(systemStatus.request_timeout_seconds || 300);
            dom.elTimeout.textContent = `${seconds}s`;
            if (dom.inputTimeout && !document.activeElement.isSameNode(dom.inputTimeout)) {
                dom.inputTimeout.value = seconds.toString();
            }
        }

        // Thinking budget
        if (dom.inputThinkingBudget) {
            const budget = Number(systemStatus.max_thinking_budget_tokens ?? 512);
            if (!document.activeElement.isSameNode(dom.inputThinkingBudget)) {
                setThinkingBudgetValue(budget.toString());
            }
        }

        renderEnvCode();
        renderCCSwitchCode();
        updateLastUpdated();
        showDashboardContent();
        setProxyConnectionState('online');

        // Show unconfigured key warning in badge
        if (systemStatus.api_key_configured === false && dom.uptimeBadge) {
            const textSpan = dom.uptimeBadge.querySelector('span:last-child');
            if (textSpan) textSpan.textContent = t('status_connected_no_key');
        }
        return true;
    } catch (err) {
        console.error('Error fetching status:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
        showDashboardContent();
        return false;
    }
}

async function loadProfiles() {
    try {
        const resp = await apiFetch('/ocgt/api/profiles');
        if (!resp.ok) throw new Error('Failed');
        const data = await resp.json();
        dom.selectProfile.innerHTML = '';
        Object.keys(data.profiles).forEach(pName => {
            const opt = document.createElement('option');
            opt.value = pName;
            opt.textContent = pName;
            if (pName === data.active_profile) opt.selected = true;
            dom.selectProfile.appendChild(opt);
        });
        const activeProfile = data.profiles[data.active_profile];
        if (activeProfile) {
            dom.inputApiKey.value = activeProfile.api_key || '';
            setSelectValue(dom.inputDefaultModel, activeProfile.default_model || '');
            const aliases = activeProfile.model_aliases || {};
            setSelectValue(dom.inputSonnetMapping, aliases.sonnet || '');
            setSelectValue(dom.inputHaikuMapping, aliases.haiku || '');
            setSelectValue(dom.inputOpusMapping, aliases.opus || '');
        }
        captureOriginalSettings();
        clearChangesDetected();
        return true;
    } catch (err) {
        console.error('Error loading profiles:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
        return false;
    }
}

async function loadHistory() {
    try {
        const resp = await apiFetch('/ocgt/api/history');
        if (!resp.ok) throw new Error('Failed');
        renderHistoryTable(await resp.json());
        return true;
    } catch (err) {
        console.error('Error loading history:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
        return false;
    }
}

async function loadPreferences() {
    if (!dom.inputCloseBehavior) return;
    try {
        const prefs = await callWails('GetPreferences');
        dom.inputCloseBehavior.value = normalizeCloseBehavior(prefs && prefs.close_behavior);
        captureOriginalSettings();
    } catch (err) {
        console.error('Failed to load preferences:', err);
        dom.inputCloseBehavior.value = DEFAULT_CLOSE_BEHAVIOR;
    }
}
// ══════════════════════════════════════════════════════

function getSettingsSnapshot() {
    return {
        profile: dom.selectProfile ? dom.selectProfile.value : '',
        apiKey: dom.inputApiKey ? dom.inputApiKey.value : '',
        defaultModel: dom.inputDefaultModel ? dom.inputDefaultModel.value : '',
        sonnet: dom.inputSonnetMapping ? dom.inputSonnetMapping.value : '',
        haiku: dom.inputHaikuMapping ? dom.inputHaikuMapping.value : '',
        opus: dom.inputOpusMapping ? dom.inputOpusMapping.value : '',
        timeout: dom.inputTimeout ? dom.inputTimeout.value : '',
        thinkingBudget: dom.inputThinkingBudget ? dom.inputThinkingBudget.value : '',
        closeBehavior: dom.inputCloseBehavior ? dom.inputCloseBehavior.value : ''
    };
}

function captureOriginalSettings() {
    originalSettingsValues = getSettingsSnapshot();
}

function checkForChanges() {
    const current = getSettingsSnapshot();
    const hasChanges = Object.keys(originalSettingsValues).some(k => current[k] !== originalSettingsValues[k]);

    if (hasChanges && dom.configActions) {
        dom.configActions.classList.add('changes-detected');
        dom.btnSaveAllConfig.textContent = `\u26A1 ${t('btn_save_config')} \u00B7 ${t('hint_changes_detected')}`;
    } else if (dom.configActions) {
        dom.configActions.classList.remove('changes-detected');
        dom.btnSaveAllConfig.textContent = t('btn_save_config');
    }
}

function clearChangesDetected() {
    if (dom.configActions) {
        dom.configActions.classList.remove('changes-detected');
        dom.btnSaveAllConfig.textContent = t('btn_save_config');
    }
    captureOriginalSettings();
}
// ══════════════════════════════════════════════════════

function renderEnvCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const thinkingBudget = Number.isInteger(Number(systemStatus.max_thinking_budget_tokens))
        ? Number(systemStatus.max_thinking_budget_tokens) : 512;
    const thinkingLines = thinkingBudget < 0
        ? {
            powershell: `$env:MAX_THINKING_TOKENS = "0"\n$env:CLAUDE_CODE_DISABLE_THINKING = "1"`,
            bash: `export MAX_THINKING_TOKENS="0"\nexport CLAUDE_CODE_DISABLE_THINKING="1"`,
            cmd: `set MAX_THINKING_TOKENS=0\nset CLAUDE_CODE_DISABLE_THINKING=1`
        }
        : {
            powershell: `$env:MAX_THINKING_TOKENS = "${thinkingBudget}"`,
            bash: `export MAX_THINKING_TOKENS="${thinkingBudget}"`,
            cmd: `set MAX_THINKING_TOKENS=${thinkingBudget}`
        };
    const authTokenLines = LOCAL_AUTH_TOKEN
        ? {
            powershell: `$env:ANTHROPIC_AUTH_TOKEN = "${LOCAL_AUTH_TOKEN}"\n`,
            bash: `export ANTHROPIC_AUTH_TOKEN="${LOCAL_AUTH_TOKEN}"\n`,
            cmd: `set ANTHROPIC_AUTH_TOKEN=${LOCAL_AUTH_TOKEN}\n`
        }
        : { powershell: '', bash: '', cmd: '' };
    const templates = {
        powershell: `$env:ANTHROPIC_BASE_URL = "http://${listen}"\n$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"\n${authTokenLines.powershell}$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: ${profile}"\nRemove-Item Env:ANTHROPIC_MODEL -ErrorAction SilentlyContinue\n${thinkingLines.powershell}`,
        bash: `export ANTHROPIC_BASE_URL="http://${listen}"\nexport ANTHROPIC_API_KEY="ocgt-local-proxy"\n${authTokenLines.bash}export ANTHROPIC_CUSTOM_HEADERS="X-Ocgt-Profile: ${profile}"\nunset ANTHROPIC_MODEL\n${thinkingLines.bash}`,
        cmd: `set ANTHROPIC_BASE_URL=http://${listen}\nset ANTHROPIC_API_KEY=ocgt-local-proxy\n${authTokenLines.cmd}set ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile: ${profile}\nset ANTHROPIC_MODEL=\n${thinkingLines.cmd}`
    };
    dom.codeEnv.innerText = templates[currentShell] || templates.powershell;
}

function renderCCSwitchCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const headers = { "X-Ocgt-Profile": profile };
    if (LOCAL_AUTH_TOKEN) headers.Authorization = `Bearer ${LOCAL_AUTH_TOKEN}`;
    dom.codeCCSwitch.innerText = JSON.stringify({
        name: `ocgt-${profile}`, type: "anthropic",
        baseURL: `http://${listen}`, apiKey: "ocgt-local-proxy",
        model: "claude-sonnet-4-5",
        headers
    }, null, 2);
}

function renderHistoryTable(logs) {
    updateTrafficStats(logs);
    if (!logs || logs.length === 0) {
        dom.tbodyHistory.innerHTML = `<tr class="empty-row"><td colspan="7"><div class="empty-state"><div class="empty-state-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg></div><span>${t('traf_empty')}</span></div></td></tr>`;
        return;
    }
    dom.tbodyHistory.innerHTML = logs.map(log => {
        const time = formatTime(new Date(log.time));
        const badge = statusBadge(log.status);
        return `<tr>
            <td class="time-cell">${time}</td>
            <td class="method">${escapeHtml(log.method)}</td>
            <td class="path-cell" title="${escapeHtml(log.path)}">${escapeHtml(log.path)}</td>
            <td class="model-cell">${escapeHtml(log.model || '-')}</td>
            <td>${badge}</td>
            <td class="duration-cell">${escapeHtml(log.duration)}</td>
            <td class="error-cell" title="${escapeHtml(log.error || '')}">${escapeHtml(log.error || '-')}</td>
        </tr>`;
    }).join('');
}

function updateTrafficStats(logs) {
    const totalEl = document.getElementById('traffic-stat-total');
    const successEl = document.getElementById('traffic-stat-success');
    const latencyEl = document.getElementById('traffic-stat-latency');
    if (!totalEl || !successEl || !latencyEl) return;
    if (!logs || logs.length === 0) {
        totalEl.textContent = '0';
        successEl.textContent = '100.0%';
        latencyEl.textContent = '0ms';
        return;
    }
    const total = logs.length;
    let successCount = 0, totalLatencyMs = 0;
    logs.forEach(log => {
        if (log.status >= 200 && log.status < 300) successCount++;
        totalLatencyMs += parseDurationToMs(log.duration);
    });
    totalEl.textContent = total.toString();
    successEl.textContent = `${((successCount / total) * 100).toFixed(1)}%`;
    latencyEl.textContent = `${Math.round(totalLatencyMs / total)}ms`;
}

function parseDurationToMs(str) {
    if (!str) return 0;
    str = str.toLowerCase().trim();
    if (str.endsWith('ms')) return parseFloat(str.replace('ms', '')) || 0;
    if (str.endsWith('s')) return (parseFloat(str.replace('s', '')) * 1000) || 0;
    return parseFloat(str) || 0;
}

function formatTime(d) {
    return `${padTwo(d.getHours())}:${padTwo(d.getMinutes())}:${padTwo(d.getSeconds())}`;
}

function statusBadge(code) {
    if (code >= 200 && code < 300) return `<span class="badge-ok">${code}</span>`;
    if (code >= 400 && code < 500) return `<span class="badge-warn">${code}</span>`;
    return `<span class="badge-err">${code}</span>`;
}

function setButtonState(btn, state) {
    if (state === 'saving') {
        btn.disabled = true;
        btn.textContent = t('status_saving');
        btn.classList.add('btn-saving');
    } else if (state === 'success') {
        btn.disabled = true;
        btn.classList.add('btn-success');
        btn.classList.remove('btn-saving');
        btn.textContent = t('status_success');
    } else { // idle
        btn.disabled = false;
        btn.classList.remove('btn-success', 'btn-saving');
        btn.textContent = t('btn_save_config');
    }
}

function copyText(text, btn) {
    const tooltip = btn.querySelector('.copied-tooltip');
    const showTooltip = () => {
        if (tooltip) {
            tooltip.textContent = t('btn_copied');
            tooltip.classList.add('show');
            btn.style.borderColor = 'var(--green-border)';
            btn.style.color = 'var(--green)';
            setTimeout(() => {
                tooltip.classList.remove('show');
                btn.style.borderColor = '';
                btn.style.color = '';
            }, 1500);
        }
    };
    if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(showTooltip).catch(e => {
            console.error(e);
            toastI18n('toast_copy_failed', 'error');
        });
    } else {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        document.body.appendChild(ta);
        ta.select();
        try { document.execCommand('copy'); showTooltip(); } catch (e) { console.error(e); }
        document.body.removeChild(ta);
    }
}

function setFieldError(fieldId, message) {
    const field = document.getElementById(fieldId);
    if (!field) return;
    field.classList.add('error');
    const errorText = field.querySelector('.field-error-text');
    if (errorText) errorText.textContent = message;
}

function clearFieldErrors() {
    document.querySelectorAll('.field.error').forEach(f => f.classList.remove('error'));
}
// ══════════════════════════════════════════════════════

function updateLanguageDOM() {
    const lang = currentLang;
    const dict = i18n[lang];
    if (!dict) return;

    // Sync language selector
    if (dom.prefLangSelect) dom.prefLangSelect.value = lang;

    if (dom.prefsToggleBtn) {
        dom.prefsToggleBtn.setAttribute('title', lang === 'zh' ? '偏好设置' : 'Preferences');
    }

    // data-i18n elements
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        if (!dict[key]) return;
        const tag = el.tagName;
        if (['SPAN', 'BUTTON', 'H2', 'H3', 'H4', 'LABEL', 'P', 'TH', 'LI', 'OPTION'].includes(tag)) {
            const textNodes = Array.from(el.childNodes).filter(node => node.nodeType === Node.TEXT_NODE);
            const value = dict[key];
            // Use innerHTML when translation contains HTML tags (e.g. <code>claude</code>, <b>提示</b>)
            const containsHTML = /<[a-z][\s>]/i.test(value);
            if (containsHTML) {
                el.innerHTML = value;
            } else if (textNodes.length > 0) {
                textNodes[textNodes.length - 1].textContent = value;
            } else {
                el.textContent = value;
            }
        } else {
            el.textContent = dict[key];
        }
    });

    // Placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        if (dict[key]) el.setAttribute('placeholder', dict[key]);
    });

    // Footer
    if (dom.footerText) dom.footerText.textContent = dict.footer_text;

    updateActiveViewHeaders();
}

function updateActiveViewHeaders() {
    const activeItem = document.querySelector('.nav-item.active');
    if (!activeItem) return;
    const viewId = activeItem.dataset.view;
    const titleEl = document.getElementById('current-view-title');
    const subtitleEl = document.getElementById('current-view-subtitle');
    if (!titleEl || !subtitleEl) return;
    const meta = {
        dashboard: { title: t('title_dashboard'), subtitle: t('subtitle_dashboard') },
        settings:  { title: t('title_settings'),  subtitle: t('subtitle_settings') },
        terminal:  { title: t('title_terminal'),   subtitle: t('subtitle_terminal') },
        history:   { title: t('title_history'),    subtitle: t('subtitle_history') }
    }[viewId];
    if (meta) {
        titleEl.textContent = meta.title;
        subtitleEl.textContent = meta.subtitle;
    }
}
// ══════════════════════════════════════════════════════

// ── 12a: Navigation ──
function setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const views = document.querySelectorAll('.view');

    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const targetViewId = item.dataset.view;
            if (!targetViewId) return;
            navItems.forEach(nav => nav.classList.remove('active'));
            item.classList.add('active');
            views.forEach(v => v.classList.remove('active'));
            const targetView = document.getElementById(`view-${targetViewId}`);
            if (targetView) targetView.classList.add('active');
            updateActiveViewHeaders();
        });
    });

    // Status pill → dashboard
    if (dom.statusPill) {
        dom.statusPill.addEventListener('click', () => {
            const dashBtn = document.getElementById('btn-nav-dashboard');
            if (dashBtn) dashBtn.click();
        });
    }

    // Sidebar brand → dashboard
    const sidebarBrand = document.getElementById('sidebarBrand');
    if (sidebarBrand) {
        sidebarBrand.addEventListener('click', () => {
            const dashBtn = document.getElementById('btn-nav-dashboard');
            if (dashBtn) dashBtn.click();
        });
        sidebarBrand.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                const dashBtn = document.getElementById('btn-nav-dashboard');
                if (dashBtn) dashBtn.click();
            }
        });
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey || e.metaKey) {
            const viewMap = { '1': 'dashboard', '2': 'settings', '3': 'terminal', '4': 'history' };
            const viewId = viewMap[e.key];
            if (viewId) {
                e.preventDefault();
                const btn = document.querySelector(`[data-view="${viewId}"]`);
                if (btn) btn.click();
            }
        }
        if (e.key === 'Escape') {
            hideModal(dom.closeDialogOverlay);
            hideModal(dom.aboutDialogOverlay);
            closeSettingsPanel();
        }
    });
}

// ── 12b: Settings form ──
function setupSettingsHandlers() {
    // Profile change
    if (dom.selectProfile) {
        dom.selectProfile.addEventListener('change', async (e) => {
            try {
                const resp = await apiFetch('/ocgt/api/profiles/active', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ profile: e.target.value })
                });
                if (resp.ok) {
                    toastI18n('toast_profile_changed', 'success');
                    await loadStatus();
                    await loadProfiles();
                }
            } catch (err) { console.error('Failed to change profile:', err); }
        });
    }

    // Toggle password visibility
    if (dom.btnToggleVisibility) {
        dom.btnToggleVisibility.addEventListener('click', () => {
            dom.inputApiKey.type = dom.inputApiKey.type === 'password' ? 'text' : 'password';
        });
    }

    // Save config
    if (dom.btnSaveAllConfig) {
        dom.btnSaveAllConfig.addEventListener('click', handleSaveConfig);
    }

    // Change detection on all settings inputs
    const settingsInputs = [
        dom.selectProfile, dom.inputApiKey, dom.inputDefaultModel, dom.inputSonnetMapping,
        dom.inputHaikuMapping, dom.inputOpusMapping, dom.inputTimeout, dom.inputThinkingBudget,
        dom.inputCloseBehavior
    ];
    settingsInputs.forEach(el => {
        if (!el) return;
        el.addEventListener('input', checkForChanges);
        el.addEventListener('change', checkForChanges);
    });

    // Custom model select handling
    [dom.inputDefaultModel, dom.inputSonnetMapping, dom.inputHaikuMapping, dom.inputOpusMapping].forEach(selectEl => {
        if (!selectEl) return;
        selectEl.addEventListener('change', (e) => {
            if (e.target.value !== 'custom') return;
            const newVal = window.prompt(t('toast_custom_model_prompt'));
            if (newVal && newVal.trim()) {
                const value = newVal.trim();
                let exists = false;
                for (let i = 0; i < selectEl.options.length; i++) {
                    if (selectEl.options[i].value === value) { selectEl.selectedIndex = i; exists = true; break; }
                }
                if (!exists) {
                    const opt = document.createElement('option');
                    opt.value = value;
                    opt.textContent = value;
                    selectEl.insertBefore(opt, selectEl.lastElementChild);
                    selectEl.value = value;
                }
            } else {
                selectEl.selectedIndex = 0;
            }
            checkForChanges();
        });
    });

    // Reset defaults — fixed: "Confirm" action now correctly triggers the reset
    if (dom.resetDefaultsBtn) {
        dom.resetDefaultsBtn.addEventListener('click', () => {
            toast(t('toast_reset_confirm'), 'warning', {
                duration: 5000,
                actionLabel: t('toast_confirm'),
                actionCallback: () => {
                    if (dom.inputTimeout) dom.inputTimeout.value = '300';
                    if (dom.inputThinkingBudget) setThinkingBudgetValue('512');
                    if (dom.inputDefaultModel) setSelectValue(dom.inputDefaultModel, 'kimi-k2.6');
                    if (dom.inputSonnetMapping) setSelectValue(dom.inputSonnetMapping, 'qwen3.6-plus');
                    if (dom.inputHaikuMapping) setSelectValue(dom.inputHaikuMapping, 'deepseek-v4-flash');
                    if (dom.inputOpusMapping) setSelectValue(dom.inputOpusMapping, 'kimi-k2.6');
                    if (dom.inputCloseBehavior) dom.inputCloseBehavior.value = 'prompt';
                    captureOriginalSettings();
                    clearChangesDetected();
                    toastI18n('toast_reset_done', 'success');
                }
            });
        });
    }

    // Close behavior auto-save
    if (dom.inputCloseBehavior) {
        dom.inputCloseBehavior.addEventListener('change', async () => {
            checkForChanges();
            try { await callWails('SavePreferences', normalizeCloseBehavior(dom.inputCloseBehavior.value)); }
            catch (err) { console.error('Failed to save close behavior:', err); }
        });
    }
}

async function handleSaveConfig() {
    const pName = dom.selectProfile.value;
    const key = dom.inputApiKey.value.trim();
    const defModel = dom.inputDefaultModel.value.trim();
    const sonnet = dom.inputSonnetMapping.value.trim();
    const haiku = dom.inputHaikuMapping.value.trim();
    const opus = dom.inputOpusMapping.value.trim();
    const timeoutSeconds = dom.inputTimeout ? dom.inputTimeout.value.trim() : '300';
    const thinkingBudget = dom.inputThinkingBudget ? dom.inputThinkingBudget.value.trim() : '512';
    const timeoutNumber = Number(timeoutSeconds);

    // Validation
    let hasErrors = false;
    clearFieldErrors();
    if (!Number.isInteger(timeoutNumber) || timeoutNumber < 1 || timeoutNumber > 3600) {
        setFieldError('field-timeout', t('err_timeout_range'));
        hasErrors = true;
    }
    if (!ALLOWED_THINKING_BUDGETS.includes(thinkingBudget)) {
        hasErrors = true;
    }
    if (hasErrors) {
        toastI18n('toast_validation_error', 'error');
        return;
    }

    setButtonState(dom.btnSaveAllConfig, 'saving');
    const app = getWailsApp();

    if (app) {
        try {
            const res = await app.SaveProfileConfig(pName, key, defModel, sonnet, haiku, opus, timeoutSeconds, thinkingBudget);
            if (res === 'success') {
                if (dom.inputCloseBehavior && typeof app.SavePreferences === 'function') {
                    const prefRes = await app.SavePreferences(normalizeCloseBehavior(dom.inputCloseBehavior.value));
                    if (prefRes !== 'success') throw new Error(prefRes);
                }
                setButtonState(dom.btnSaveAllConfig, 'success');
                clearChangesDetected();
                toastI18n('toast_saved', 'success');
                await loadStatus();
                await loadPreferences();
                await loadProfiles();
                setTimeout(() => setButtonState(dom.btnSaveAllConfig, 'idle'), 1500);
            } else {
                setButtonState(dom.btnSaveAllConfig, 'idle');
                toast(t('toast_save_failed') + ': ' + res, 'error');
            }
        } catch (err) {
            console.error('Failed to save config via Wails:', err);
            setButtonState(dom.btnSaveAllConfig, 'idle');
            toast(t('toast_save_failed') + ': ' + err.message, 'error');
        }
    } else {
        // Fallback: HTTP API
        try {
            const resp = await apiFetch('/ocgt/api/key', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    profile: pName, api_key: key, default_model: defModel,
                    model_aliases: { sonnet, haiku, opus },
                    request_timeout_seconds: timeoutNumber,
                    max_thinking_budget_tokens: Number(thinkingBudget)
                })
            });
            if (resp.ok) {
                setButtonState(dom.btnSaveAllConfig, 'success');
                clearChangesDetected();
                toastI18n('toast_saved', 'success');
                await loadStatus();
                await loadProfiles();
                setTimeout(() => setButtonState(dom.btnSaveAllConfig, 'idle'), 1500);
            } else {
                setButtonState(dom.btnSaveAllConfig, 'idle');
                toastI18n('toast_save_failed', 'error');
            }
        } catch (err) {
            console.error('Fallback save error:', err);
            setButtonState(dom.btnSaveAllConfig, 'idle');
            toastI18n('toast_save_failed', 'error');
        }
    }
}

// ── 12c: Terminal ──
function setupTerminalHandlers() {
    // Shell tabs
    if (dom.shellTabs) {
        dom.shellTabs.addEventListener('click', (e) => {
            const btn = e.target.closest('.shell-tab');
            if (!btn) return;
            dom.shellTabs.querySelectorAll('.shell-tab').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentShell = btn.dataset.shell;
            renderEnvCode();
        });
    }

    // Launch terminal
    if (dom.btnLaunchTerminal) {
        dom.btnLaunchTerminal.addEventListener('click', handleLaunchTerminal);
    }

    // Copy buttons
    if (dom.btnCopyEnv) {
        dom.btnCopyEnv.addEventListener('click', () => copyText(dom.codeEnv.innerText, dom.btnCopyEnv));
    }
    if (dom.btnCopyCCSwitch) {
        dom.btnCopyCCSwitch.addEventListener('click', () => copyText(dom.codeCCSwitch.innerText, dom.btnCopyCCSwitch));
    }
}

async function handleLaunchTerminal() {
    const app = getWailsApp();
    if (!app) {
        toast(t('warn_desktop_only_launch'), 'warning');
        return;
    }
    dom.btnLaunchTerminal.disabled = true;
    const originalText = dom.btnLaunchTerminal.innerHTML;
    dom.btnLaunchTerminal.innerHTML = `<span class="status-dot pulse" style="background-color:white;width:6px;height:6px;margin-right:8px;"></span>${t('term_launching')}`;
    try {
        const res = await app.LaunchClaudeTerminal(currentShell, currentLang);
        if (res === 'success') {
            dom.btnLaunchTerminal.innerHTML = t('term_launched');
            dom.btnLaunchTerminal.style.background = 'var(--green)';
            toastI18n('toast_launch_success', 'success');
            setTimeout(() => {
                dom.btnLaunchTerminal.disabled = false;
                dom.btnLaunchTerminal.innerHTML = originalText;
                dom.btnLaunchTerminal.style.background = '';
            }, 2000);
        } else {
            dom.btnLaunchTerminal.disabled = false;
            dom.btnLaunchTerminal.innerHTML = originalText;
            toast(t('toast_launch_failed') + ': ' + res, 'error');
        }
    } catch (err) {
        dom.btnLaunchTerminal.disabled = false;
        dom.btnLaunchTerminal.innerHTML = originalText;
        dom.btnLaunchTerminal.style.background = '';
        console.error('Launch terminal error:', err);
        toast(t('toast_launch_failed') + ': ' + err.message, 'error');
    }
}

// ── 12d: Environment repair ──
function setupEnvRepairHandlers() {
    if (dom.btnInstallEnv) {
        dom.btnInstallEnv.addEventListener('click', () => installClaudeUserEnv(true));
    }
    if (dom.btnInstallEnvTerminal) {
        dom.btnInstallEnvTerminal.addEventListener('click', () => installClaudeUserEnv(true));
    }
}

async function installClaudeUserEnv(showAlert) {
    const app = getWailsApp();
    if (!app || typeof app.InstallClaudeUserEnv !== 'function') {
        if (showAlert) toast(t('warn_desktop_only_env'), 'info');
        return false;
    }
    const buttons = [dom.btnInstallEnv, dom.btnInstallEnvTerminal].filter(Boolean);
    buttons.forEach(btn => {
        btn.disabled = true;
        btn.dataset.originalText = btn.textContent;
        btn.textContent = t('env_repairing');
    });
    try {
        const res = await app.InstallClaudeUserEnv();
        if (res === 'success') {
            buttons.forEach(btn => { btn.textContent = t('env_repaired_hint'); });
            if (showAlert) toastI18n('toast_env_repaired', 'success');
            setTimeout(() => {
                buttons.forEach(btn => {
                    btn.disabled = false;
                    btn.textContent = btn.dataset.originalText || t('btn_repair_env');
                });
            }, 2200);
            return true;
        }
        if (showAlert) toast(t('toast_env_repair_failed') + ': ' + res, 'error');
        return false;
    } catch (err) {
        console.error('InstallClaudeUserEnv failed:', err);
        if (showAlert) toast(t('toast_env_repair_failed') + ': ' + err.message, 'error');
        return false;
    } finally {
        const repairedText = t('env_repaired_hint');
        if (!buttons.some(btn => btn.textContent === repairedText)) {
            buttons.forEach(btn => {
                btn.disabled = false;
                btn.textContent = btn.dataset.originalText || t('btn_repair_env');
            });
        }
    }
}

// ── 12e: History ──
function setupHistoryHandlers() {
    if (dom.clearHistoryBtn) {
        dom.clearHistoryBtn.addEventListener('click', async () => {
            try {
                const resp = await apiFetch('/ocgt/api/history', { method: 'DELETE' });
                if (resp.ok) {
                    renderHistoryTable([]);
                    toastI18n('toast_history_cleared', 'success');
                }
            } catch (err) {
                console.error('Clear history failed:', err);
                renderHistoryTable([]);
                toastI18n('toast_history_cleared', 'success');
            }
        });
    }
}

// ── 12f: Theme & preferences center panel ──
function applyTheme(theme) {
    if (theme === 'system') {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        document.documentElement.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
        document.documentElement.setAttribute('data-theme', theme);
    }
    localStorage.setItem('theme', theme);
    syncThemeButtons(theme);
}

function syncThemeButtons(theme) {
    document.querySelectorAll('.sp-theme-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.themeValue === theme);
    });
}

function openSettingsPanel() {
    const overlay = document.getElementById('settingsPanelOverlay');
    if (overlay) {
        overlay.classList.add('active');
        overlay.setAttribute('aria-hidden', 'false');
        // Sync current state
        const currentTheme = localStorage.getItem('theme') || 'light';
        syncThemeButtons(currentTheme);
    }
}

function closeSettingsPanel() {
    const overlay = document.getElementById('settingsPanelOverlay');
    if (overlay) {
        overlay.classList.remove('active');
        overlay.setAttribute('aria-hidden', 'true');
    }
}

function setupThemeLangHandlers() {
    // Settings panel open/close
    if (dom.prefsToggleBtn) {
        dom.prefsToggleBtn.addEventListener('click', () => openSettingsPanel());
    }

    const settingsPanelClose = document.getElementById('settingsPanelClose');
    if (settingsPanelClose) {
        settingsPanelClose.addEventListener('click', () => closeSettingsPanel());
    }

    const settingsPanelOverlay = document.getElementById('settingsPanelOverlay');
    if (settingsPanelOverlay) {
        settingsPanelOverlay.addEventListener('click', (e) => {
            if (e.target === settingsPanelOverlay) closeSettingsPanel();
        });
    }

    // Theme toggle group
    document.querySelectorAll('.sp-theme-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const theme = btn.dataset.themeValue;
            applyTheme(theme);
        });
    });

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (localStorage.getItem('theme') === 'system') {
            applyTheme('system');
        }
    });

    // Language select inside settings panel
    if (dom.prefLangSelect) {
        dom.prefLangSelect.value = currentLang;
        dom.prefLangSelect.addEventListener('change', (e) => {
            currentLang = e.target.value;
            localStorage.setItem('lang', currentLang);
            updateLanguageDOM();
            loadStatus();
        });
    }
}

// ── 12g: Dashboard actions ──
function setupDashboardHandlers() {
    const btnOpenConfig = document.getElementById('open-config-btn');
    if (btnOpenConfig) {
        btnOpenConfig.addEventListener('click', async () => {
            const app = getWailsApp();
            if (!app || typeof app.OpenConfigLocation !== 'function') {
                toast(t('warn_desktop_only_folder'), 'info');
                return;
            }
            try {
                const res = await app.OpenConfigLocation();
                if (res !== 'success') toast(t('err_open_folder') + ': ' + res, 'error');
            } catch (e) {
                console.error('OpenConfigLocation error:', e);
                toast(t('err_open_folder_generic') + ': ' + e.message, 'error');
            }
        });
    }
}

// ── 12h: Modals ──
function setupModalHandlers() {
    if (dom.btnAboutApp) {
        dom.btnAboutApp.addEventListener('click', () => {
            const app = getWailsApp();
            if (app && typeof app.ShowAboutDialog === 'function') {
                app.ShowAboutDialog();
            } else {
                showModal(dom.aboutDialogOverlay);
            }
        });
    }

    if (dom.closeDialogExit) dom.closeDialogExit.addEventListener('click', async () => {
        hideModal(dom.closeDialogOverlay);
        try { await callWails('QuitApp'); } catch (e) { console.error('QuitApp error:', e); }
    });
    if (dom.closeDialogMinimize) dom.closeDialogMinimize.addEventListener('click', async () => {
        hideModal(dom.closeDialogOverlay);
        try { await callWails('HideToTray'); } catch (e) { console.error('HideToTray error:', e); }
    });
    if (dom.closeDialogCancel) dom.closeDialogCancel.addEventListener('click', () => hideModal(dom.closeDialogOverlay));
    if (dom.aboutDialogClose) dom.aboutDialogClose.addEventListener('click', () => hideModal(dom.aboutDialogOverlay));

    // Click outside modal to close
    [dom.closeDialogOverlay, dom.aboutDialogOverlay].forEach(overlay => {
        if (!overlay) return;
        overlay.addEventListener('click', (e) => { if (e.target === overlay) hideModal(overlay); });
    });
}

// ── 12i: Wails runtime events ──
function setupWailsEvents() {
    if (!(window.runtime && typeof window.runtime.EventsOn === 'function')) return;
    window.runtime.EventsOn('nav-to-settings', () => {
        const settingsNavBtn = document.getElementById('btn-nav-settings');
        if (settingsNavBtn) settingsNavBtn.click();
    });
    window.runtime.EventsOn('show-close-dialog', () => showModal(dom.closeDialogOverlay));
    window.runtime.EventsOn('show-about-dialog', () => showModal(dom.aboutDialogOverlay));
}

/** Master event handler setup — delegates to focused sub-functions */
function setupEventHandlers() {
    setupNavigation();
    setupSettingsHandlers();
    setupTerminalHandlers();
    setupEnvRepairHandlers();
    setupHistoryHandlers();
    setupThemeLangHandlers();
    setupDashboardHandlers();
    setupModalHandlers();
    setupWailsEvents();
}
// ══════════════════════════════════════════════════════

(function initMinimizeDetection() {
    let minimizeDebounce = null;
    function tryHideToTray() {
        if (minimizeDebounce) clearTimeout(minimizeDebounce);
        minimizeDebounce = setTimeout(() => {
            if (!dom.inputCloseBehavior || dom.inputCloseBehavior.value !== 'minimize') return;
            if (document.hidden || (window.outerWidth < 100 && window.outerHeight < 100)) {
                callWails('HideToTray').catch(e => console.error('[ocgt] HideToTray error:', e));
            }
        }, 200);
    }
    document.addEventListener('visibilitychange', () => { if (document.hidden) tryHideToTray(); });
    window.addEventListener('resize', () => {
        if (window.outerWidth < 100 || window.outerHeight < 100) tryHideToTray();
    });
})();
// ══════════════════════════════════════════════════════

document.addEventListener('DOMContentLoaded', () => {
    cacheDom();

    // Stamp version from single source of truth
    if (dom.appVersion) dom.appVersion.textContent = APP_VERSION;
    if (dom.aboutVersion) dom.aboutVersion.textContent = APP_VERSION;
    if (dom.footerText) dom.footerText.textContent = t('footer_text');

    setupEventHandlers();
    updateLanguageDOM();
    loadPreferences();
    initializeApp();

    // Polling: refresh history when online, otherwise try to reconnect
    setInterval(async () => {
        if (proxyReady) { await loadHistory(); }
        else { await initializeApp(); }
    }, 2500);
});
