package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
)

// LocalToken returns the active auth token, whether configured or auto-generated.
// Used by the Wails frontend to authenticate API requests.
func (s *Server) LocalToken() string {
	if s.config.LocalAuthToken != "" {
		return s.config.LocalAuthToken
	}
	return s.autoAuthToken
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/ocgt/profile", s.profile)
	mux.HandleFunc("/v1/models", s.models)
	mux.HandleFunc("/v1/messages/count_tokens", s.countTokens)
	mux.HandleFunc("/v1/messages", s.messages)

	// Web Dashboard API
	mux.HandleFunc("/ocgt/api/status", s.apiStatus)
	mux.HandleFunc("/ocgt/api/profiles", s.apiProfiles)
	mux.HandleFunc("/ocgt/api/profiles/active", s.apiSetActiveProfile)
	mux.HandleFunc("/ocgt/api/key", s.apiSetKey)
	mux.HandleFunc("/ocgt/api/history", s.apiHistory)

	mux.HandleFunc("/", s.serveStatic)

	// Apply middlewares in order: rate limit -> auth -> logging
	handler := requestLogger(mux)

	// Enforce auth — use configured token, or auto-generated one from ListenAndServe
	token := s.config.LocalAuthToken
	if token == "" {
		token = s.autoAuthToken
	}
	if token != "" {
		handler = authMiddleware(token, handler)
	}
	// Apply rate limiting using config values (defaults: 100 req/s, burst 200)
	ratePerSec, rateBurst := s.config.RateLimit()
	rl := newRateLimiter(ratePerSec, rateBurst)
	handler = rateLimitMiddleware(rl, handler)

	return handler
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.config.Listen,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
	}

	// Ensure auth token is generated for production use
	if s.config.LocalAuthToken == "" && s.autoAuthToken == "" {
		s.autoAuthOnce.Do(func() {
			buf := make([]byte, 24)
			if _, err := rand.Read(buf); err != nil {
				log.Printf("ocgt: failed to generate auth token: %v", err)
				buf = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
			}
			s.autoAuthToken = hex.EncodeToString(buf)
			log.Printf("ocgt: auto-generated auth token (set local_auth_token in config to customize)")
		})
	}

	// Start configuration hot-reloading watcher
	go s.watchConfig(ctx)

	go func() {
		<-ctx.Done()
		log.Println("shutting down, waiting for in-flight streaming requests...")

		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			log.Println("all in-flight streaming requests completed")
		case <-time.After(30 * time.Second):
			log.Println("timed out waiting for in-flight streaming requests")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Println("calling server.Shutdown...")
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("ocgt OpenCode Go proxy listening on http://%s -> %s", s.config.Listen, s.config.Upstream)
	err := server.ListenAndServe()
	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
		log.Println("server stopped")
		return nil
	}
	return err
}

func (s *Server) watchConfig(ctx context.Context) {
	if s.configPath == "" {
		return
	}

	var lastModTime time.Time
	if info, err := os.Stat(s.configPath); err == nil {
		lastModTime = info.ModTime()
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(s.configPath)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()
				cfg, err := config.Load(s.configPath)
				if err != nil {
					log.Printf("ocgt: config reload error: %v", err)
				} else {
					s.configMu.Lock()
					s.config = cfg
					s.configMu.Unlock()
					log.Printf("ocgt: config hot-reloaded from %s", s.configPath)
				}
			}
		}
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	_, name, err := s.profileFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"active_profile": name, "upstream": s.config.Upstream})
}

