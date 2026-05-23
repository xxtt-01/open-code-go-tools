// 全局状态变量
let API_BASE = 'http://127.0.0.1:8787';
let systemStatus = null;
let currentShell = 'powershell';
let proxyReady = false;
let currentLang = localStorage.getItem('lang') || 'zh'; // default 'zh'

// Bilingual Dictionary
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
        status_auth_enabled: "已启用",
        status_auth_disabled: "未启用",
        status_model_unset: "未设定",
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
        lbl_auth: "本地认证",
        lbl_profile: "当前活跃 Profile",
        lbl_model: "默认解析模型",
        lbl_config_path: "本地配置文件路径",
        btn_open_folder: "打开所在文件夹",
        sett_title: "一键配置管理中心",
        sett_profile: "当前配置 Profile",
        sett_default_model: "全局默认模型 (Default Model)",
        sett_api_key: "OpenCode Go API Key (代理密钥)",
        placeholder_api_key: "请输入您的 sk-... 密钥",
        sett_timeout: "请求超时检查 (秒，1-3600)",
        sett_thinking: "思考强度",
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
        hint_save: "保存后同步更新 config.json、热重载 Go 服务，并清理旧登录 Token/CC Switch 本地路由残留",
        hint_tip: "💡 提示：在“终端启动”页中只需选择并一键启动任意一种您习惯的命令行窗口即可，无需重复配置多种终端。",
        term_title: "一键唤醒代理控制台",
        term_shell_type: "目标命令行类型",
        btn_launch_term: "一键拉起配置终端 (Launch)",
        btn_persistent_env: "修复以后所有新终端环境变量",
        hint_launch: "一键注入当前 Profile 代理变量并打开原生 shell。直接打 <code>claude</code> 即可开始运行！",
        guide_title: "💡 快捷运行极简指南",
        guide_1: "在上方选项卡选择您常用的命令终端。",
        guide_2: "点击 <b>“一键拉起配置终端”</b>，系统会自动唤醒控制台。",
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
        traf_listening: "实时流量雷达持续监听中"
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
        status_auth_enabled: "Enabled",
        status_auth_disabled: "Disabled",
        status_model_unset: "Unset",
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
        lbl_auth: "Local Auth",
        lbl_profile: "Active Profile",
        lbl_model: "Default Model",
        lbl_config_path: "Local Config Path",
        btn_open_folder: "Open Directory",
        sett_title: "Easy Configuration Center",
        sett_profile: "Current Profile",
        sett_default_model: "Global Default Model",
        sett_api_key: "OpenCode Go API Key",
        placeholder_api_key: "Enter your OpenCode sk-... API Key",
        sett_timeout: "Request Timeout (Seconds, 1-3600)",
        sett_thinking: "Reasoning Intensity",
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
        hint_save: "Saves configuration, updates config.json, hot-reloads Go proxy, and clears old cache/CC Switch conflicts",
        hint_tip: "💡 Tip: Just select and launch any terminal shell of your choice. No need to repeatedly configure all shells.",
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
        traf_listening: "Live Traffic Radar Active & Listening"
    }
};

