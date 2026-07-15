package proxy

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/pricing"
)

// ── 响应数据结构 ──

type StatsSummary struct {
	Period     PeriodInfo         `json:"period"`
	Summary    SummaryTotals       `json:"summary"`
	ByModel    []ModelStat         `json:"by_model"`
	ByClient   []ClientStat        `json:"by_client"`
	DailyTrend []DailyStat         `json:"daily_trend,omitempty"`
	PlanUsage  pricing.PlanUsage `json:"plan_usage,omitempty"`
}

type PeriodInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days,omitempty"`
}

// TimeRange 定义查询时间范围。零值表示不限制（全部数据）。
type TimeRange struct {
	From time.Time // 开始（含）
	To   time.Time // 结束（含）
}

// IsAll 返回是否不限制时间范围
func (tr TimeRange) IsAll() bool {
	return tr.From.IsZero() && tr.To.IsZero()
}

// PeriodInfo 将 TimeRange 转换为前端使用的 PeriodInfo
func (tr TimeRange) PeriodInfo(now time.Time) PeriodInfo {
	from := tr.From
	to := tr.To
	if to.IsZero() {
		to = now
	}
	pi := PeriodInfo{
		From: from.Format("2006-01-02"),
		To:   to.Format("2006-01-02"),
	}
	if !from.IsZero() {
		pi.Days = int(now.Sub(from).Hours()/24) + 1
	}
	return pi
}

type SummaryTotals struct {
	TotalRequests          int     `json:"total_requests"`
	SuccessCount           int     `json:"success_count"`
	SuccessRate            float64 `json:"success_rate"`
	AvgLatencyMs           float64 `json:"avg_latency_ms"`
	TotalInputTokens       int64   `json:"total_input_tokens"`
	TotalOutputTokens      int64   `json:"total_output_tokens"`
	TotalCacheReadTokens   int64   `json:"total_cache_read_tokens"`
	TotalCacheCreateTokens int64   `json:"total_cache_create_tokens"`
	TotalTokens            int64   `json:"total_tokens"`
	EstimatedCost          float64 `json:"estimated_cost"`
	CacheHitRate           float64 `json:"cache_hit_rate"`
}

type ModelStat struct {
	Name         string  `json:"name"`
	Requests     int     `json:"requests"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CacheTokens  int64   `json:"cache_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	Cost         float64 `json:"cost_usd"`
	Pct          float64 `json:"pct"`
	CacheHitRate float64 `json:"cache_hit_rate"`
}

type ClientStat struct {
	Name     string `json:"name"`
	Requests int    `json:"requests"`
	Pct      float64 `json:"pct"`
}

