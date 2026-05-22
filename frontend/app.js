// 全局状态变量
let API_BASE = 'http://127.0.0.1:8787';
let systemStatus = null;
let currentShell = 'powershell';
let proxyReady = false;

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
const statusPill = document.getElementById('statusPill');
const uptimeBadge = document.querySelector('.uptime-badge');

document.addEventListener('DOMContentLoaded', () => {
    setupEventHandlers();
    initializeApp();
    setInterval(() => {
        if (proxyReady) {
            loadHistory();
        } else {
            initializeApp();
        }
    }, 2500);
});

async function initializeApp() {
    setProxyConnectionState('connecting');
    await resolveApiBase();
    proxyReady = await waitForProxyReady();
    if (!proxyReady) {
        setProxyConnectionState('offline');
        return;
    }
    setProxyConnectionState('online');
    await Promise.all([loadStatus(), loadProfiles(), loadHistory()]);
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

async function waitForProxyReady(timeoutMs = 12000) {
    const started = Date.now();
    while (Date.now() - started < timeoutMs) {
        try {
            const resp = await apiFetch('/healthz', { cache: 'no-store' }, 1200);
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
    const meta = {
        connecting: { text: '代理连接中', className: 'connecting' },
        online: { text: '代理已连接', className: 'online' },
        offline: { text: '代理未连接', className: 'offline' }
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
        elModel.textContent = systemStatus.default_model || '未设定';
        elConfigPath.textContent = systemStatus.config_path;
        if (elApiKey) {
            elApiKey.textContent = systemStatus.api_key_configured === false ? '未配置' : '已配置';
            elApiKey.style.color = systemStatus.api_key_configured === false ? 'var(--yellow)' : 'var(--green)';
        }
        if (elTimeout) {
            const seconds = Number(systemStatus.request_timeout_seconds || 300);
            elTimeout.textContent = `${seconds}s`;
            if (inputTimeout && !document.activeElement.isSameNode(inputTimeout)) {
                inputTimeout.value = seconds.toString();
            }
        }
        // Show auth status
        const elAuth = document.getElementById('status-auth');
        if (elAuth) {
            elAuth.textContent = systemStatus.auth_enabled ? '已启用' : '未启用';
            elAuth.style.color = systemStatus.auth_enabled ? 'var(--green)' : 'var(--gray)';
        }
        renderEnvCode();
        renderCCSwitchCode();
        setProxyConnectionState('online');
        if (systemStatus.api_key_configured === false && uptimeBadge) {
            const text = uptimeBadge.querySelector('span:last-child');
            if (text) text.textContent = '代理已连接，密钥未配置';
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
    } catch (err) {
        console.error('Error loading profiles:', err);
    }
}

async function loadHistory() {
    try {
        const resp = await apiFetch('/ocgt/api/history');
        if (!resp.ok) throw new Error('Failed');
        renderHistoryTable(await resp.json());
    } catch (err) {
        console.error('Error loading history:', err);
    }
}

function setupEventHandlers() {
    // 侧边栏多工作区视图切换
    const navItems = document.querySelectorAll('.nav-item');
    const views = document.querySelectorAll('.view');
    const titleEl = document.getElementById('current-view-title');
    const subtitleEl = document.getElementById('current-view-subtitle');

    const viewMeta = {
        dashboard: { title: "系统状态监控", subtitle: "查看当前代理服务运行指标与后台状态" },
        settings: { title: "一键配置管理中心", subtitle: "快速设置您的 API 密钥与高阶 Claude 模型代理映射" },
        terminal: { title: "一键控制台激活", subtitle: "一键启动已载入代理环境变量的原生命令行窗口" },
        history: { title: "流量雷达监控", subtitle: "实时捕获并通过仪表盘统计来自 Claude Code 的 API 请求日志" }
    };

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
            const meta = viewMeta[targetViewId];
            if (meta) {
                titleEl.textContent = meta.title;
                subtitleEl.textContent = meta.subtitle;
            }
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
        const timeoutNumber = Number(timeoutSeconds);

        if (!Number.isInteger(timeoutNumber) || timeoutNumber < 1 || timeoutNumber > 3600) {
            alert('请求超时必须是 1 到 3600 之间的整数秒。');
            return;
        }

        setButtonState(btnSaveAllConfig, 'saving');

        if (window.go && window.go.main && window.go.main.App) {
            try {
                const res = await window.go.main.App.SaveProfileConfig(pName, key, defModel, sonnet, haiku, opus, timeoutSeconds);
                if (res === "success") {
                    setButtonState(btnSaveAllConfig, 'success');
                    await installClaudeUserEnv(false);
                    await loadStatus();
                    await loadProfiles();
                    setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500);
                } else {
                    setButtonState(btnSaveAllConfig, 'idle');
                    alert('保存失败: ' + res);
                }
            } catch (err) {
                console.error('Failed to save config via Wails:', err);
                setButtonState(btnSaveAllConfig, 'idle');
                alert('保存出错: ' + err.message);
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
                        request_timeout_seconds: timeoutNumber
                    })
                });
                if (resp.ok) {
                    setButtonState(btnSaveAllConfig, 'success');
                    await loadStatus();
                    await loadProfiles();
                    setTimeout(() => setButtonState(btnSaveAllConfig, 'idle'), 1500);
                } else {
                    setButtonState(btnSaveAllConfig, 'idle');
                    alert('保存失败，请检查控制台。');
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
                btnLaunchTerminal.innerHTML = '<span class="status-dot pulse" style="background-color:white;width:6px;height:6px;margin-right:8px;"></span>启动中...';
                try {
                    const res = await window.go.main.App.LaunchClaudeTerminal(currentShell);
                    if (res === "success") {
                        btnLaunchTerminal.innerHTML = '已启动终端 ✓';
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
                        alert('启动失败: ' + res);
                    }
                } catch (err) {
                    btnLaunchTerminal.disabled = false;
                    btnLaunchTerminal.innerHTML = originalText;
                    btnLaunchTerminal.style.background = '';
                    console.error("Launch terminal error:", err);
                    alert('启动终端发生错误: ' + err.message);
                }
            } else {
                alert('一键启动终端仅在桌面版 app 客户端可用，请在桌面端中点击使用！');
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

    // 绑定四个模型下拉框的“自定义”选择逻辑
    const handleModelSelectChange = (selectEl) => {
        if (!selectEl) return;
        selectEl.addEventListener('change', (e) => {
            if (e.target.value === 'custom') {
                const newVal = prompt("请输入您想映射或设定的自定义模型名称 (例如: my-custom-model):");
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
                alert("该功能仅在桌面客户端可用。您的配置文件夹通常在您的个人用户目录下的 .ocgt 文件夹中。");
            }
        });
    }
}

async function installClaudeUserEnv(showAlert) {
    if (!(window.go && window.go.main && window.go.main.App && window.go.main.App.InstallClaudeUserEnv)) {
        if (showAlert) {
            alert('该功能仅桌面端可用。当前浏览器模式请复制右侧环境变量手动执行。');
        }
        return false;
    }

    const buttons = [btnInstallEnv, btnInstallEnvTerminal].filter(Boolean);
    buttons.forEach(btn => {
        btn.disabled = true;
        btn.dataset.originalText = btn.textContent;
        btn.textContent = '修复中...';
    });

    try {
        const res = await window.go.main.App.InstallClaudeUserEnv();
        if (res === 'success') {
            buttons.forEach(btn => {
                btn.textContent = '已修复，重新打开终端生效';
            });
            if (showAlert) {
                alert('已写入 Windows 用户环境变量。请关闭旧 PowerShell，重新打开终端后再运行 claude。');
            }
            setTimeout(() => {
                buttons.forEach(btn => {
                    btn.disabled = false;
                    btn.textContent = btn.dataset.originalText || '一键修复 Claude Code 系统环境变量';
                });
            }, 2200);
            return true;
        }
        if (showAlert) alert('修复失败: ' + res);
        return false;
    } catch (err) {
        console.error('InstallClaudeUserEnv failed:', err);
        if (showAlert) alert('修复失败: ' + err.message);
        return false;
    } finally {
        if (!buttons.some(btn => btn.textContent === '已修复，重新打开终端生效')) {
            buttons.forEach(btn => {
                btn.disabled = false;
                btn.textContent = btn.dataset.originalText || '一键修复 Claude Code 系统环境变量';
            });
        }
    }
}

function renderEnvCode() {
    if (!systemStatus) return;
    const { listen, active_profile: profile } = systemStatus;
    const model = systemStatus.default_model || 'kimi-k2.6';
    const templates = {
        powershell: `$env:ANTHROPIC_BASE_URL = "http://${listen}"\n$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"\n$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: ${profile}"\n$env:ANTHROPIC_MODEL = "${model}"`,
        bash: `export ANTHROPIC_BASE_URL="http://${listen}"\nexport ANTHROPIC_API_KEY="ocgt-local-proxy"\nexport ANTHROPIC_CUSTOM_HEADERS="X-Ocgt-Profile: ${profile}"\nexport ANTHROPIC_MODEL="${model}"`,
        cmd: `set ANTHROPIC_BASE_URL=http://${listen}\nset ANTHROPIC_API_KEY=ocgt-local-proxy\nset ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile:${profile}\nset ANTHROPIC_MODEL=${model}`
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
    // 动态更新流量仪表盘统计大屏
    updateTrafficStats(logs);

    if (!logs || logs.length === 0) {
        tbodyHistory.innerHTML = `<tr class="empty-row"><td colspan="7"><div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>暂无流量记录。请使用一键终端或在其他 Shell 中向代理发送请求...</div></td></tr>`;
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

function statusBadge(code) {
    if (code >= 200 && code < 300) return `<span class="badge-ok">${code}</span>`;
    if (code >= 400 && code < 500) return `<span class="badge-warn">${code}</span>`;
    return `<span class="badge-err">${code}</span>`;
}

function setButtonState(btn, state) {
    if (state === 'saving') {
        btn.disabled = true;
        btn.textContent = '保存中...';
        btn.style.opacity = '0.7';
    } else if (state === 'success') {
        btn.disabled = true;
        btn.className = 'btn-primary btn-success';
        btn.textContent = '已保存';
        btn.style.opacity = '1';
    } else {
        btn.disabled = false;
        btn.className = 'btn-primary';
        btn.textContent = '保存并热重载配置';
        btn.style.opacity = '1';
    }
}

function copyText(text, btn) {
    const doCopy = () => {
        const span = btn.querySelector('span');
        const orig = span.textContent;
        span.textContent = '已复制';
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
