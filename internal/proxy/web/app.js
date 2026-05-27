// ── ocgt Web Dashboard v0.2.1 ──
// Premium polished web control panel

let API_BASE = 'http://127.0.0.1:8787';
let systemStatus = null;
let currentShell = 'powershell';
let proxyReady = false;

// ── Toast System ──
function toast(message, type) {
    type = type || 'info';
    const container = document.getElementById('toastContainer');
    if (!container) return;
    const el = document.createElement('div');
    el.className = `toast toast-${type}`;
    const icons = {
        success: '<svg style="width:16px;height:16px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>',
        error: '<svg style="width:16px;height:16px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
        warning: '<svg style="width:16px;height:16px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
        info: '<svg style="width:16px;height:16px" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
    };
    el.innerHTML = (icons[type] || icons.info) + `<span class="toast-msg">${escapeHtml(message)}</span><button class="toast-close" aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>`;
    container.appendChild(el);
    const closeBtn = el.querySelector('.toast-close');
    const dismiss = () => {
        if (el.classList.contains('toast-out')) return;
        el.classList.add('toast-out');
        el.addEventListener('animationend', () => { if (el.parentNode) el.remove(); }, { once: true });
    };
    closeBtn.addEventListener('click', dismiss);
    const timer = setTimeout(dismiss, 4000);
    el.addEventListener('mouseenter', () => clearTimeout(timer));
    el.addEventListener('mouseleave', () => { const t = setTimeout(dismiss, 2000); el._timer = t; });
}

function escapeHtml(value) {
    return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

function delay(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }

async function apiFetch(path, options = {}, timeoutMs = 8000) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    try { return await fetch(`${API_BASE}${path}`, { ...options, signal: controller.signal }); }
    finally { clearTimeout(timeout); }
}

// ── DOM ──
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
const statusText = document.getElementById('status-text');
const inputCloseBehavior = document.getElementById('close-behavior-input');
const btnAboutApp = document.getElementById('about-app-btn');
const aboutDialogOverlay = document.getElementById('aboutDialogOverlay');
const aboutDialogClose = document.getElementById('aboutDialogClose');

// ── Modal helpers ──
function showModal(el) { if (el) { el.classList.add('active'); el.setAttribute('aria-hidden', 'false'); } }
function hideModal(el) { if (el) { el.classList.remove('active'); el.setAttribute('aria-hidden', 'true'); } }

// ── Init ──
document.addEventListener('DOMContentLoaded', () => {
    setupEventHandlers();
    initializeApp();
    setInterval(() => { if (proxyReady) loadHistory(); else initializeApp(); }, 2500);
});

async function initializeApp() {
    setProxyConnectionState('connecting');
    await resolveApiBase();
    proxyReady = await waitForProxyReady();
    if (!proxyReady) { setProxyConnectionState('offline'); return; }
    setProxyConnectionState('online');
    await Promise.all([loadStatus(), loadProfiles(), loadHistory(), loadPreferences()]);
}

async function resolveApiBase() {
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.GetListenAddress)) return;
    try {
        const addr = await window.go.main.App.GetListenAddress();
        if (addr) API_BASE = `http://${addr}`;
    } catch (err) { console.error("Wails GetListenAddress error:", err); }
}

async function waitForProxyReady(timeoutMs = 2500) {
    const started = Date.now();
    while (Date.now() - started < timeoutMs) {
        try { const resp = await apiFetch('/healthz', { cache: 'no-store' }, 700); if (resp.ok) return true; }
        catch (err) { /* retry */ }
        await delay(350);
    }
    return false;
}

async function isProxyHealthy() {
    try {
        const resp = await apiFetch('/healthz', { cache: 'no-store' }, 1200);
        return resp.ok;
    } catch (err) {
        return false;
    }
}

function setProxyConnectionState(state) {
    if (!statusText) return;
    const texts = { connecting: '代理连接中', online: '代理已连接', offline: '代理未连接' };
    statusText.textContent = texts[state] || texts.offline;
    const dot = document.querySelector('.status-dot');
    if (dot) {
        if (state === 'online') dot.style.background = 'var(--success)';
        else if (state === 'connecting') dot.style.background = 'var(--warning)';
        else dot.style.background = 'var(--danger)';
    }
}