// DOM
const elListen = document.getElementById('status-listen');
const elUpstream = document.getElementById('status-upstream');
const elProfile = document.getElementById('status-profile');
const elModel = document.getElementById('status-model');
const elConfigPath = document.getElementById('status-config-path');
const elTimeout = document.getElementById('status-timeout');
const elApiKey = document.getElementById('status-api-key');
const selectProfile = document.getElementById('profile-select');
const inputApiKey = document.getElementById('api-key-input');
const inputTimeout = document.getElementById('timeout-input');
const inputThinkingBudget = document.getElementById('thinking-budget-input');
const inputDefaultModel = document.getElementById('default-model-input');
const inputSonnetMapping = document.getElementById('mapping-sonnet-input');
const inputHaikuMapping = document.getElementById('mapping-haiku-input');
const inputOpusMapping = document.getElementById('mapping-opus-input');
const btnSaveAllConfig = document.getElementById('save-all-config-btn');
const btnInstallEnv = document.getElementById('install-env-btn');
const btnInstallEnvTerminal = document.getElementById('install-env-terminal-btn');
const btnToggleVisibility = document.getElementById('toggle-key-visibility');
const btnLaunchTerminal = document.getElementById('launch-terminal-btn');
const shellTabs = document.getElementById('shell-tabs');
const codeEnv = document.getElementById('env-code-block');
const codeCCSwitch = document.getElementById('ccswitch-code-block');
const btnCopyEnv = document.getElementById('copy-env-btn');
const btnCopyCCSwitch = document.getElementById('copy-ccswitch-btn');
const tbodyHistory = document.getElementById('history-tbody');
const themeToggleBtn = document.getElementById('themeToggleBtn');
const langToggleBtn = document.getElementById('langToggleBtn');
const statusPill = document.getElementById('statusPill');
const uptimeBadge = document.querySelector('.uptime-badge');

let isInitializing = false;

document.addEventListener('DOMContentLoaded', () => {
    setupEventHandlers();
    updateLanguageDOM();
    initializeApp();
    setInterval(async () => {
        if (proxyReady) {
            await loadHistory();
        } else {
            await initializeApp();
        }
    }, 2500);
});

async function initializeApp() {
    if (isInitializing) return;
    isInitializing = true;
    setProxyConnectionState('connecting');
    await resolveApiBase();
    proxyReady = await waitForProxyReady();
    if (!proxyReady) {
        setProxyConnectionState('offline');
        isInitializing = false;
        return;
    }
    setProxyConnectionState('online');
    try {
        await Promise.all([loadStatus(), loadProfiles(), loadHistory()]);
    } catch (err) {
        console.error('Error during initial load:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
    } finally {
        isInitializing = false;
    }
}

async function resolveApiBase() {
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.GetListenAddress)) {
        return;
    }
    try {
        const addr = await window.go.main.App.GetListenAddress();
        if (addr) {
            API_BASE = `http://${addr}`;
        }
    } catch (err) {
        console.error("Wails GetListenAddress error:", err);
    }
}

async function waitForProxyReady(timeoutMs = 2500) {
    const started = Date.now();
    while (Date.now() - started < timeoutMs) {
        try {
            const resp = await apiFetch('/healthz', { cache: 'no-store' }, 700);
            if (resp.ok) return true;
        } catch (err) {
            // Retry until the Wails-started proxy finishes binding its port.
        }
        await delay(350);
    }
    return false;
}

function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function apiFetch(path, options = {}, timeoutMs = 8000) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    try {
        return await fetch(`${API_BASE}${path}`, {
            ...options,
            signal: controller.signal
        });
    } finally {
        clearTimeout(timeout);
    }
}

function setProxyConnectionState(state) {
    const dict = i18n[currentLang];
    const meta = {
        connecting: { text: dict.status_connecting, className: 'connecting' },
        online: { text: dict.status_online, className: 'online' },
        offline: { text: dict.status_offline, className: 'offline' }
    }[state];
    if (!meta) return;

    if (statusPill) {
        statusPill.classList.remove('online', 'offline', 'connecting');
        statusPill.classList.add(meta.className);
        const text = statusPill.querySelector('span:last-child');
        if (text) text.textContent = meta.text;
    }
    if (uptimeBadge) {
        uptimeBadge.classList.remove('online', 'offline', 'connecting');
        uptimeBadge.classList.add(meta.className);
        const text = uptimeBadge.querySelector('span:last-child');
        if (text) text.textContent = meta.text;
    }
}

