package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/ocgt/profile", s.profile)
	mux.HandleFunc("/v1/models", s.models)
	mux.HandleFunc("/v1/messages/count_tokens", s.countTokens)
	mux.HandleFunc("/v1/messages", s.messages)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, fmt.Errorf("unsupported path %q", r.URL.Path))
	})
	return requestLogger(mux)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.config.Listen,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
	}
	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
		writeError(w, http.StatusBadGateway, err)
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
	payload.Model = profile.ResolveModel(payload.Model)
	if profile.UsesMessagesEndpoint(payload.Model) {
		s.forwardAnthropicMessages(w, r, profile, payload, data)
		return
	}
	s.forwardChatCompletions(w, r, profile, payload)
}

func (s *Server) forwardAnthropicMessages(w http.ResponseWriter, r *http.Request, profile config.Profile, payload anthropicRequest, original []byte) {
	var raw map[string]any
	if err := json.Unmarshal(original, &raw); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	raw["model"] = payload.Model
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
	applyAnthropicAuth(req, profile)
	for _, key := range []string{"Anthropic-Beta"} {
		if val := r.Header.Get(key); val != "" {
			req.Header.Set(key, val)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("upstream route=messages model=%s status=%d", payload.Model, resp.StatusCode)
	copyHeaders(w.Header(), resp.Header)
	stripHopByHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, _ = copyResponse(w, resp.Body)
}

func (s *Server) forwardChatCompletions(w http.ResponseWriter, r *http.Request, profile config.Profile, payload anthropicRequest) {
	chatReq := anthropicToOpenAI(payload)
	s.attachReasoningContent(chatReq.Messages)
	if isDeepSeekThinkingModel(chatReq.Model) {
		chatReq.Thinking = map[string]any{"type": "disabled"}
	}
	bridgeToolStream := payload.Stream && len(payload.Tools) > 0
	if bridgeToolStream {
		chatReq.Stream = false
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
	resp, err := s.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("upstream route=chat/completions model=%s status=%d", payload.Model, resp.StatusCode)
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))
		writeUpstreamError(w, resp.StatusCode, data)
		return
	}
	if payload.Stream && !bridgeToolStream {
		streamOpenAIAsAnthropic(w, resp.Body, payload.Model)
		return
	}
	var out openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	s.cacheReasoningContent(out)
	message := openAIToAnthropic(out, payload.Model)
	if bridgeToolStream {
		streamAnthropicMessage(w, message)
		return
	}
	writeJSON(w, http.StatusOK, message)
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
		reasoning := choice.Message.ReasoningContent
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

func (s *Server) profileFromRequest(r *http.Request) (config.Profile, string, error) {
	name := strings.TrimSpace(r.Header.Get("X-Ocgt-Profile"))
	if name == "" {
		name = strings.TrimSpace(r.URL.Query().Get("ocgt_profile"))
	}
	return s.config.Profile(name)
}

func normalizeModels(data []byte, profile config.Profile) map[string]any {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return configuredModels(profile)
	}
	list, ok := raw["data"].([]any)
	if !ok {
		return configuredModels(profile)
	}
	models := make([]map[string]any, 0, len(list))
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
		models = append(models, map[string]any{"id": id, "type": "model", "display_name": id})
	}
	if len(models) == 0 {
		return configuredModels(profile)
	}
	return map[string]any{"data": models, "has_more": false}
}

func configuredModels(profile config.Profile) map[string]any {
	seen := map[string]bool{}
	var models []map[string]any
	add := func(id string) {
		id = profile.ResolveModel(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		models = append(models, map[string]any{"id": id, "type": "model", "display_name": id})
	}
	add(profile.DefaultModel)
	for alias := range profile.ModelAliases {
		add(alias)
	}
	for _, id := range profile.MessageModels {
		add(id)
	}
	return map[string]any{"data": models, "has_more": false}
}