func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	profile, _, err := s.profileFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req, err := s.newUpstreamRequest(r.Context(), http.MethodGet, "/v1/models", nil, profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		writeProxyError(w, err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if resp.StatusCode >= 400 {
		writeUpstreamError(w, resp.StatusCode, body)
		return
	}
	writeJSON(w, http.StatusOK, normalizeModels(body, profile))
}

func (s *Server) countTokens(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var payload anthropicRequest
	if err := json.Unmarshal(data, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	profile, _, err := s.profileFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, errors.New("model is required"))
		return
	}
	payload.Model = profile.ResolveModel(payload.Model)
	if profile.UsesMessagesEndpoint(payload.Model) {
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		raw["model"] = payload.Model
		body, err := json.Marshal(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		req, err := s.newUpstreamRequest(r.Context(), http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(body), profile)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		applyAnthropicAuth(req, profile)
		resp, err := s.client.Do(req)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]int{"input_tokens": estimateTokens(payload)})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			writeJSON(w, http.StatusOK, map[string]int{"input_tokens": estimateTokens(payload)})
			return
		}
		copyHeaders(w.Header(), resp.Header)
		stripHopByHopHeaders(w.Header())
		w.WriteHeader(resp.StatusCode)
		_, _ = copyResponse(w, resp.Body)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"input_tokens": estimateTokens(payload)})
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("only POST is supported"))
		return
	}
	profile, _, err := s.profileFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var payload anthropicRequest
	if err := json.Unmarshal(data, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Validate required fields before forwarding
	if strings.TrimSpace(payload.Model) == "" {
		writeError(w, http.StatusBadRequest, errors.New("model is required"))
		return
	}
	if len(payload.Messages) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("messages must contain at least one message"))
		return
	}
	if payload.MaxTokens < 0 {
		writeError(w, http.StatusBadRequest, errors.New("max_tokens must be a non-negative integer"))
		return
	}
	payload.Model = profile.ResolveModel(payload.Model)
	if profile.UsesMessagesEndpoint(payload.Model) {
		s.forwardAnthropicMessages(w, r, profile, payload, data)
		return
	}
	s.forwardChatCompletions(w, r, profile, payload)
}

func (s *Server) buildCandidateModels(payloadModel string, profile config.Profile) []string {
	candidates := []string{payloadModel}
	for _, fallback := range profile.FallbackChain {
		resolved := profile.ResolveModel(fallback)
		if resolved != "" && resolved != payloadModel {
			duplicate := false
			for _, c := range candidates {
				if c == resolved {
					duplicate = true
					break
				}
			}
			if !duplicate {
				candidates = append(candidates, resolved)
			}
		}
	}
	return candidates
}

func (s *Server) forwardAnthropicMessages(w http.ResponseWriter, r *http.Request, profile config.Profile, payload anthropicRequest, original []byte) {
	// Track in-flight streaming requests for graceful shutdown.
	// This handler is used for all Anthropic Messages API calls, including streaming.
	s.wg.Add(1)
	defer s.wg.Done()

	candidates := s.buildCandidateModels(payload.Model, profile)

	var lastErr error
	var lastStatus int
	var lastBody []byte

	for idx, candidate := range candidates {
		// Circuit Breaker check: skip if tripped, unless this is the only or last option
		if s.isModelCircuitTripped(candidate) && idx < len(candidates)-1 {
			log.Printf("[CircuitBreaker] Skipping tripped model %q in fallback chain", candidate)
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal(original, &raw); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		raw["model"] = candidate

		// Sanitize image content for non-vision models to prevent upstream errors
		// (e.g. "unknown variant image_url, expected text" from DeepSeek).
		if !supportsVisionInput(candidate) {
			if msgs, ok := raw["messages"].([]interface{}); ok {
				data, _ := json.Marshal(msgs)
				var anthropicMsgs []anthropicMsg
				json.Unmarshal(data, &anthropicMsgs)
				if sanitizeContentBlocksForNonVision(anthropicMsgs) {
					raw["messages"] = anthropicMsgs
				}
			}
		}

		if thinking, ok := raw["thinking"]; ok {
			if !supportsAnthropicThinkingRequest(candidate) {
				delete(raw, "thinking")
			} else {
				bounded := boundedThinkingPayload(thinking, s.thinkingBudgetTokens())
				if bounded == nil {
					delete(raw, "thinking")
				} else {
					raw["thinking"] = bounded
				}
			}
		}
		body, err := json.Marshal(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		req, err := s.newUpstreamRequest(r.Context(), http.MethodPost, "/v1/messages", bytes.NewReader(body), profile)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if payload.Stream {
			prepareStreamingUpstreamRequest(req)
		}
		applyAnthropicAuth(req, profile)
		for _, key := range []string{"Anthropic-Beta"} {
			if val := r.Header.Get(key); val != "" {
				req.Header.Set(key, val)
			}
		}

		start := time.Now()
		resp, err := s.client.Do(req)
		duration := time.Since(start)

		if err != nil {
			s.recordModelFailure(candidate)
			lastErr = err
			lastStatus = proxyErrorStatus(err)
			log.Printf("[Fallback] Request to model %q failed: %v. Remaining candidates: %d", candidate, err, len(candidates)-idx-1)
			s.addHistoryEntryWithError(r.Method, r.URL.Path, lastStatus, duration, candidate, "messages", err.Error())
			continue
		}

		log.Printf("upstream route=messages model=%s status=%d", candidate, resp.StatusCode)

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))
			resp.Body.Close()
			errText := upstreamErrorSummary(resp.StatusCode, respBody)

			// Record failure in Circuit Breaker
			s.recordModelFailure(candidate)

			// If it's a client error (4xx) other than rate limit (429), do not fallback, return immediately.
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				writeUpstreamError(w, resp.StatusCode, respBody)
				s.addHistoryEntryWithError(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "messages", errText)
				return
			}

			// Otherwise, save state and try next fallback model
			lastErr = fmt.Errorf("upstream model %s returned status %d: %s", candidate, resp.StatusCode, errText)
			lastStatus = resp.StatusCode
			lastBody = respBody
			s.addHistoryEntryWithError(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "messages", errText)
			continue
		}

		// Success!
		s.recordModelSuccess(candidate)
		copyHeaders(w.Header(), resp.Header)
		stripHopByHopHeaders(w.Header())
		w.WriteHeader(resp.StatusCode)
		_, _ = copyResponse(w, resp.Body)
		resp.Body.Close()
		s.addHistoryEntry(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "messages")
		return
	}

	// All fallback candidates failed
	if lastErr != nil {
		if len(lastBody) > 0 {
			writeUpstreamError(w, lastStatus, lastBody)
		} else {
			writeError(w, lastStatus, lastErr)
		}
		return
	}
	writeError(w, http.StatusBadGateway, fmt.Errorf("all fallback candidates failed"))
}