async function loadStatus() {
    try {
        const resp = await apiFetch('/ocgt/api/status');
        if (!resp.ok) throw new Error('Failed');
        systemStatus = await resp.json();
        elListen.textContent = systemStatus.listen;
        elUpstream.textContent = systemStatus.upstream;
        elProfile.textContent = systemStatus.active_profile;
        
        const dict = i18n[currentLang];
        elModel.textContent = systemStatus.default_model || dict.status_model_unset;
        elConfigPath.textContent = systemStatus.config_path;
        
        if (elApiKey) {
            elApiKey.textContent = systemStatus.api_key_configured === false ? dict.status_api_key_not_configured : dict.status_api_key_configured;
            elApiKey.style.color = systemStatus.api_key_configured === false ? 'var(--yellow)' : 'var(--green)';
        }
        if (elTimeout) {
            const seconds = Number(systemStatus.request_timeout_seconds || 300);
            elTimeout.textContent = `${seconds}s`;
            if (inputTimeout && !document.activeElement.isSameNode(inputTimeout)) {
                inputTimeout.value = seconds.toString();
            }
        }
        if (inputThinkingBudget) {
            const budget = Number(systemStatus.max_thinking_budget_tokens ?? 512);
            if (!document.activeElement.isSameNode(inputThinkingBudget)) {
                setThinkingBudgetValue(budget.toString());
            }
        }
        // Show auth status
        const elAuth = document.getElementById('status-auth');
        if (elAuth) {
            elAuth.textContent = systemStatus.auth_enabled ? dict.status_auth_enabled : dict.status_auth_disabled;
            elAuth.style.color = systemStatus.auth_enabled ? 'var(--green)' : 'var(--gray)';
        }
        renderEnvCode();
        renderCCSwitchCode();
        setProxyConnectionState('online');
        if (systemStatus.api_key_configured === false && uptimeBadge) {
            const text = uptimeBadge.querySelector('span:last-child');
            if (text) text.textContent = currentLang === 'zh' ? '代理已连接，密钥未配置' : 'Connected, API Key Unconfigured';
        }
        return true;
    } catch (err) {
        console.error('Error fetching status:', err);
        proxyReady = false;
        setProxyConnectionState('offline');
        return false;
    }
}

function setSelectValue(selectEl, value) {
    if (!selectEl) return;
    if (!value) {
        selectEl.selectedIndex = 0;
        return;
    }
    let exists = false;
    for (let i = 0; i < selectEl.options.length; i++) {
        if (selectEl.options[i].value === value) {
            selectEl.value = value;
            exists = true;
            break;
        }
    }
    if (!exists) {
        const opt = document.createElement('option');
        opt.value = value;
        opt.textContent = value;
        selectEl.insertBefore(opt, selectEl.lastElementChild);
        selectEl.value = value;
    }
}

function setThinkingBudgetValue(value) {
    if (!inputThinkingBudget) return;
    const allowed = ['256', '512', '1024', '2048', '-1'];
    if (allowed.includes(value)) {
        inputThinkingBudget.value = value;
        return;
    }
    let opt = Array.from(inputThinkingBudget.options).find(item => item.value === value);
    if (!opt) {
        opt = document.createElement('option');
        opt.value = value;
        opt.textContent = `${value} · ${currentLang === 'zh' ? '当前自定义值' : 'Custom value'}`;
        inputThinkingBudget.insertBefore(opt, inputThinkingBudget.lastElementChild);
    }
    inputThinkingBudget.value = value;
}

function isAllowedThinkingBudget(value) {
    return ['256', '512', '1024', '2048', '-1'].includes(value);
}