type DailyStat struct {
	Date         string `json:"date"`
	Requests     int    `json:"requests"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

// ── API Handlers ──

func (s *Server) apiStatsSummary(w http.ResponseWriter, r *http.Request) {
	tr := s.parseTimeRange(r, 7)
	entries := s.readJSONLLogs(tr)
	if len(entries) == 0 {
		writeJSON(w, http.StatusOK, emptyStats(tr))
		return
	}
	summary := aggregateStats(entries, tr)
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) apiStatsTrend(w http.ResponseWriter, r *http.Request) {
	tr := s.parseTimeRange(r, 30)
	granularity := determineGranularity(tr)
	entries := s.readJSONLLogs(tr)
	if len(entries) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"period":      tr.PeriodInfo(time.Now()),
			"daily":       []DailyStat{},
			"granularity": granularity,
		})
		return
	}
	trend := dailyTrend(entries, granularity)
	writeJSON(w, http.StatusOK, map[string]any{
		"period":      tr.PeriodInfo(time.Now()),
		"daily":       trend,
		"granularity": granularity,
	})
}

func (s *Server) apiStatsModels(w http.ResponseWriter, r *http.Request) {
	tr := s.parseTimeRange(r, 7)
	entries := s.readJSONLLogs(tr)
	if len(entries) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"models": []ModelStat{}})
		return
	}
	models := modelBreakdown(entries)
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

// ── 公共方法，供前端路由注册 ──

func (s *Server) registerStatsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ocgt/api/stats/summary", s.apiStatsSummary)
	mux.HandleFunc("/ocgt/api/stats/trend", s.apiStatsTrend)
	mux.HandleFunc("/ocgt/api/stats/models", s.apiStatsModels)
}

// ── 辅助函数 ──

// parseTimeRange 从 HTTP 请求解析 TimeRange。
// 优先级：from/to > days > defaultDays
//
//	from=2026-07-01&to=2026-07-15  显式范围
//	from=2026-07-01                 从某天到今天
//	days=7                          最近7天（向后兼容）
//	days=-1                         全部
//	无参数                          defaultDays 天
func (s *Server) parseTimeRange(r *http.Request, defaultDays int) TimeRange {
	fromStr := strings.TrimSpace(r.URL.Query().Get("from"))
	toStr := strings.TrimSpace(r.URL.Query().Get("to"))

	// from/to 优先
	if fromStr != "" || toStr != "" {
		var from, to time.Time
		now := time.Now()
		loc := now.Location()

		if fromStr != "" {
			from, _ = time.ParseInLocation("2006-01-02", fromStr, loc)
		}
		if toStr != "" {
			to, _ = time.ParseInLocation("2006-01-02", toStr, loc)
			if !to.IsZero() {
				to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, loc)
			}
		} else {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, loc)
		}
		return TimeRange{From: from, To: to}
	}

	// days 参数（向后兼容）
	daysStr := strings.TrimSpace(r.URL.Query().Get("days"))
	if daysStr == "-1" {
		return TimeRange{} // 全部
	}
	days := parseIntParam(r, "days", defaultDays)
	if days <= 0 {
		return TimeRange{} // 全部
	}
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return TimeRange{
		From: startOfToday.AddDate(0, 0, -(days - 1)),
		To:   time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
	}
}

// parseIntParam 从 URL 查询参数解析整数值，非法或空时返回 defaultVal。
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	if n <= 0 {
		return defaultVal
	}
	return n
}

// readJSONLLogs 从 JSONL 日志目录读取指定时间范围内的所有记录
func (s *Server) readJSONLLogs(tr TimeRange) []requestLogEntry {
	s.historyLogMu.Lock()
	dir := s.historyLogDir
	s.historyLogMu.Unlock()
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".ocgt", "log")
	}

	var allEntries []requestLogEntry

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[stats] readJSONLLogs: cannot read dir %q: %v", dir, err)
		return nil
	}
	log.Printf("[stats] readJSONLLogs: reading %q, found %d files", dir, len(files))

	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "ocgt-") || !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		filePath := filepath.Join(dir, f.Name())
		entries := readJSONLFile(filePath, tr)
		allEntries = append(allEntries, entries...)
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Time.After(allEntries[j].Time)
	})

	// Fallback: 如果 JSONL 文件没有数据（日志未启用或目录不存在），
	// 从内存历史记录读取，确保 stats API 总有数据可返回
	if len(allEntries) == 0 {
		s.historyMu.RLock()
		hist := make([]requestLogEntry, len(s.history))
		copy(hist, s.history)
		s.historyMu.RUnlock()
		if len(hist) > 0 {
			for _, e := range hist {
				if tr.IsAll() || (e.Time.After(tr.From) && (tr.To.IsZero() || e.Time.Before(tr.To))) {
					allEntries = append(allEntries, e)
				}
			}
			log.Printf("[stats] JSONL empty, falling back to in-memory history: %d entries after filter", len(allEntries))
		}
	}

	return allEntries
}

func readJSONLFile(path string, tr TimeRange) []requestLogEntry {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []requestLogEntry
	scanner := bufio.NewScanner(io.LimitReader(f, 50<<20)) // 最多读 50MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry requestLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		// 用 TimeRange 过滤
		if !tr.IsAll() {
			if !tr.From.IsZero() && entry.Time.Before(tr.From) {
				continue
			}
			if !tr.To.IsZero() && entry.Time.After(tr.To) {
				continue
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

func emptyStats(tr TimeRange) StatsSummary {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from := tr.From
	if from.IsZero() {
		from = startOfToday.AddDate(0, 0, -6) // 默认7天
	}
	return StatsSummary{
		Period: PeriodInfo{
			From: from.Format("2006-01-02"),
			To:   now.Format("2006-01-02"),
		},
	}
}

func aggregateStats(entries []requestLogEntry, tr TimeRange) StatsSummary {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	displayFrom := tr.From
	if displayFrom.IsZero() {
		displayFrom = startOfToday.AddDate(0, 0, -6)
	}
	result := StatsSummary{
		Period: TimeRange{From: displayFrom, To: tr.To}.PeriodInfo(now),
	}

	modelMap := make(map[string]*ModelStat)
	clientMap := make(map[string]*ClientStat)
	dayMap := make(map[string]*DailyStat)

	for _, e := range entries {
		// Summary
		result.Summary.TotalRequests++
		if e.Status >= 200 && e.Status < 300 {
			result.Summary.SuccessCount++
		}
		result.Summary.TotalInputTokens += int64(e.InputTokens)
		result.Summary.TotalOutputTokens += int64(e.OutputTokens)
		result.Summary.TotalCacheReadTokens += int64(e.CacheReadTokens)
		result.Summary.TotalCacheCreateTokens += int64(e.CacheCreationTokens)
		result.Summary.TotalTokens += int64(e.TotalTokens)
		result.Summary.AvgLatencyMs += parseDurationFloat(e.Duration)

		// By model
		model := e.Model
		if model == "" {
			model = "unknown"
		}
		if _, ok := modelMap[model]; !ok {
			modelMap[model] = &ModelStat{Name: model}
		}
		ms := modelMap[model]
		ms.Requests++
		ms.InputTokens += int64(e.InputTokens)
		ms.OutputTokens += int64(e.OutputTokens)
		ms.CacheTokens += int64(e.CacheReadTokens + e.CacheCreationTokens)
		ms.TotalTokens += int64(e.TotalTokens)
		ms.Cost += pricing.EstimateCost(model, e.InputTokens, e.OutputTokens, e.CacheReadTokens, e.CacheCreationTokens)

		// By client
		client := e.Client
		if client == "" {
			client = "Unknown"
		}
		if _, ok := clientMap[client]; !ok {
			clientMap[client] = &ClientStat{Name: client}
		}
		clientMap[client].Requests++

		// Daily trend
		dateKey := e.Time.Format("2006-01-02")
		if _, ok := dayMap[dateKey]; !ok {
			dayMap[dateKey] = &DailyStat{Date: dateKey}
		}
		ds := dayMap[dateKey]
		ds.Requests++
		ds.InputTokens += int64(e.InputTokens)
		ds.OutputTokens += int64(e.OutputTokens)
		ds.TotalTokens += int64(e.TotalTokens)
	}

	// Finalize summary
	if result.Summary.TotalRequests > 0 {
		result.Summary.SuccessRate = float64(result.Summary.SuccessCount) / float64(result.Summary.TotalRequests) * 100
		result.Summary.AvgLatencyMs = result.Summary.AvgLatencyMs / float64(result.Summary.TotalRequests)
	}
	if result.Summary.TotalInputTokens > 0 {
		result.Summary.CacheHitRate = float64(result.Summary.TotalCacheReadTokens) / float64(result.Summary.TotalInputTokens) * 100
	}
	result.Summary.EstimatedCost = 0

	// By model — calculate percentages and total cost
	var totalTokens float64
	for _, ms := range modelMap {
		totalTokens += float64(ms.TotalTokens)
		result.Summary.EstimatedCost += ms.Cost
	}
	for _, ms := range modelMap {
		if totalTokens > 0 {
			ms.Pct = float64(ms.TotalTokens) / totalTokens * 100
		}
		result.ByModel = append(result.ByModel, *ms)
	}
	sort.Slice(result.ByModel, func(i, j int) bool {
		return result.ByModel[i].TotalTokens > result.ByModel[j].TotalTokens
	})

	// By client
	var totalReq float64
	for _, cs := range clientMap {
		totalReq += float64(cs.Requests)
	}
	for _, cs := range clientMap {
		if totalReq > 0 {
			cs.Pct = float64(cs.Requests) / totalReq * 100
		}
		result.ByClient = append(result.ByClient, *cs)
	}
	sort.Slice(result.ByClient, func(i, j int) bool {
		return result.ByClient[i].Requests > result.ByClient[j].Requests
	})

	// Daily trend
	for _, ds := range dayMap {
		result.DailyTrend = append(result.DailyTrend, *ds)
	}
	sort.Slice(result.DailyTrend, func(i, j int) bool {
		return result.DailyTrend[i].Date < result.DailyTrend[j].Date
	})

	// Plan usage based on total estimated cost
	result.PlanUsage = pricing.EstimateSpendingUsage(result.Summary.EstimatedCost)

	// "全部"模式下用实际最早数据日期更新 PeriodInfo.From
	if tr.IsAll() && len(entries) > 0 {
		earliest := entries[0].Time
		for _, e := range entries {
			if e.Time.Before(earliest) {
				earliest = e.Time
			}
		}
		result.Period.From = earliest.Format("2006-01-02")
		result.Period.Days = int(now.Sub(earliest).Hours()/24) + 1
	}

	return result
}

func determineGranularity(tr TimeRange) string {
	if tr.IsAll() {
		return "week"
	}
	now := time.Now()
	from := tr.From
	to := tr.To
	if to.IsZero() {
		to = now
	}
	if from.IsZero() {
		return "week"
	}
	span := to.Sub(from)
	switch {
	case span <= 48*time.Hour:
		return "hour"
	case span <= 62*24*time.Hour:
		return "day"
	default:
		return "week"
	}
}

func timeKey(t time.Time, granularity string) string {
	switch granularity {
	case "hour":
		return t.Format("2006-01-02 15:00")
	case "week":
		// Monday as start of week
		weekday := t.Weekday()
		daysFromMonday := int(weekday - time.Monday)
		if daysFromMonday < 0 {
			daysFromMonday += 7
		}
		monday := t.AddDate(0, 0, -daysFromMonday)
		return monday.Format("2006-01-02")
	default: // "day"
		return t.Format("2006-01-02")
	}
}

func dailyTrend(entries []requestLogEntry, granularity string) []DailyStat {
	dayMap := make(map[string]*DailyStat)
	for _, e := range entries {
		dateKey := timeKey(e.Time, granularity)
		if _, ok := dayMap[dateKey]; !ok {
			dayMap[dateKey] = &DailyStat{Date: dateKey}
		}
		ds := dayMap[dateKey]
		ds.Requests++
		ds.InputTokens += int64(e.InputTokens)
		ds.OutputTokens += int64(e.OutputTokens)
		ds.TotalTokens += int64(e.TotalTokens)
	}
	var trend []DailyStat
	for _, ds := range dayMap {
		trend = append(trend, *ds)
	}
	sort.Slice(trend, func(i, j int) bool {
		return trend[i].Date < trend[j].Date
	})
	return trend
}

func modelBreakdown(entries []requestLogEntry) []ModelStat {
	modelMap := make(map[string]*modelBreakdownAccum)
	for _, e := range entries {
		model := e.Model
		if model == "" {
			model = "unknown"
		}
		if _, ok := modelMap[model]; !ok {
			modelMap[model] = &modelBreakdownAccum{Name: model}
		}
		ms := modelMap[model]
		ms.Requests++
		ms.InputTokens += int64(e.InputTokens)
		ms.OutputTokens += int64(e.OutputTokens)
		ms.CacheReadTokens += int64(e.CacheReadTokens)
		ms.CacheCreationTokens += int64(e.CacheCreationTokens)
		ms.TotalTokens += int64(e.TotalTokens)
		ms.Cost += pricing.EstimateCost(model, e.InputTokens, e.OutputTokens, e.CacheReadTokens, e.CacheCreationTokens)
	}

	var totalTokens float64
	for _, ms := range modelMap {
		totalTokens += float64(ms.TotalTokens)
	}
	var result []ModelStat
	for _, ms := range modelMap {
		pct := 0.0
		if totalTokens > 0 {
			pct = float64(ms.TotalTokens) / totalTokens * 100
		}
		cacheHitRate := 0.0
		if ms.InputTokens > 0 {
			cacheHitRate = float64(ms.CacheReadTokens) / float64(ms.InputTokens) * 100
		}
		result = append(result, ModelStat{
			Name:         ms.Name,
			Requests:     ms.Requests,
			InputTokens:  ms.InputTokens,
			OutputTokens: ms.OutputTokens,
			CacheTokens:  ms.CacheReadTokens + ms.CacheCreationTokens,
			TotalTokens:  ms.TotalTokens,
			Cost:         ms.Cost,
			Pct:          pct,
			CacheHitRate: cacheHitRate,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalTokens > result[j].TotalTokens
	})
	return result
}

// modelBreakdownAccum 是 modelBreakdown 内部使用的累加器
type modelBreakdownAccum struct {
	Name                string
	Requests            int
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
	Cost                float64
}

func parseDurationFloat(str string) float64 {
	if str == "" {
		return 0
	}
	str = strings.TrimSpace(strings.ToLower(str))
	if strings.HasSuffix(str, "ms") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(str, "ms"), 64)
		if err != nil {
			return 0
		}
		return v
	}
	if strings.HasSuffix(str, "s") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(str, "s"), 64)
		if err != nil {
			return 0
		}
		return v * 1000
	}
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return v
}