func (s *Server) forwardChatCompletions(w http.ResponseWriter, r *http.Request, profile config.Profile, payload anthropicRequest) {
	// Track in-flight streaming requests for graceful shutdown.
	// This handler is used for all OpenAI-compatible Chat Completions calls, including streaming.
	s.wg.Add(1)
	defer s.wg.Done()

	candidates := s.buildCandidateModels(payload.Model, profile)

	var lastErr error
	var lastStatus int
	var lastBody []byte

	for idx, candidate := range candidates {
		// Circuit Breaker check: skip if tripped, unless this is the only or last option
		if s.isModelCircuitTripped(candidate) && idx < len(candidates)-1 {
			log.Printf("[CircuitBreaker] Skipping tripped model %q in fallback chain", candidate)
			continue
		}

		// Sanitize image content for non-vision models to prevent upstream errors
		// (e.g. "unknown variant image_url, expected text" from DeepSeek).
		if !supportsVisionInput(candidate) {
			sanitizeContentBlocksForNonVision(payload.Messages)
		}

		chatReq := anthropicToOpenAI(payload)
		chatReq.Model = candidate
		chatReq.Thinking, chatReq.ReasoningEffort = chatCompletionThinkingControls(candidate, payload.Thinking, s.thinkingBudgetTokens())
		if supportsReasoningContentReplay(candidate) {
			s.attachReasoningContent(chatReq.Messages)
		}
		body, err := json.Marshal(chatReq)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		req, err := s.newUpstreamRequest(r.Context(), http.MethodPost, "/v1/chat/completions", bytes.NewReader(body), profile)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if payload.Stream {
			prepareStreamingUpstreamRequest(req)
		}

		start := time.Now()
		resp, err := s.client.Do(req)
		duration := time.Since(start)

		if err != nil {
			s.recordModelFailure(candidate)
			lastErr = err
			lastStatus = proxyErrorStatus(err)
			log.Printf("[Fallback] Request to model %q failed: %v. Remaining candidates: %d", candidate, err, len(candidates)-idx-1)
			s.addHistoryEntryWithError(r.Method, r.URL.Path, lastStatus, duration, candidate, "chat/completions", err.Error())
			continue
		}

		log.Printf("upstream route=chat/completions model=%s status=%d", candidate, resp.StatusCode)

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))
			resp.Body.Close()
			errText := upstreamErrorSummary(resp.StatusCode, respBody)

			// Record failure in Circuit Breaker
			s.recordModelFailure(candidate)

			// If it's a client error (4xx) other than rate limit (429), do not fallback, return immediately.
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				writeUpstreamError(w, resp.StatusCode, respBody)
				s.addHistoryEntryWithError(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "chat/completions", errText)
				return
			}

			// Otherwise, save state and try next fallback model
			lastErr = fmt.Errorf("upstream model %s returned status %d: %s", candidate, resp.StatusCode, errText)
			lastStatus = resp.StatusCode
			lastBody = respBody
			s.addHistoryEntryWithError(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "chat/completions", errText)
			continue
		}

		// Success!
		s.recordModelSuccess(candidate)
		if payload.Stream {
			streamOpenAIAsAnthropic(w, resp.Body, candidate, s.setReasoningLocked)
			resp.Body.Close()
			s.addHistoryEntry(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "chat/completions (stream)")
			return
		}

		var out openAIResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			writeError(w, http.StatusBadGateway, err)
			s.addHistoryEntry(r.Method, r.URL.Path, http.StatusBadGateway, duration, candidate, "chat/completions")
			return
		}
		resp.Body.Close()
		s.cacheReasoningContent(out)
		message := openAIToAnthropic(out, candidate)
		writeJSON(w, http.StatusOK, message)
		s.addHistoryEntry(r.Method, r.URL.Path, resp.StatusCode, duration, candidate, "chat/completions")
		return
	}

	// All fallback candidates failed
	if lastErr != nil {
		if len(lastBody) > 0 {
			writeUpstreamError(w, lastStatus, lastBody)
		} else {
			writeError(w, lastStatus, lastErr)
		}
		return
	}
	writeError(w, http.StatusBadGateway, fmt.Errorf("all fallback candidates failed"))
}