async function loadProfiles() {
    try {
        const resp = await apiFetch('/ocgt/api/profiles');
        if (!resp.ok) throw new Error('Failed');
        const data = await resp.json();
        selectProfile.innerHTML = '';
        Object.keys(data.profiles).forEach(pName => {
            const opt = document.createElement('option');
            opt.value = pName;
            opt.textContent = pName;
            if (pName === data.active_profile) opt.selected = true;
            selectProfile.appendChild(opt);
        });

        // Populate fields for active profile
        const activeProfileName = data.active_profile;
        const activeProfile = data.profiles[activeProfileName];
        if (activeProfile) {
            inputApiKey.value = activeProfile.api_key || '';
            setSelectValue(inputDefaultModel, activeProfile.default_model || '');
            
            const aliases = activeProfile.model_aliases || {};
            setSelectValue(inputSonnetMapping, aliases.sonnet || '');
            setSelectValue(inputHaikuMapping, aliases.haiku || '');
            setSelectValue(inputOpusMapping, aliases.opus || '');
        }
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

function setupEventHandlers() {
    // 侧边栏多工作区视图切换
    const navItems = document.querySelectorAll('.nav-item');
    const views = document.querySelectorAll('.view');

    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const targetViewId = item.dataset.view;
            if (!targetViewId) return;

            // 1. 更新导航激活状态
            navItems.forEach(nav => nav.classList.remove('active'));
            item.classList.add('active');

            // 2. 切换视图显示
            views.forEach(v => v.classList.remove('active'));
            const targetView = document.getElementById(`view-${targetViewId}`);
            if (targetView) targetView.classList.add('active');

            // 3. 动态更新顶栏标题与副标题
            updateActiveViewHeaders();
        });
    });

    selectProfile.addEventListener('change', async (e) => {
        try {
            const resp = await apiFetch('/ocgt/api/profiles/active', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ profile: e.target.value })
            });
            if (resp.ok) {
                await loadStatus();
                await loadProfiles();
            }
        } catch (err) {
            console.error('Failed to change profile:', err);
        }
    });

    btnToggleVisibility.addEventListener('click', () => {
        inputApiKey.type = inputApiKey.type === 'password' ? 'text' : 'password';
    });

    btnSaveAllConfig.addEventListener('click', async () => {
        const pName = selectProfile.value;
        const key = inputApiKey.value.trim();
        const defModel = inputDefaultModel.value.trim();
        const sonnet = inputSonnetMapping.value.trim();
        const haiku = inputHaikuMapping.value.trim();
        const opus = inputOpusMapping.value.trim();
        const timeoutSeconds = inputTimeout ? inputTimeout.value.trim() : '300';
        const thinkingBudget = inputThinkingBudget ? inputThinkingBudget.value.trim() : '512';
        const timeoutNumber = Number(timeoutSeconds);

        if (!Number.isInteger(timeoutNumber) || timeoutNumber < 1 || timeoutNumber > 3600) {
            alert(currentLang === 'zh' ? '请求超时必须是 1 到 3600 之间的整数秒。' : 'Request timeout must be an integer between 1 and 3600 seconds.');
            return;
        }
        if (!isAllowedThinkingBudget(thinkingBudget)) {
            alert(currentLang === 'zh' ? '请选择有效的思考强度。' : 'Please select a valid reasoning strength.');
            return;
        }

        setButtonState(btnSaveAllConfig, 'saving');

        if (window.go && window.go.main && window.go.main.App) {
            try {
                const res = await window.go.main.App.SaveProfileConfig(pName, key, defModel, sonnet, haiku, opus, timeoutSeconds, thinkingBudget);
                if (res === "success") {
                    setButtonState(btnSaveAllConfig, 'success');
                    await installClaudeUserEnv(false);
                    await loadStatus();
                    await loadProfiles();
                    setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500);
                } else {
                    setButtonState(btnSaveAllConfig, 'idle');
                    alert((currentLang === 'zh' ? '保存失败: ' : 'Save failed: ') + res);
                }
            } catch (err) {
                console.error('Failed to save config via Wails:', err);
                setButtonState(btnSaveAllConfig, 'idle');
                alert((currentLang === 'zh' ? '保存出错: ' : 'Save error: ') + err.message);
            }
        } else {
            // Fallback for API call if running in browser directly
            try {
                const resp = await apiFetch('/ocgt/api/key', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        profile: pName,
                        api_key: key,
                        default_model: defModel,
                        model_aliases: { sonnet, haiku, opus },
                        request_timeout_seconds: timeoutNumber,
                        max_thinking_budget_tokens: Number(thinkingBudget)
                    })
                });
                if (resp.ok) {
                    setButtonState(btnSaveAllConfig, 'success');
                    await loadStatus();
                    await loadProfiles();
                    setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500);
                } else {
                    setButtonState(btnSaveAllConfig, 'idle');
                    alert(currentLang === 'zh' ? '保存失败，请检查控制台。' : 'Save failed, please check console.');
                }
            } catch (err) {
                console.error('Fallback save error:', err);
                setButtonState(btnSaveAllConfig, 'idle');
            }
        }
    });

    if (btnInstallEnv) {
        btnInstallEnv.addEventListener('click', () => installClaudeUserEnv(true));
    }

    if (btnInstallEnvTerminal) {
        btnInstallEnvTerminal.addEventListener('click', () => installClaudeUserEnv(true));
    }

    if (btnLaunchTerminal) {
        btnLaunchTerminal.addEventListener('click', async () => {
            if (window.go && window.go.main && window.go.main.App) {
                btnLaunchTerminal.disabled = true;
                const originalText = btnLaunchTerminal.innerHTML;
                btnLaunchTerminal.innerHTML = `<span class="status-dot pulse" style="background-color:white;width:6px;height:6px;margin-right:8px;"></span>${currentLang === 'zh' ? '启动中...' : 'Launching...'}`;
                try {
                    const res = await window.go.main.App.LaunchClaudeTerminal(currentShell);
                    if (res === "success") {
                        btnLaunchTerminal.innerHTML = currentLang === 'zh' ? '已启动终端 ✓' : 'Terminal Launched ✓';
                        btnLaunchTerminal.style.background = 'var(--green)';
                        setTimeout(() => {
                            btnLaunchTerminal.disabled = false;
                            btnLaunchTerminal.innerHTML = originalText;
                            btnLaunchTerminal.style.background = '';
                        }, 2000);
                    } else {
                        btnLaunchTerminal.disabled = false;
                        btnLaunchTerminal.innerHTML = originalText;
                        btnLaunchTerminal.style.background = '';
                        alert((currentLang === 'zh' ? '启动失败: ' : 'Launch failed: ') + res);
                    }
                } catch (err) {
                    btnLaunchTerminal.disabled = false;
                    btnLaunchTerminal.innerHTML = originalText;
                    btnLaunchTerminal.style.background = '';
                    console.error("Launch terminal error:", err);
                    alert((currentLang === 'zh' ? '启动终端发生错误: ' : 'Launch error: ') + err.message);
                }
            } else {
                alert(currentLang === 'zh' ? '一键启动终端仅在桌面版 app 客户端可用，请在桌面端中点击使用！' : 'One-click launch is only available in the desktop app!');
            }
        });
    }

    shellTabs.addEventListener('click', (e) => {
        const btn = e.target.closest('.shell-tab');
        if (!btn) return;
        shellTabs.querySelectorAll('.shell-tab').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        currentShell = btn.dataset.shell;
        renderEnvCode();
    });

    btnCopyEnv.addEventListener('click', () => copyText(codeEnv.innerText, btnCopyEnv));
    btnCopyCCSwitch.addEventListener('click', () => copyText(codeCCSwitch.innerText, btnCopyCCSwitch));

    if (themeToggleBtn) {
        themeToggleBtn.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
            const newTheme = currentTheme === 'light' ? 'dark' : 'light';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        });
    }

    if (langToggleBtn) {
        langToggleBtn.addEventListener('click', () => {
            currentLang = currentLang === 'zh' ? 'en' : 'zh';
            localStorage.setItem('lang', currentLang);
            updateLanguageDOM();
            loadStatus(); // refresh values with i18n
        });
    }

    // 绑定四个模型下拉框的“自定义”选择逻辑
    const handleModelSelectChange = (selectEl) => {
        if (!selectEl) return;
        selectEl.addEventListener('change', (e) => {
            if (e.target.value === 'custom') {
                const promptMsg = currentLang === 'zh' 
                    ? "请输入您想映射或设定的自定义模型名称 (例如: my-custom-model):" 
                    : "Please enter custom model name (e.g., my-custom-model):";
                const newVal = prompt(promptMsg);
                if (newVal && newVal.trim()) {
                    const value = newVal.trim();
                    let exists = false;
                    for (let i = 0; i < selectEl.options.length; i++) {
                        if (selectEl.options[i].value === value) {
                            selectEl.selectedIndex = i;
                            exists = true;
                            break;
                        }
                    }
                    if (!exists) {
                        const opt = document.createElement('option');
                        opt.value = value;
                        opt.textContent = value;
                        selectEl.insertBefore(opt, selectEl.lastElementChild);
                        selectEl.value = value;
                    }
                } else {
                    selectEl.selectedIndex = 0; // 回滚到默认
                }
            }
        });
    };

    handleModelSelectChange(inputDefaultModel);
    handleModelSelectChange(inputSonnetMapping);
    handleModelSelectChange(inputHaikuMapping);
    handleModelSelectChange(inputOpusMapping);

    // 绑定“打开所在文件夹”按钮
    const btnOpenConfig = document.getElementById('open-config-btn');
    if (btnOpenConfig) {
        btnOpenConfig.addEventListener('click', async () => {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.OpenConfigLocation) {
                try {
                    const res = await window.go.main.App.OpenConfigLocation();
                    if (res !== "success") {
                        alert("打开失败: " + res);
                    }
                } catch (e) {
                    console.error("OpenConfigLocation error:", e);
                    alert("无法打开文件夹: " + e.message);
                }
            } else {
                alert(currentLang === 'zh' 
                    ? "该功能仅在桌面客户端可用。您的配置文件夹通常在您的个人用户目录下的 .ocgt 文件夹中。" 
                    : "Only available in the desktop client. Config is typically under ~/.ocgt directory.");
            }
        });
    }
}