// ── Load Status ──
async function loadStatus() {
    try {
        const resp = await apiFetch('/ocgt/api/status');
        if (!resp.ok) throw new Error('Failed');
        systemStatus = await resp.json();
        elListen.textContent = systemStatus.listen;
        elUpstream.textContent = systemStatus.upstream;
        elProfile.textContent = systemStatus.active_profile;
        elModel.textContent = systemStatus.default_model || '未设定';
        elConfigPath.textContent = systemStatus.config_path;
        if (elApiKey) {
            elApiKey.textContent = systemStatus.api_key_configured === false ? '未配置' : '已配置';
            elApiKey.style.color = systemStatus.api_key_configured === false ? 'var(--warning)' : 'var(--success)';
        }
        if (elTimeout) {
            const seconds = Number(systemStatus.request_timeout_seconds || 300);
            elTimeout.textContent = `${seconds}s`;
            if (inputTimeout && !document.activeElement.isSameNode(inputTimeout)) inputTimeout.value = seconds.toString();
        }
        if (inputThinkingBudget) {
            const budget = Number(systemStatus.max_thinking_budget_tokens ?? 512);
            if (!document.activeElement.isSameNode(inputThinkingBudget)) setThinkingBudgetValue(budget.toString());
        }
        const elAuth = document.getElementById('status-auth');
        if (elAuth) {
            elAuth.textContent = systemStatus.auth_enabled ? '已启用' : '未启用';
            elAuth.style.color = systemStatus.auth_enabled ? 'var(--success)' : 'var(--text-muted)';
        }
        renderEnvCode();
        renderCCSwitchCode();
        setProxyConnectionState('online');
        return true;
    } catch (err) {
        console.error('Error fetching status:', err);
        setProxyConnectionState(await isProxyHealthy() ? 'online' : 'offline');
        return false;
    }
}

function setSelectValue(selectEl, value) {
    if (!selectEl) return;
    if (!value) { selectEl.selectedIndex = 0; return; }
    let exists = false;
    for (let i = 0; i < selectEl.options.length; i++) {
        if (selectEl.options[i].value === value) { selectEl.value = value; exists = true; break; }
    }
    if (!exists) {
        const opt = document.createElement('option'); opt.value = value; opt.textContent = value;
        selectEl.insertBefore(opt, selectEl.lastElementChild); selectEl.value = value;
    }
}

function setThinkingBudgetValue(value) {
    if (!inputThinkingBudget) return;
    const allowed = ['256', '512', '1024', '2048', '-1'];
    if (allowed.includes(value)) { inputThinkingBudget.value = value; return; }
    let opt = Array.from(inputThinkingBudget.options).find(item => item.value === value);
    if (!opt) {
        opt = document.createElement('option'); opt.value = value;
        opt.textContent = `${value} · 当前自定义值`;
        inputThinkingBudget.insertBefore(opt, inputThinkingBudget.lastElementChild);
    }
    inputThinkingBudget.value = value;
}

function isAllowedThinkingBudget(value) { return ['256', '512', '1024', '2048', '-1'].includes(value); }

// ── Load Profiles ──
async function loadProfiles() {
    try {
        const resp = await apiFetch('/ocgt/api/profiles');
        if (!resp.ok) throw new Error('Failed');
        const data = await resp.json();
        selectProfile.innerHTML = '';
        Object.keys(data.profiles).forEach(pName => {
            const opt = document.createElement('option'); opt.value = pName; opt.textContent = pName;
            if (pName === data.active_profile) opt.selected = true;
            selectProfile.appendChild(opt);
        });
        const ap = data.profiles[data.active_profile];
        if (ap) {
            inputApiKey.value = ap.api_key || '';
            setSelectValue(inputDefaultModel, ap.default_model || '');
            const aliases = ap.model_aliases || {};
            setSelectValue(inputSonnetMapping, aliases.sonnet || '');
            setSelectValue(inputHaikuMapping, aliases.haiku || '');
            setSelectValue(inputOpusMapping, aliases.opus || '');
        }
    } catch (err) { console.error('Error loading profiles:', err); }
}