func (s *Server) thinkingBudgetTokens() int {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config.ThinkingBudgetTokens()
}

func (s *Server) newUpstreamRequest(ctx context.Context, method, path string, body io.Reader, profile config.Profile) (*http.Request, error) {
	upstream, err := url.Parse(s.upstream)
	if err != nil {
		return nil, err
	}
	target := *upstream
	target.Path = singleJoin(target.Path, path)
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	req.Host = upstream.Host
	req.Header.Set("Accept", "application/json")
	for k, v := range profile.Headers {
		req.Header.Set(k, v)
	}
	if key := profile.APIKeyValue(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	stripHopByHopHeaders(req.Header)
	return req, nil
}

func prepareStreamingUpstreamRequest(req *http.Request) {
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Encoding", "identity")
}

func applyAnthropicAuth(req *http.Request, profile config.Profile) {
	key := profile.APIKeyValue()
	if key == "" {
		return
	}
	req.Header.Del("Authorization")
	req.Header.Set("X-Api-Key", key)
	req.Header.Set("Anthropic-Version", "2023-06-01")
}

func (s *Server) attachReasoningContent(messages []openAIMessage) {
	s.reasoningMu.Lock()
	defer s.reasoningMu.Unlock()
	for i := range messages {
		if messages[i].Role != "assistant" || messages[i].ReasoningContent != "" {
			continue
		}
		for _, call := range messages[i].ToolCalls {
			if reasoning := s.reasoningByTool[call.ID]; reasoning != "" {
				messages[i].ReasoningContent = reasoning
				break
			}
		}
	}
}

func (s *Server) cacheReasoningContent(resp openAIResponse) {
	s.reasoningMu.Lock()
	defer s.reasoningMu.Unlock()
	for _, choice := range resp.Choices {
		reasoning := reasoningText(choice.Message.ReasoningContent, choice.Message.ThinkingContent, choice.Message.Thinking, choice.Message.Reasoning, choice.Message.ReasoningDetails)
		if reasoning == "" {
			continue
		}
		for _, call := range choice.Message.ToolCalls {
			if call.ID != "" {
				s.setReasoning(call.ID, reasoning)
			}
		}
	}
}

func (s *Server) setReasoning(id, reasoning string) {
	if _, exists := s.reasoningByTool[id]; !exists {
		s.reasoningOrder = append(s.reasoningOrder, id)
	}
	s.reasoningByTool[id] = reasoning
	for len(s.reasoningByTool) > maxReasoningEntries {
		oldest := s.reasoningOrder[0]
		s.reasoningOrder = s.reasoningOrder[1:]
		delete(s.reasoningByTool, oldest)
	}
}

func (s *Server) setReasoningLocked(id, reasoning string) {
	s.reasoningMu.Lock()
	defer s.reasoningMu.Unlock()
	s.setReasoning(id, reasoning)
}

func (s *Server) getReasoningLocked(id string) string {
	s.reasoningMu.Lock()
	defer s.reasoningMu.Unlock()
	return s.reasoningByTool[id]
}

func hasToolHistory(messages []openAIMessage) bool {
	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

func (s *Server) profileFromRequest(r *http.Request) (config.Profile, string, error) {
	name := strings.TrimSpace(r.Header.Get("X-Ocgt-Profile"))
	if name == "" {
		name = strings.TrimSpace(r.URL.Query().Get("ocgt_profile"))
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config.Profile(name)
}

func normalizeModels(data []byte, profile config.Profile) map[string]any {
	out := configuredModels(profile)
	models := out["data"].([]map[string]any)
	seen := map[string]bool{}
	for _, model := range models {
		id, _ := model["id"].(string)
		if id != "" {
			seen[id] = true
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return out
	}
	list, ok := raw["data"].([]any)
	if !ok {
		return out
	}
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := obj["id"].(string)
		if id == "" {
			id, _ = obj["name"].(string)
		}
		if id == "" {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, map[string]any{"id": id, "type": "model", "display_name": id})
	}
	return map[string]any{"data": models, "has_more": false}
}

func configuredModels(profile config.Profile) map[string]any {
	seen := map[string]bool{}
	var models []map[string]any
	add := func(id string, display string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		if display == "" {
			display = id
		}
		models = append(models, map[string]any{"id": id, "type": "model", "display_name": display})
	}
	add("claude-sonnet-4-5", "Claude Sonnet -> "+profile.ResolveModel("claude-sonnet-4-5"))
	add("claude-haiku-4-5", "Claude Haiku -> "+profile.ResolveModel("claude-haiku-4-5"))
	add("claude-opus-4-7", "Claude Opus -> "+profile.ResolveModel("claude-opus-4-7"))
	add(profile.DefaultModel, "Default -> "+profile.ResolveModel(""))
	for alias, target := range profile.ModelAliases {
		add(alias, alias+" -> "+target)
	}
	for _, id := range profile.MessageModels {
		add(id, "Messages -> "+id)
	}
	return map[string]any{"data": models, "has_more": false}
}

func (s *Server) addHistoryEntry(method, path string, status int, duration time.Duration, model, route string) {
	s.addHistoryEntryWithError(method, path, status, duration, model, route, "")
}

func (s *Server) addHistoryEntryWithError(method, path string, status int, duration time.Duration, model, route, errorText string) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	entry := requestLogEntry{
		ID:       fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Time:     time.Now(),
		Method:   method,
		Path:     path,
		Status:   status,
		Duration: duration.Round(time.Millisecond).String(),
		Model:    model,
		Route:    route,
		Error:    errorText,
	}
	s.history = append([]requestLogEntry{entry}, s.history...) // prepend so newest is first
	if len(s.history) > 100 {
		s.history = s.history[:100]
	}
}

func (s *Server) apiStatus(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	activeProfile := s.config.ActiveProfile
	profile, _, _ := s.config.Profile(activeProfile)
	listen := s.config.Listen
	upstream := s.config.Upstream
	timeoutSeconds := s.config.RequestTimeoutSeconds
	thinkingBudgetTokens := s.config.ThinkingBudgetTokens()
	authEnabled := s.config.LocalAuthToken != ""
	configPath := s.configPath
	s.configMu.RUnlock()

	status := map[string]any{
		"status":                     "running",
		"listen":                     listen,
		"upstream":                   upstream,
		"request_timeout_seconds":    timeoutSeconds,
		"max_thinking_budget_tokens": thinkingBudgetTokens,
		"api_key_configured":         profile.APIKeyValue() != "",
		"config_path":                configPath,
		"active_profile":             activeProfile,
		"default_model":              profile.DefaultModel,
		"auth_enabled":               authEnabled,
	}
	writeJSON(w, http.StatusOK, status)
}

// maskAPIKey returns a masked version of the key showing only the first 4 and last 4 chars.
// If the key is empty or too short, returns an appropriate placeholder.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (s *Server) apiProfiles(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	// Mask API keys before sending to frontend
	masked := make(map[string]any, len(s.config.Profiles))
	for name, p := range s.config.Profiles {
		masked[name] = map[string]any{
			"api_key_env":        p.APIKeyEnv,
			"api_key":            maskAPIKey(p.APIKey),
			"api_key_configured": p.APIKeyValue() != "",
			"default_model":      p.DefaultModel,
			"model_aliases":      p.ModelAliases,
			"message_models":     p.MessageModels,
			"fallback_chain":     p.FallbackChain,
			"headers":            p.Headers,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"active_profile": s.config.ActiveProfile,
		"profiles":       masked,
	})
}

func (s *Server) apiSetActiveProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("POST required"))
		return
	}
	var req struct {
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Validate profile exists with read lock
	s.configMu.RLock()
	_, _, err := s.config.Profile(req.Profile)
	s.configMu.RUnlock()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Update with write lock
	s.configMu.Lock()
	s.config.ActiveProfile = req.Profile
	err = s.config.Save(s.configPath)
	s.configMu.Unlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to save config: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "active_profile": req.Profile})
}

