package quota

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	openCodeGoBaseURL   = "https://opencode.ai"
	openCodeGoServiceID = "c7389bd0e731f80f49593e5ee53835475f4e28594dd6bd83eb229bab753498cd"
)

// QuotaUsage represents quota usage for a single time dimension.
// Mirrors the opencode-tui-usage QuotaUsage type.
type QuotaUsage struct {
	Status       string `json:"status"`        // "active" or "unlimited"
	UsagePercent int    `json:"usage_percent"`  // 0–100
	ResetInSec   int    `json:"reset_in_sec"`
	ResetDisplay string `json:"reset_display"`  // compact duration: "2h", "30m", "5d"
}

// QuotaData holds rolling / weekly / monthly quota info.
type QuotaData struct {
	Rolling   QuotaUsage  `json:"rolling"`
	Weekly    QuotaUsage  `json:"weekly"`
	Monthly   *QuotaUsage `json:"monthly,omitempty"`
	FetchedAt time.Time   `json:"fetched_at"`
}

// QuotaResult is the JSON API response envelope.
type QuotaResult struct {
	Success      bool       `json:"success"`
	ProviderName string     `json:"provider_name"`
	Data         *QuotaData `json:"data,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// FetchOpenCodeGoQuota calls the opencode.ai internal RPC endpoint to retrieve
// current quota usage (rolling / weekly / monthly). It mirrors the logic in
// @yinxe/opencode-tui-usage OpenCodeGoQuotaProvider.fetchQuota().
//
// Caller must provide the browser cookie and workspace ID:
//
//	OPENCODE_GO_AUTH_COOKIE  — full cookie string from opencode.ai
//	OPENCODE_GO_WORKSPACE_ID — wrk_xxxxxxxxxxxx
func FetchOpenCodeGoQuota(cookie, workspaceID string) (*QuotaData, error) {
	if cookie == "" {
		return nil, fmt.Errorf("OpenCode Go auth cookie not configured (see quota_cookie in profile config)")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("OpenCode Go workspace ID not configured (see quota_workspace_id in profile config)")
	}

	args := buildRPCArgs(workspaceID)
	reqURL := fmt.Sprintf("%s/_server?id=%s&args=%s",
		openCodeGoBaseURL, openCodeGoServiceID, url.QueryEscape(args))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("accept", "*/*")
	req.Header.Set("cookie", cookie)
	req.Header.Set("x-server-id", openCodeGoServiceID)
	req.Header.Set("x-server-instance", "server-fn:3")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("quota API returned %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return parseQuotaResponse(string(body))
}

// CredentialsFromEnv reads OpenCode Go credentials from environment variables.
// This mirrors the fallback logic in opencode-tui-usage's OpenCodeGoQuotaProvider.init().
func CredentialsFromEnv() (cookie, workspaceID string) {
	cookie = os.Getenv("OPENCODE_GO_AUTH_COOKIE")
	workspaceID = os.Getenv("OPENCODE_GO_WORKSPACE_ID")
	return
}

// buildRPCArgs builds the JSON-encoded RPC args string.
// Equivalent to the JS literal in opencode-tui-usage:
//
//	{t:{t:9,i:0,l:1,a:[{t:1,s:workspaceId}],o:0},f:31,m:[]}
func buildRPCArgs(workspaceID string) string {
	raw := map[string]any{
		"t": map[string]any{
			"t": 9,
			"i": 0,
			"l": 1,
			"a": []any{
				map[string]any{"t": 1, "s": workspaceID},
			},
			"o": 0,
		},
		"f": 31,
		"m": []any{},
	}
	data, _ := json.Marshal(raw)
	return string(data)
}

// parseQuotaResponse extracts quota data from the RPC response text.
// The upstream returns a Go-encoding-like format:
//
//	rollingUsage:$R[1]={status:"active",resetInSec:3600,usagePercent:45}
//	weeklyUsage:$R[2]={status:"active",resetInSec:604800,usagePercent:30}
//	monthlyUsage:$R[3]={status:"unlimited",resetInSec:0,usagePercent:0}
func parseQuotaResponse(text string) (*QuotaData, error) {
	rollingMatch := regexp.MustCompile(
		`rollingUsage:\$R\[1\]=\{status:"([^"]+)",resetInSec:(\d+),usagePercent:(\d+)\}`,
	).FindStringSubmatch(text)

	weeklyMatch := regexp.MustCompile(
		`weeklyUsage:\$R\[2\]=\{status:"([^"]+)",resetInSec:(\d+),usagePercent:(\d+)\}`,
	).FindStringSubmatch(text)

	monthlyMatch := regexp.MustCompile(
		`monthlyUsage:\$R\[3\]=\{status:"([^"]+)",resetInSec:(\d+),usagePercent:(\d+)\}`,
	).FindStringSubmatch(text)

	if rollingMatch == nil {
		return nil, fmt.Errorf("failed to parse rollingUsage from response")
	}
	if weeklyMatch == nil {
		return nil, fmt.Errorf("failed to parse weeklyUsage from response")
	}

	rolling := parseUsage(rollingMatch)
	weekly := parseUsage(weeklyMatch)

	var monthly *QuotaUsage
	if monthlyMatch != nil {
		m := parseUsage(monthlyMatch)
		if m.Status != "unlimited" {
			monthly = &m
		}
	}

	return &QuotaData{
		Rolling:   rolling,
		Weekly:    weekly,
		Monthly:   monthly,
		FetchedAt: time.Now(),
	}, nil
}

func parseUsage(matches []string) QuotaUsage {
	reset, _ := strconv.Atoi(matches[2])
	percent, _ := strconv.Atoi(matches[3])
	return QuotaUsage{
		Status:       matches[1],
		UsagePercent: percent,
		ResetInSec:   reset,
		ResetDisplay: formatDurationCompact(reset),
	}
}

// formatDurationCompact formats a duration in seconds as a compact human-readable string.
// Mirrors formatters.ts formatDurationCompact() from opencode-tui-usage.
//
//	45  → "45s"
//	90  → "2m"
//	3600  → "1h"
//	86400 → "1d"
func formatDurationCompact(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	return fmt.Sprintf("%dd", seconds/86400)
}