// ── Load Preferences ──
async function loadPreferences() {
    if (!inputCloseBehavior) return;
    const DEFAULT_CLOSE_BEHAVIOR = 'prompt';
    const CLOSE_BEHAVIORS = new Set(['prompt', 'minimize', 'exit']);
    const normalize = v => CLOSE_BEHAVIORS.has(v) ? v : DEFAULT_CLOSE_BEHAVIOR;
    
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.GetPreferences)) {
        inputCloseBehavior.value = DEFAULT_CLOSE_BEHAVIOR;
        return;
    }
    try {
        const prefs = await window.go.main.App.GetPreferences();
        inputCloseBehavior.value = normalize(prefs && prefs.close_behavior);
    } catch (err) { console.error('Failed to load preferences:', err); inputCloseBehavior.value = DEFAULT_CLOSE_BEHAVIOR; }
}

// ── Load History ──
async function loadHistory() {
    try {
        const resp = await apiFetch('/ocgt/api/history');
        if (!resp.ok) throw new Error('Failed');
        renderHistoryTable(await resp.json());
    } catch (err) { console.error('Error loading history:', err); }
}

// ── Event Handlers ──
function setupEventHandlers() {
    // Theme toggle
    if (themeToggleBtn) {
        themeToggleBtn.addEventListener('click', () => {
            const current = document.documentElement.getAttribute('data-theme') || 'light';
            const next = current === 'light' ? 'dark' : 'light';
            document.documentElement.setAttribute('data-theme', next);
            localStorage.setItem('theme', next);
        });
    }

    // Profile change
    selectProfile.addEventListener('change', async (e) => {
        try {
            const resp = await apiFetch('/ocgt/api/profiles/active', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ profile: e.target.value })
            });
            if (resp.ok) { toast('Profile 已切换', 'success'); await loadStatus(); await loadProfiles(); }
        } catch (err) { console.error('Failed to change profile:', err); }
    });

    // Password visibility
    btnToggleVisibility.addEventListener('click', () => {
        inputApiKey.type = inputApiKey.type === 'password' ? 'text' : 'password';
    });

    // Save config
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
            toast('请求超时必须是 1 到 3600 之间的整数秒。', 'error'); return;
        }
        if (!isAllowedThinkingBudget(thinkingBudget)) {
            toast('请选择有效的思考强度。', 'error'); return;
        }

        setButtonState(btnSaveAllConfig, 'saving');

        if (window.go && window.go.main && window.go.main.App) {
            try {
                const claudeEnvJSON = JSON.stringify(systemStatus && systemStatus.claude_env ? systemStatus.claude_env : {});
                const res = await window.go.main.App.SaveProfileConfig(pName, key, defModel, sonnet, haiku, opus, timeoutSeconds, thinkingBudget, '', '', '', claudeEnvJSON);
                if (res === "success") {
                    if (inputCloseBehavior && window.go.main.App.SavePreferences) {
                        const prefs = normalizeCloseBehavior(inputCloseBehavior.value);
                        await window.go.main.App.SavePreferences(prefs);
                    }
                    setButtonState(btnSaveAllConfig, 'success');
                    toast('配置已保存并立即生效', 'success');
                    await loadStatus(); await loadProfiles();
                    setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500);
                } else { setButtonState(btnSaveAllConfig, 'idle'); toast('保存失败: ' + res, 'error'); }
            } catch (err) { console.error('Failed to save config via Wails:', err); setButtonState(btnSaveAllConfig, 'idle'); toast('保存出错: ' + err.message, 'error'); }
        } else {
            try {
                const resp = await apiFetch('/ocgt/api/key', {
                    method: 'POST', headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ profile: pName, api_key: key, default_model: defModel,
                        model_aliases: { sonnet, haiku, opus }, request_timeout_seconds: timeoutNumber,
                        max_thinking_budget_tokens: Number(thinkingBudget) })
                });
                if (resp.ok) { setButtonState(btnSaveAllConfig, 'success'); toast('配置已保存并立即生效', 'success'); await loadStatus(); await loadProfiles(); setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500); }
                else { setButtonState(btnSaveAllConfig, 'idle'); toast('保存失败', 'error'); }
            } catch (err) { console.error('Fallback save error:', err); setButtonState(btnSaveAllConfig, 'idle'); }
        }
    });

    // Close behavior
    if (inputCloseBehavior) {
        inputCloseBehavior.addEventListener('change', async () => {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SavePreferences) {
                try { await window.go.main.App.SavePreferences(normalizeCloseBehavior(inputCloseBehavior.value)); }
                catch (err) { console.error('Failed to save close behavior:', err); }
            }
        });
    }

    // Install env
    if (btnInstallEnv) btnInstallEnv.addEventListener('click', () => installClaudeUserEnv(true));
    if (btnInstallEnvTerminal) btnInstallEnvTerminal.addEventListener('click', () => installClaudeUserEnv(true));

    // Launch terminal
    if (btnLaunchTerminal) {
        btnLaunchTerminal.addEventListener('click', async () => {
            if (window.go && window.go.main && window.go.main.App) {
                btnLaunchTerminal.disabled = true;
                const original = btnLaunchTerminal.innerHTML;
                btnLaunchTerminal.innerHTML = '启动中...';
                try {
                    const res = await window.go.main.App.LaunchClaudeTerminal(currentShell);
                    if (res === "success") {
                        btnLaunchTerminal.innerHTML = '已启动终端 ✓';
                        btnLaunchTerminal.classList.add('btn-success');
                        toast('终端已启动', 'success');
                        setTimeout(() => { btnLaunchTerminal.disabled = false; btnLaunchTerminal.innerHTML = original; btnLaunchTerminal.classList.remove('btn-success'); }, 2000);
                    } else { btnLaunchTerminal.disabled = false; btnLaunchTerminal.innerHTML = original; toast('启动失败: ' + res, 'error'); }
                } catch (err) { btnLaunchTerminal.disabled = false; btnLaunchTerminal.innerHTML = original; toast('启动终端发生错误: ' + err.message, 'error'); }
            } else { toast('一键启动终端仅在桌面版 app 客户端可用', 'warning'); }
        });
    }

    // Shell tabs
    shellTabs.addEventListener('click', (e) => {
        const btn = e.target.closest('.tab-btn');
        if (!btn) return;
        shellTabs.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        currentShell = btn.dataset.shell;
        renderEnvCode();
    });

    // Copy buttons
    btnCopyEnv.addEventListener('click', () => copyText(codeEnv.innerText, btnCopyEnv));
    btnCopyCCSwitch.addEventListener('click', () => copyText(codeCCSwitch.innerText, btnCopyCCSwitch));

    // Custom model handlers
    const handleModelSelectChange = (selectEl) => {
        if (!selectEl) return;
        selectEl.addEventListener('change', (e) => {
            if (e.target.value === 'custom') {
                const newVal = window.prompt("请输入您想映射或设定的自定义模型名称 (例如: my-custom-model):");
                if (newVal && newVal.trim()) {
                    const value = newVal.trim();
                    let exists = false;
                    for (let i = 0; i < selectEl.options.length; i++) {
                        if (selectEl.options[i].value === value) { selectEl.selectedIndex = i; exists = true; break; }
                    }
                    if (!exists) {
                        const opt = document.createElement('option'); opt.value = value; opt.textContent = value;
                        selectEl.insertBefore(opt, selectEl.lastElementChild); selectEl.value = value;
                    }
                } else { selectEl.selectedIndex = 0; }
            }
        });
    };
    handleModelSelectChange(inputDefaultModel);
    handleModelSelectChange(inputSonnetMapping);
    handleModelSelectChange(inputHaikuMapping);
    handleModelSelectChange(inputOpusMapping);

    // About modal
    if (btnAboutApp) {
        btnAboutApp.addEventListener('click', () => {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.ShowAboutDialog) {
                window.go.main.App.ShowAboutDialog();
            } else { showModal(aboutDialogOverlay); }
        });
    }
    if (aboutDialogClose) aboutDialogClose.addEventListener('click', () => hideModal(aboutDialogOverlay));
    if (aboutDialogOverlay) aboutDialogOverlay.addEventListener('click', (e) => { if (e.target === aboutDialogOverlay) hideModal(aboutDialogOverlay); });

    // ESC for modals
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') hideModal(aboutDialogOverlay); });

    // Wails events
    if (window.runtime && typeof window.runtime.EventsOn === 'function') {
        window.runtime.EventsOn('nav-to-settings', () => {});
        window.runtime.EventsOn('show-close-dialog', () => {});
        window.runtime.EventsOn('show-about-dialog', () => showModal(aboutDialogOverlay));
    }
}