function updateLanguageDOM() {
    const lang = currentLang;
    const dict = i18n[lang];
    if (!dict) return;

    // Toggle button text
    if (langToggleBtn) {
        langToggleBtn.querySelector('span').textContent = lang === 'zh' ? 'EN' : '中';
    }

    // Translate elements with data-i18n attribute
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        if (dict[key]) {
            if (['SPAN', 'BUTTON', 'H2', 'H3', 'H4', 'LABEL', 'P', 'TH', 'LI', 'OPTION'].includes(el.tagName)) {
                // Safely replace only the text node inside elements containing SVGs or other HTML tags
                const textNodes = Array.from(el.childNodes).filter(node => node.nodeType === Node.TEXT_NODE);
                if (textNodes.length > 0) {
                    textNodes[textNodes.length - 1].textContent = dict[key];
                } else {
                    el.textContent = dict[key];
                }
            } else {
                el.textContent = dict[key];
            }
        }
    });

    // Translate placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        if (dict[key]) {
            el.setAttribute('placeholder', dict[key]);
        }
    });

    updateActiveViewHeaders();
}

function updateActiveViewHeaders() {
    const activeItem = document.querySelector('.nav-item.active');
    if (!activeItem) return;
    const viewId = activeItem.dataset.view;
    const titleEl = document.getElementById('current-view-title');
    const subtitleEl = document.getElementById('current-view-subtitle');
    if (!titleEl || !subtitleEl) return;

    const dict = i18n[currentLang];
    if (viewId === 'dashboard') {
        titleEl.textContent = dict.title_dashboard;
        subtitleEl.textContent = dict.subtitle_dashboard;
    } else if (viewId === 'settings') {
        titleEl.textContent = dict.title_settings;
        subtitleEl.textContent = dict.subtitle_settings;
    } else if (viewId === 'terminal') {
        titleEl.textContent = dict.title_terminal;
        subtitleEl.textContent = dict.subtitle_terminal;
    } else if (viewId === 'history') {
        titleEl.textContent = dict.title_history;
        subtitleEl.textContent = dict.subtitle_history;
    }
}