func (s *Server) apiSetKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("POST required"))
		return
	}
	var req struct {
		Profile                 string            `json:"profile"`
		APIKey                  string            `json:"api_key"`
		DefaultModel            string            `json:"default_model"`
		ModelAliases            map[string]string `json:"model_aliases"`
		RequestTimeoutSeconds   int               `json:"request_timeout_seconds"`
		MaxThinkingBudgetTokens int               `json:"max_thinking_budget_tokens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	profileName := req.Profile
	if profileName == "" {
		profileName = s.config.ActiveProfile
	}

	p, ok := s.config.Profiles[profileName]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("profile %q not found", profileName))
		return
	}
	if req.RequestTimeoutSeconds != 0 && (req.RequestTimeoutSeconds < 1 || req.RequestTimeoutSeconds > 3600) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("request_timeout_seconds must be between 1 and 3600, got %d", req.RequestTimeoutSeconds))
		return
	}
	if req.MaxThinkingBudgetTokens != 0 && (req.MaxThinkingBudgetTokens < -1 || req.MaxThinkingBudgetTokens > 8192) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("max_thinking_budget_tokens must be -1, 0, or between 1 and 8192, got %d", req.MaxThinkingBudgetTokens))
		return
	}

	// If the API key looks masked (contains "..." or is the short placeholder),
	// don't overwrite the existing key — the frontend sent back the masked value
	// because the user didn't change it.
	if strings.Contains(req.APIKey, "...") || req.APIKey == "****" {
		req.APIKey = p.APIKey
	}
	p.APIKey = req.APIKey
	if req.DefaultModel != "" {
		p.DefaultModel = req.DefaultModel
	}
	if len(req.ModelAliases) > 0 {
		if p.ModelAliases == nil {
			p.ModelAliases = map[string]string{}
		}
		for k, v := range req.ModelAliases {
			p.ModelAliases[k] = v
		}
	}
	s.config.Profiles[profileName] = p
	if req.RequestTimeoutSeconds != 0 {
		s.config.RequestTimeoutSeconds = req.RequestTimeoutSeconds
		s.client.Timeout = s.config.RequestTimeout()
	}
	if req.MaxThinkingBudgetTokens != 0 {
		s.config.MaxThinkingBudgetTokens = req.MaxThinkingBudgetTokens
	}

	if err := s.config.Save(s.configPath); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to save config: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "profile": profileName})
}

func (s *Server) apiHistory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.historyMu.RLock()
		if s.history == nil {
			s.historyMu.RUnlock()
			writeJSON(w, http.StatusOK, []requestLogEntry{})
			return
		}
		hist := make([]requestLogEntry, len(s.history))
		copy(hist, s.history)
		s.historyMu.RUnlock()
		writeJSON(w, http.StatusOK, hist)
	case http.MethodDelete:
		s.historyMu.Lock()
		s.history = s.history[:0]
		s.historyMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
	default:
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not supported", r.Method))
	}
}