function normalizeCloseBehavior(value) {
    const CLOSE_BEHAVIORS = new Set(['prompt', 'minimize', 'exit']);
    return CLOSE_BEHAVIORS.has(value) ? value : 'prompt';
}

// ── Install Claude Env ──
async function installClaudeUserEnv(showAlert) {
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.InstallClaudeUserEnv)) {
        if (showAlert) toast('该功能仅桌面端可用。当前浏览器模式请复制右侧环境变量手动执行。', 'info');
        return false;
    }
    const buttons = [btnInstallEnv, btnInstallEnvTerminal].filter(Boolean);
    buttons.forEach(btn => { btn.disabled = true; btn.dataset.originalText = btn.textContent; btn.textContent = '修复中...'; });
    try {
        const res = await window.go.main.App.InstallClaudeUserEnv();
        if (res === 'success') {
            buttons.forEach(btn => { btn.textContent = '已修复，重新打开终端生效'; });
            if (showAlert) toast('已写入 Windows 用户环境变量。请关闭旧 PowerShell，重新打开终端后再运行 claude。', 'success');
            setTimeout(() => { buttons.forEach(btn => { btn.disabled = false; btn.textContent = btn.dataset.originalText || '一键修复 Claude Code 系统环境变量'; }); }, 2200);
            return true;
        }
        if (showAlert) toast('修复失败: ' + res, 'error');
        return false;
    } catch (err) {
        console.error('InstallClaudeUserEnv failed:', err);
        if (showAlert) toast('修复失败: ' + err.message, 'error');
        return false;
    } finally {
        const resetText = '已修复，重新打开终端生效';
        if (!buttons.some(btn => btn.textContent === resetText)) {
            buttons.forEach(btn => { btn.disabled = false; btn.textContent = btn.dataset.originalText || '一键修复 Claude Code 系统环境变量'; });
        }
    }
}