async function installClaudeUserEnv(showAlert) {
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.InstallClaudeUserEnv)) {
        if (showAlert) {
            alert(currentLang === 'zh' 
                ? '该功能仅桌面端可用。当前浏览器模式请复制右侧环境变量手动执行。' 
                : 'Only available in desktop app. Please copy the env variables manually on the right.');
        }
        return false;
    }

    const buttons = [btnInstallEnv, btnInstallEnvTerminal].filter(Boolean);
    buttons.forEach(btn => {
        btn.disabled = true;
        btn.dataset.originalText = btn.textContent;
        btn.textContent = currentLang === 'zh' ? '修复中...' : 'Repairing...';
    });

    try {
        const res = await window.go.main.App.InstallClaudeUserEnv();
        if (res === 'success') {
            buttons.forEach(btn => {
                btn.textContent = currentLang === 'zh' ? '已修复，重新打开终端生效' : 'Repaired! Reopen terminals to apply';
            });
            if (showAlert) {
                alert(currentLang === 'zh' 
                    ? '已写入 Windows 用户环境变量。请关闭旧 PowerShell，重新打开终端后再运行 claude。' 
                    : 'User environment variables successfully updated. Restart your shell and run `claude`.');
            }
            setTimeout(() => {
                buttons.forEach(btn => {
                    btn.disabled = false;
                    btn.textContent = btn.dataset.originalText || (currentLang === 'zh' ? '一键修复 Claude Code 系统环境变量' : 'One-click Repair Claude Code System Env');
                });
            }, 2200);
            return true;
        }
        if (showAlert) alert((currentLang === 'zh' ? '修复失败: ' : 'Repair failed: ') + res);
        return false;
    } catch (err) {
        console.error('InstallClaudeUserEnv failed:', err);
        if (showAlert) alert((currentLang === 'zh' ? '修复失败: ' : 'Repair failed: ') + err.message);
        return false;
    } finally {
        const resetText = currentLang === 'zh' ? '已修复，重新打开终端生效' : 'Repaired! Reopen terminals to apply';
        if (!buttons.some(btn => btn.textContent === resetText)) {
            buttons.forEach(btn => {
                btn.disabled = false;
                btn.textContent = btn.dataset.originalText || (currentLang === 'zh' ? '一键修复 Claude Code 系统环境变量' : 'One-click Repair Claude Code System Env');
            });
        }
    }
}

function renderEnvCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const model = systemStatus.default_model || 'kimi-k2.6';
    const thinkingBudget = Number.isInteger(Number(systemStatus.max_thinking_budget_tokens))
        ? Number(systemStatus.max_thinking_budget_tokens)
        : 512;
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
    const templates = {
        powershell: `$env:ANTHROPIC_BASE_URL = "http://${listen}"\n$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"\n$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: ${profile}"\n$env:ANTHROPIC_MODEL = "${model}"\n${thinkingLines.powershell}`,
        bash: `export ANTHROPIC_BASE_URL="http://${listen}"\nexport ANTHROPIC_API_KEY="ocgt-local-proxy"\nexport ANTHROPIC_CUSTOM_HEADERS="X-Ocgt-Profile: ${profile}"\nexport ANTHROPIC_MODEL="${model}"\n${thinkingLines.bash}`,
        cmd: `set ANTHROPIC_BASE_URL=http://${listen}\nset ANTHROPIC_API_KEY=ocgt-local-proxy\nset ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile:${profile}\nset ANTHROPIC_MODEL=${model}\n${thinkingLines.cmd}`
    };
    codeEnv.innerText = templates[currentShell] || templates.powershell;
}

function renderCCSwitchCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const model = systemStatus.default_model || 'kimi-k2.6';
    codeCCSwitch.innerText = JSON.stringify({
        name: `ocgt-${profile}`,
        type: "anthropic",
        baseURL: `http://${listen}`,
        apiKey: "ocgt-local-proxy",
        model,
        headers: { "X-Ocgt-Profile": profile }
    }, null, 2);
}

