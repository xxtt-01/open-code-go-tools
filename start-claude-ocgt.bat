@echo off
title Claude Code via ocgt
chcp 65001 > nul
setlocal
pushd "%~dp0"

set "OCGT_EXE=%CD%\build\bin\ocgt.exe"
set "OCGT_BASE_URL=http://127.0.0.1:8787"

echo ===================================================
echo   Claude Code via ocgt local proxy
echo ===================================================
echo.

if not exist "%OCGT_EXE%" (
    echo ocgt.exe not found. Building first...
    call "%~dp0start-ocgt.bat"
)

powershell -NoProfile -ExecutionPolicy Bypass -Command "$ok=$false; try { $r=Invoke-WebRequest -UseBasicParsing -Uri '%OCGT_BASE_URL%/healthz' -TimeoutSec 1; $ok=($r.StatusCode -eq 200) } catch {}; if (-not $ok) { Start-Process -FilePath '%OCGT_EXE%'; for ($i=0; $i -lt 40; $i++) { try { $r=Invoke-WebRequest -UseBasicParsing -Uri '%OCGT_BASE_URL%/healthz' -TimeoutSec 1; if ($r.StatusCode -eq 200) { exit 0 } } catch {}; Start-Sleep -Milliseconds 500 }; exit 1 }"
if errorlevel 1 (
    echo ocgt proxy did not become ready on %OCGT_BASE_URL%.
    pause
    popd
    exit /b 1
)

set "ANTHROPIC_BASE_URL=%OCGT_BASE_URL%"
set "ANTHROPIC_API_KEY=ocgt-local-proxy"
set "ANTHROPIC_AUTH_TOKEN="
set "ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile: opencode-go"
set "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1"
set "CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1"
set "ANTHROPIC_MODEL=deepseek-v4-pro"

echo Proxy: %ANTHROPIC_BASE_URL%
echo Model: %ANTHROPIC_MODEL%
echo.
echo Starting Claude Code...
echo.

claude

popd
endlocal