// ── Render env/code ──
function renderEnvCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const thinkingBudget = Number.isInteger(Number(systemStatus.max_thinking_budget_tokens)) ? Number(systemStatus.max_thinking_budget_tokens) : 512;
    const thinkingLines = thinkingBudget < 0
        ? { powershell: `$env:MAX_THINKING_TOKENS = "0"\n$env:CLAUDE_CODE_DISABLE_THINKING = "1"`, bash: `export MAX_THINKING_TOKENS="0"\nexport CLAUDE_CODE_DISABLE_THINKING="1"`, cmd: `set MAX_THINKING_TOKENS=0\nset CLAUDE_CODE_DISABLE_THINKING=1` }
        : { powershell: `$env:MAX_THINKING_TOKENS = "${thinkingBudget}"`, bash: `export MAX_THINKING_TOKENS="${thinkingBudget}"`, cmd: `set MAX_THINKING_TOKENS=${thinkingBudget}` };
    const templates = {
        powershell: `$env:ANTHROPIC_BASE_URL = "http://${listen}"\n$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"\n$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: ${profile}, X-Ocgt-Client: claude-code-cli"\nRemove-Item Env:ANTHROPIC_MODEL -ErrorAction SilentlyContinue\n${thinkingLines.powershell}`,
        bash: `export ANTHROPIC_BASE_URL="http://${listen}"\nexport ANTHROPIC_API_KEY="ocgt-local-proxy"\nexport ANTHROPIC_CUSTOM_HEADERS="X-Ocgt-Profile: ${profile}, X-Ocgt-Client: claude-code-cli"\nunset ANTHROPIC_MODEL\n${thinkingLines.bash}`,
        cmd: `set ANTHROPIC_BASE_URL=http://${listen}\nset ANTHROPIC_API_KEY=ocgt-local-proxy\nset ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile: ${profile}, X-Ocgt-Client: claude-code-cli\nset ANTHROPIC_MODEL=\n${thinkingLines.cmd}`
    };
    codeEnv.innerText = templates[currentShell] || templates.powershell;
}

function renderCCSwitchCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    codeCCSwitch.innerText = JSON.stringify({ name: `ocgt-${profile}`, type: "anthropic", baseURL: `http://${listen}`, apiKey: "ocgt-local-proxy", model: "claude-sonnet-4-5", headers: { "X-Ocgt-Profile": profile } }, null, 2);
}

// ── History table ──
function renderHistoryTable(logs) {
    if (!logs || logs.length === 0) {
        tbodyHistory.innerHTML = `<tr class="empty-row"><td colspan="7"><div class="empty-message"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>暂无流量记录。请使用一键终端或在其他 Shell 中向代理发送请求...</div></td></tr>`;
        return;
    }
    tbodyHistory.innerHTML = logs.map(log => {
        const time = formatTime(new Date(log.time));
        const badge = statusBadge(log.status);
        return `<tr>
            <td style="color:var(--text-muted);font-size:0.85rem;">${time}</td>
            <td class="font-mono" style="color:var(--primary);font-weight:700;">${escapeHtml(log.method)}</td>
            <td class="font-mono" style="max-width:250px;overflow:hidden;text-overflow:ellipsis;" title="${escapeHtml(log.path)}">${escapeHtml(log.path)}</td>
            <td class="font-mono">${escapeHtml(log.model || '-')}</td>
            <td>${badge}</td>
            <td class="font-mono" style="color:var(--text-muted);">${escapeHtml(log.duration)}</td>
            <td class="error-cell" title="${escapeHtml(log.error || '')}">${escapeHtml(log.error || '-')}</td>
        </tr>`;
    }).join('');
}

function formatTime(d) { const p = n => n.toString().padStart(2, '0'); return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`; }

function statusBadge(code) {
    if (code >= 200 && code < 300) return `<span class="status-badge status-2xx">${code}</span>`;
    if (code >= 400 && code < 500) return `<span class="status-badge status-4xx">${code}</span>`;
    return `<span class="status-badge status-5xx">${code}</span>`;
}

function setButtonState(btn, state) {
    if (state === 'saving') { btn.disabled = true; btn.textContent = '保存中...'; btn.style.opacity = '0.7'; }
    else if (state === 'success') { btn.disabled = true; btn.classList.add('btn-success'); btn.textContent = '已保存 ✓'; btn.style.opacity = '1'; }
    else { btn.disabled = false; btn.classList.remove('btn-success'); btn.textContent = '保存并应用配置'; btn.style.opacity = '1'; }
}

function copyText(text, btn) {
    const span = btn.querySelector('span');
    const orig = span.textContent;
    const doCopy = () => {
        span.textContent = '已复制 ✓';
        btn.style.borderColor = 'var(--success-glow)';
        btn.style.color = 'var(--success)';
        setTimeout(() => { span.textContent = orig; btn.style.borderColor = ''; btn.style.color = ''; }, 1500);
    };
    if (navigator.clipboard) { navigator.clipboard.writeText(text).then(doCopy).catch(e => console.error(e)); }
    else {
        const ta = document.createElement('textarea'); ta.value = text; ta.style.position = 'fixed';
        document.body.appendChild(ta); ta.select();
        try { document.execCommand('copy'); doCopy(); } catch (e) { console.error(e); }
        document.body.removeChild(ta);
    }
}