function renderHistoryTable(logs) {
    updateTrafficStats(logs);

    const dict = i18n[currentLang];
    if (!logs || logs.length === 0) {
        tbodyHistory.innerHTML = `<tr class="empty-row"><td colspan="7"><div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>${dict.traf_empty}</div></td></tr>`;
        return;
    }
    tbodyHistory.innerHTML = logs.map(log => {
        const time = formatTime(new Date(log.time));
        const badge = statusBadge(log.status);
        return `<tr>
            <td class="time-cell">${time}</td>
            <td class="method">${log.method}</td>
            <td class="path-cell" title="${log.path}">${log.path}</td>
            <td class="model-cell">${log.model || '-'}</td>
            <td>${badge}</td>
            <td class="duration-cell">${log.duration}</td>
            <td class="error-cell" title="${escapeHtml(log.error || '')}">${escapeHtml(log.error || '-')}</td>
        </tr>`;
    }).join('');
}

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function updateTrafficStats(logs) {
    const totalEl = document.getElementById('traffic-stat-total');
    const successEl = document.getElementById('traffic-stat-success');
    const latencyEl = document.getElementById('traffic-stat-latency');

    if (!totalEl || !successEl || !latencyEl) return;

    if (!logs || logs.length === 0) {
        totalEl.textContent = "0";
        successEl.textContent = "100.0%";
        latencyEl.textContent = "0ms";
        return;
    }

    const total = logs.length;
    let successCount = 0;
    let totalLatencyMs = 0;

    logs.forEach(log => {
        if (log.status >= 200 && log.status < 300) {
            successCount++;
        }
        totalLatencyMs += parseDurationToMs(log.duration);
    });

    const successRate = ((successCount / total) * 100).toFixed(1);
    const avgLatency = Math.round(totalLatencyMs / total);

    totalEl.textContent = total.toString();
    successEl.textContent = `${successRate}%`;
    latencyEl.textContent = `${avgLatency}ms`;
}

function parseDurationToMs(str) {
    if (!str) return 0;
    str = str.toLowerCase().trim();
    if (str.endsWith('ms')) {
        return parseFloat(str.replace('ms', '')) || 0;
    }
    if (str.endsWith('s')) {
        return (parseFloat(str.replace('s', '')) * 1000) || 0;
    }
    return parseFloat(str) || 0;
}

function formatTime(d) {
    const p = n => n.toString().padStart(2, '0');
    return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

// Custom status badge rendering
function statusBadge(code) {
    if (code >= 200 && code < 300) return `<span class="badge-ok">${code}</span>`;
    if (code >= 400 && code < 500) return `<span class="badge-warn">${code}</span>`;
    return `<span class="badge-err">${code}</span>`;
}

function setButtonState(btn, state) {
    const dict = i18n[currentLang];
    if (state === 'saving') {
        btn.disabled = true;
        btn.textContent = dict.status_saving;
        btn.style.opacity = '0.7';
    } else if (state === 'success') {
        btn.disabled = true;
        btn.className = 'btn-primary btn-success';
        btn.textContent = dict.status_success;
        btn.style.opacity = '1';
    } else {
        btn.disabled = false;
        btn.className = 'btn-primary';
        btn.textContent = dict.btn_save_config;
        btn.style.opacity = '1';
    }
}

function copyText(text, btn) {
    const dict = i18n[currentLang];
    const doCopy = () => {
        const span = btn.querySelector('span');
        const orig = span.textContent;
        span.textContent = dict.btn_copied;
        btn.style.borderColor = 'var(--green-border)';
        btn.style.color = 'var(--green)';
        setTimeout(() => {
            span.textContent = orig;
            btn.style.borderColor = '';
            btn.style.color = '';
        }, 1200);
    };

    if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(doCopy).catch(e => console.error(e));
    } else {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        document.body.appendChild(ta);
        ta.select();
        try { document.execCommand('copy'); doCopy(); } catch (e) { console.error(e); }
        document.body.removeChild(ta);
    }
}
