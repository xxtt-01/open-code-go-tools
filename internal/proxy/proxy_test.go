package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
)

// ----- Streaming tests -----

func TestStreamOpenAIAsAnthropic_TextOnly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"kimi-k2.6\",\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"kimi-k2.6\",\"choices\":[{\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","stream":true,"max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	respBody := rr.Body.String()
	if !strings.Contains(respBody, "event: message_start") {
		t.Fatalf("expected message_start event, got: %s", respBody)
	}
	if !strings.Contains(respBody, "event: content_block_delta") {
		t.Fatalf("expected content_block_delta event, got: %s", respBody)
	}
	if !strings.Contains(respBody, "event: message_stop") {
		t.Fatalf("expected message_stop event, got: %s", respBody)
	}
}

func TestStreamOpenAIAsAnthropic_WithReasoning(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"deepseek-v4-pro\",\"choices\":[{\"delta\":{\"reasoning_content\":\"Let me think\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"deepseek-v4-pro\",\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "deepseek-v4-pro"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"deepseek-v4-pro","stream":true,"max_tokens":16,"messages":[{"role":"user","content":"think"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	respBody := rr.Body.String()
	if !strings.Contains(respBody, "thinking") {
		t.Fatalf("expected thinking block in SSE, got: %s", respBody)
	}
}

func TestStreamOpenAIAsAnthropic_WithStructuredReasoning(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"deepseek-v4-pro\",\"choices\":[{\"delta\":{\"reasoning\":{\"content\":\"nested think\"}},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"deepseek-v4-pro\",\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "deepseek-v4-pro"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"deepseek-v4-pro","stream":true,"max_tokens":16,"messages":[{"role":"user","content":"think"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	respBody := rr.Body.String()
	if !strings.Contains(respBody, "nested think") {
		t.Fatalf("expected structured reasoning in SSE, got: %s", respBody)
	}
}

func TestStreamOpenAIAsAnthropic_WithToolCalls(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"kimi-k2.6\",\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"test.txt\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","stream":true,"max_tokens":16,"messages":[{"role":"user","content":"read"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	respBody := rr.Body.String()
	if !strings.Contains(respBody, "tool_use") {
		t.Fatalf("expected tool_use in SSE, got: %s", respBody)
	}
}

func TestStreamOpenAIAsAnthropic_DoneSignal(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"kimi-k2.6\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","stream":true,"max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	respBody := rr.Body.String()
	if !strings.Contains(respBody, "message_stop") {
		t.Fatalf("expected message_stop event after [DONE], got: %s", respBody)
	}
}

// ----- Health endpoint test -----

func TestHealthEndpoint(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("health endpoint returned %d", rr.Code)
	}
	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", result["status"])
	}
}

// ----- Profile endpoint test -----

func TestProfileEndpoint(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "opencode-go",
		Profiles: map[string]config.Profile{
			"opencode-go": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ocgt/profile", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("profile endpoint returned %d", rr.Code)
	}
	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["active_profile"] != "opencode-go" {
		t.Fatalf("expected opencode-go, got %q", result["active_profile"])
	}
}

func TestProfileEndpoint_CustomHeader(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {APIKey: "key1", DefaultModel: "kimi-k2.6"},
			"custom":  {APIKey: "key2", DefaultModel: "glm-5.1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ocgt/profile", nil)
	req.Header.Set("X-Ocgt-Profile", "custom")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("profile endpoint returned %d", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["active_profile"] != "custom" {
		t.Fatalf("expected custom profile, got %q", result["active_profile"])
	}
}

// ----- Method not allowed test -----

func TestMessagesMethodNotAllowed(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// ----- Unsupported path test -----

func TestUnsupportedPath(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// ----- Empty messages test -----

func TestMessagesWithEmptyBody(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", rr.Code)
	}
}

// ----- Invalid JSON test -----

func TestMessagesWithInvalidJSON(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

// ----- Large body timeout test (body exceeds max) -----

func TestMessagesWithOversizedBody(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Create a body larger than MaxBodySize
	largeBody := make([]byte, MaxBodySize+1)
	for i := range largeBody {
		largeBody[i] = 'a'
	}
	largeBody = append([]byte(`{"model":"kimi-k2.6","messages":[{"role":"user","content":"`), largeBody...)
	largeBody = append(largeBody, []byte(`"}]}`)...)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	// Should still process but truncated - the handler should fail gracefully
	// Since we truncate at MaxBodySize, the JSON will be incomplete
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", rr.Code)
	}
}

// ----- Concurrent reasoning cache test -----

func TestConcurrentReasoningCache(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("call_%d", i)
			srv.setReasoningLocked(id, fmt.Sprintf("thinking_%d", i))
			if got := srv.getReasoningLocked(id); got != fmt.Sprintf("thinking_%d", i) {
				t.Errorf("reasoning mismatch for %s: got %q", id, got)
			}
		}(i)
	}
	wg.Wait()
}

// ----- Count tokens endpoint -----

func TestCountTokensEndpoint(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"kimi-k2.6","max_tokens":16,"messages":[{"role":"user","content":"hello world"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var result map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["input_tokens"] <= 0 {
		t.Fatalf("expected positive token count, got %d", result["input_tokens"])
	}
}

// ----- CJK token estimation test -----

func TestEstimateTokensCJK(t *testing.T) {
	asciiPayload := anthropicRequest{
		Model:    "kimi-k2.6",
		Messages: []anthropicMsg{{Role: "user", Content: "hello world"}},
	}
	cjkPayload := anthropicRequest{
		Model:    "kimi-k2.6",
		Messages: []anthropicMsg{{Role: "user", Content: "你好世界"}},
	}
	asciiTokens := estimateTokens(asciiPayload)
	cjkTokens := estimateTokens(cjkPayload)
	if cjkTokens <= asciiTokens {
		t.Fatalf("CJK text should have more tokens than same-length ASCII, got CJK=%d ASCII=%d", cjkTokens, asciiTokens)
	}
}

// ----- Upstream error forwarding test -----

func TestUpstreamErrorForwarding(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"Too many requests"}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

func TestUpstreamErrorHistoryIncludesReason(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"type":"bad_gateway","message":"upstream temporarily unavailable"}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rr.Code)
	}
	if len(srv.history) != 1 {
		t.Fatalf("history len = %d", len(srv.history))
	}
	if srv.history[0].Error != "upstream temporarily unavailable" {
		t.Fatalf("history error = %q", srv.history[0].Error)
	}
}

// ----- Models endpoint test -----

func TestModelsEndpointFallback(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"kimi-k2.6"},{"id":"glm-5.1"}]}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	data, ok := result["data"].([]any)
	if !ok || len(data) < 1 {
		t.Fatalf("expected models data, got %v", result)
	}
}

// ----- writeSSE test -----

func TestWriteSSE(t *testing.T) {
	var buf bytes.Buffer
	err := writeSSE(&buf, "message_start", map[string]string{"type": "message_start"})
	if err != nil {
		t.Fatalf("writeSSE returned error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "event: message_start") {
		t.Fatalf("expected event in output, got: %s", output)
	}
	if !strings.Contains(output, "data: ") {
		t.Fatalf("expected data in output, got: %s", output)
	}
}

func TestWriteSSE_MarshalError(t *testing.T) {
	var buf bytes.Buffer
	// Channels cannot be marshaled to JSON
	err := writeSSE(&buf, "test", map[string]any{"ch": make(chan int)})
	if err == nil {
		t.Fatal("expected error for unmarshallable value")
	}
}

// ----- copyResponse test -----

func TestCopyResponse(t *testing.T) {
	src := strings.NewReader("hello world")
	rec := httptest.NewRecorder()
	written, err := copyResponse(rec, src)
	if err != nil {
		t.Fatalf("copyResponse returned error: %v", err)
	}
	if written != 11 {
		t.Fatalf("expected 11 bytes written, got %d", written)
	}
	if rec.Body.String() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", rec.Body.String())
	}
}

// ----- Upstream connection failure test -----

func TestUpstreamConnectionFailure(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "http://127.0.0.1:1", // port 1 should be unreachable
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"kimi-k2.6","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for upstream failure, got %d", rr.Code)
	}
}

// ----- Profile header via query param -----

func TestProfileFromQueryParam(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","model":"glm-5.1","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {APIKey: "key1", DefaultModel: "kimi-k2.6"},
			"custom":  {APIKey: "key2", DefaultModel: "glm-5.1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"glm-5.1","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages?ocgt_profile=custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// ----- DeepSeek tool request disables thinking -----

func TestDeepSeekToolRequestDisablesThinkingAndReplaysReasoning(t *testing.T) {
	var sawReasoning string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		for _, msg := range req.Messages {
			if msg.Role == "assistant" {
				sawReasoning = msg.ReasoningContent
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","model":"deepseek-v4-pro","choices":[{"message":{"content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "opencode-go",
		Profiles: map[string]config.Profile{
			"opencode-go": {APIKey: "test-key", DefaultModel: "deepseek-v4-pro"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.setReasoningLocked("call_1", "private reasoning")
	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":16,"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"list_files","input":{"path":"."}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"README.md"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if sawReasoning != "private reasoning" {
		t.Fatalf("reasoning_content = %q", sawReasoning)
	}
}

// ----- AnthropicToOpenAI converter tests -----

func TestAnthropicToOpenAI(t *testing.T) {
	req := anthropicRequest{
		Model: "kimi-k2.6",
		System: []any{
			map[string]any{"type": "text", "text": "You are concise."},
		},
		Messages: []anthropicMsg{
			{Role: "user", Content: []any{map[string]any{"type": "text", "text": "hi"}}},
		},
		Tools: []anthropicTool{{
			Name:        "read_file",
			Description: "Read a file",
			InputSchema: map[string]any{
				"type": "object",
			},
		}},
	}
	out := anthropicToOpenAI(req)
	if out.Model != "kimi-k2.6" {
		t.Fatalf("model = %q", out.Model)
	}
	if len(out.Messages) != 2 {
		t.Fatalf("messages len = %d", len(out.Messages))
	}
	if out.Messages[0].Role != "system" || out.Messages[1].Content != "hi" {
		t.Fatalf("unexpected messages: %#v", out.Messages)
	}
	if len(out.Tools) != 1 || out.Tools[0].Function.Name != "read_file" {
		t.Fatalf("unexpected tools: %#v", out.Tools)
	}
}

func TestAnthropicToOpenAIConvertsImagesToVisionParts(t *testing.T) {
	req := anthropicRequest{
		Model: "kimi-k2.6",
		Messages: []anthropicMsg{{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "describe this"},
				map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": "image/png",
						"data":       "aW1hZ2U=",
					},
				},
			},
		}},
	}
	out := anthropicToOpenAI(req)
	if len(out.Messages) != 1 {
		t.Fatalf("messages = %#v", out.Messages)
	}
	parts, ok := out.Messages[0].Content.([]map[string]any)
	if !ok {
		t.Fatalf("content type = %T, want []map[string]any", out.Messages[0].Content)
	}
	if len(parts) != 2 || parts[0]["type"] != "text" || parts[1]["type"] != "image_url" {
		t.Fatalf("parts = %#v", parts)
	}
	imageURL, _ := parts[1]["image_url"].(map[string]any)
	if imageURL["url"] != "data:image/png;base64,aW1hZ2U=" {
		t.Fatalf("image_url = %#v", imageURL)
	}
}

func TestAnthropicServerToolsAreNotConvertedToBrokenOpenAIFunctions(t *testing.T) {
	req := anthropicRequest{
		Model: "kimi-k2.6",
		Messages: []anthropicMsg{
			{Role: "user", Content: "search"},
		},
		Tools: []anthropicTool{{
			Type: "web_search_20250305",
			Name: "web_search",
		}},
		ToolChoice: map[string]any{"type": "tool", "name": "web_search"},
	}
	out := anthropicToOpenAI(req)
	if len(out.Tools) != 0 {
		t.Fatalf("server-side web search tool should not be converted to OpenAI function tools: %#v", out.Tools)
	}
	if out.ToolChoice != nil {
		t.Fatalf("tool choice for skipped server-side tool should be dropped: %#v", out.ToolChoice)
	}
}

func TestAnthropicThinkingIsBoundedForOpenAIChatCompletions(t *testing.T) {
	req := anthropicRequest{
		Model:     "deepseek-v4-pro",
		Thinking:  map[string]any{"type": "enabled", "budget_tokens": float64(1024)},
		MaxTokens: 16,
		Messages: []anthropicMsg{
			{Role: "user", Content: "think"},
		},
	}
	out := anthropicToOpenAI(req)
	out.Thinking = boundedThinkingPayload(req.Thinking, 256)
	thinking, ok := out.Thinking.(map[string]any)
	if !ok {
		t.Fatalf("thinking was not forwarded: %#v", out.Thinking)
	}
	if thinking["type"] != "enabled" || thinking["budget_tokens"] != 256 {
		t.Fatalf("unexpected bounded thinking payload: %#v", thinking)
	}
}

func TestAnthropicThinkingCanBeDisabledForOpenAIChatCompletions(t *testing.T) {
	thinking := boundedThinkingPayload(map[string]any{"type": "enabled", "budget_tokens": float64(1024)}, -1)
	if thinking != nil {
		t.Fatalf("thinking should be disabled: %#v", thinking)
	}
}

func TestDeepSeekV4ChatCompletionsUseProviderThinkingSchema(t *testing.T) {
	var sawThinking map[string]any
	var sawReasoningEffort string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		sawThinking, _ = req.Thinking.(map[string]any)
		sawReasoningEffort = req.ReasoningEffort
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","model":"deepseek-v4-pro","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:                  "127.0.0.1:0",
		Upstream:                upstream.URL,
		MaxThinkingBudgetTokens: 256,
		ActiveProfile:           "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "deepseek-v4-pro"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"deepseek-v4-pro","max_tokens":16,"thinking":{"type":"enabled","budget_tokens":8192},"messages":[{"role":"user","content":"think"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if sawThinking["type"] != "enabled" {
		t.Fatalf("unexpected thinking payload: %#v", sawThinking)
	}
	if _, ok := sawThinking["budget_tokens"]; ok {
		t.Fatalf("DeepSeek thinking payload must not include Anthropic budget_tokens: %#v", sawThinking)
	}
	if sawReasoningEffort != "max" {
		t.Fatalf("reasoning_effort = %q, want max", sawReasoningEffort)
	}
}

func TestDefaultChatCompletionsModelsDoNotForwardUnsupportedThinking(t *testing.T) {
	unsupportedModels := []string{
		"kimi-k2.6",
		"kimi-k2.5",
		"qwen3.6-plus",
		"qwen3.5-plus",
		"glm-5.1",
		"glm-5",
		"hy3-preview",
		"mimo-v2.5-pro",
		"mimo-v2.5",
	}
	for _, model := range unsupportedModels {
		t.Run(model, func(t *testing.T) {
			var sawThinking any
			var sawReasoningEffort string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req openAIRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				if req.Model != model {
					t.Fatalf("model = %q, want %q", req.Model, model)
				}
				sawThinking = req.Thinking
				sawReasoningEffort = req.ReasoningEffort
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"chatcmpl_1","model":%q,"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`, model)))
			}))
			defer upstream.Close()

			srv, err := New(config.Config{
				Listen:                  "127.0.0.1:0",
				Upstream:                upstream.URL,
				MaxThinkingBudgetTokens: 256,
				ActiveProfile:           "test",
				Profiles: map[string]config.Profile{
					"test": {APIKey: "test-key", DefaultModel: model},
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			body := []byte(fmt.Sprintf(`{"model":%q,"max_tokens":16,"thinking":{"type":"enabled","budget_tokens":8192},"messages":[{"role":"user","content":"think"}]}`, model))
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			if sawThinking != nil {
				t.Fatalf("unsupported model should not receive thinking payload: %#v", sawThinking)
			}
			if sawReasoningEffort != "" {
				t.Fatalf("unsupported model should not receive reasoning_effort: %q", sawReasoningEffort)
			}
		})
	}
}

func TestMessagesEndpointStripsThinkingForUnsupportedModels(t *testing.T) {
	var sawThinking any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		sawThinking = req["thinking"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"minimax-m2.7","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:                  "127.0.0.1:0",
		Upstream:                upstream.URL,
		MaxThinkingBudgetTokens: 256,
		ActiveProfile:           "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "minimax-m2.7", MessageModels: []string{"minimax-m2.7"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"minimax-m2.7","max_tokens":16,"thinking":{"type":"enabled","budget_tokens":8192},"messages":[{"role":"user","content":"think"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if sawThinking != nil {
		t.Fatalf("unsupported model should not receive thinking payload: %#v", sawThinking)
	}
}

func TestAnthropicToolHistoryToOpenAI(t *testing.T) {
	assistant := anthropicMsg{
		Role: "assistant",
		Content: []any{
			map[string]any{"type": "text", "text": "I will inspect it."},
			map[string]any{"type": "tool_use", "id": "toolu_1", "name": "list_files", "input": map[string]any{"path": "."}},
		},
	}
	assistantOut := anthropicMessageToOpenAI(assistant)
	if len(assistantOut) != 1 {
		t.Fatalf("assistant messages len = %d", len(assistantOut))
	}
	if assistantOut[0].Role != "assistant" || len(assistantOut[0].ToolCalls) != 1 {
		t.Fatalf("assistant tool call was not preserved: %#v", assistantOut)
	}

	user := anthropicMsg{
		Role: "user",
		Content: []any{
			map[string]any{"type": "tool_result", "tool_use_id": "toolu_1", "content": "README.md\ncmd/"},
		},
	}
	userOut := anthropicMessageToOpenAI(user)
	if len(userOut) != 1 || userOut[0].Role != "tool" || userOut[0].ToolCallID != "toolu_1" {
		t.Fatalf("tool result was not preserved: %#v", userOut)
	}
}

func TestOpenAIToAnthropic(t *testing.T) {
	var resp openAIResponse
	data := []byte(`{"id":"chatcmpl_1","model":"kimi-k2.6","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`)
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	out := openAIToAnthropic(resp, "kimi-k2.6")
	if out["type"] != "message" || out["role"] != "assistant" {
		t.Fatalf("unexpected anthropic response: %#v", out)
	}
	content := out["content"].([]map[string]any)
	if content[0]["text"] != "ok" {
		t.Fatalf("content = %#v", content)
	}
}

func TestOpenAIToAnthropicWithReasoning(t *testing.T) {
	var resp openAIResponse
	data := []byte(`{"id":"chatcmpl_1","model":"deepseek-v4-pro","choices":[{"message":{"content":"done","reasoning_content":"let me think"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`)
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	out := openAIToAnthropic(resp, "deepseek-v4-pro")
	content := out["content"].([]map[string]any)
	if len(content) != 2 {
		t.Fatalf("expected thinking + text blocks, got %d", len(content))
	}
	if content[0]["type"] != "thinking" {
		t.Fatalf("first block type = %q, want thinking", content[0]["type"])
	}
	if content[0]["thinking"] != "let me think" {
		t.Fatalf("thinking = %q", content[0]["thinking"])
	}
	if content[1]["type"] != "text" {
		t.Fatalf("second block type = %q, want text", content[1]["type"])
	}
}

// ----- Config test for models -----

func TestConfiguredModelsIncludesRoutes(t *testing.T) {
	profile := config.Profile{
		DefaultModel:  "kimi",
		ModelAliases:  map[string]string{"kimi": "kimi-k2.6"},
		MessageModels: []string{"minimax-m2.7"},
	}
	out := configuredModels(profile)
	models := out["data"].([]map[string]any)
	if len(models) != 5 {
		t.Fatalf("configured model count = %#v", out)
	}
	ids := map[string]bool{}
	for _, model := range models {
		id, _ := model["id"].(string)
		ids[id] = true
	}
	for _, want := range []string{"claude-sonnet-4-5", "claude-haiku-4-5", "claude-opus-4-7", "kimi", "minimax-m2.7"} {
		if !ids[want] {
			t.Fatalf("configured models missing %q: %#v", want, out)
		}
	}
}

func TestSingleJoin(t *testing.T) {
	if got := singleJoin("/zen/go", "/v1/messages"); got != "/zen/go/v1/messages" {
		t.Fatalf("singleJoin = %q", got)
	}
}

func TestReasoningLRUEviction(t *testing.T) {
	srv := &Server{
		reasoningByTool: map[string]string{},
		reasoningOrder:  []string{},
	}
	for i := 0; i < maxReasoningEntries+10; i++ {
		id := fmt.Sprintf("call_%d", i)
		srv.setReasoningLocked(id, "thinking")
	}
	if len(srv.reasoningByTool) > maxReasoningEntries {
		t.Fatalf("reasoningByTool has %d entries, expected <= %d", len(srv.reasoningByTool), maxReasoningEntries)
	}
	if _, ok := srv.reasoningByTool["call_0"]; ok {
		t.Fatal("oldest entry should have been evicted")
	}
	last := fmt.Sprintf("call_%d", maxReasoningEntries+9)
	if _, ok := srv.reasoningByTool[last]; !ok {
		t.Fatal("newest entry should still exist")
	}
}

func TestMessagesEndpointUsesAnthropicAuth(t *testing.T) {
	var sawPath, sawAPIKey, sawBearer, sawVersion string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAPIKey = r.Header.Get("X-Api-Key")
		sawBearer = r.Header.Get("Authorization")
		sawVersion = r.Header.Get("Anthropic-Version")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"minimax-m2.7","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "opencode-go",
		Profiles: map[string]config.Profile{
			"opencode-go": {
				APIKey:        "test-key",
				DefaultModel:  "minimax-m2.7",
				MessageModels: []string{"minimax-m2.7"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"minimax-m2.7","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if sawPath != "/v1/messages" {
		t.Fatalf("upstream path = %q", sawPath)
	}
	if sawAPIKey != "test-key" {
		t.Fatalf("x-api-key = %q", sawAPIKey)
	}
	if sawBearer != "" {
		t.Fatalf("authorization should be empty, got %q", sawBearer)
	}
	if sawVersion != "2023-06-01" {
		t.Fatalf("anthropic-version = %q", sawVersion)
	}
}

func TestToolStreamIsBridgedAsAnthropicSSE(t *testing.T) {
	var upstreamStream bool
	var sawAccept string
	var sawEncoding string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		upstreamStream = req.Stream
		sawAccept = r.Header.Get("Accept")
		sawEncoding = r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"id\":\"chatcmpl_1\",\"model\":\"kimi-k2.6\",\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"list_files\",\"arguments\":\"{\\\"path\\\":\\\"test.txt\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "opencode-go",
		Profiles: map[string]config.Profile{
			"opencode-go": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"kimi-k2.6","stream":true,"max_tokens":16,"tools":[{"name":"list_files","input_schema":{"type":"object"}}],"messages":[{"role":"user","content":"list"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !upstreamStream {
		t.Fatal("upstream stream should be enabled for tool calls")
	}
	if sawAccept != "text/event-stream" {
		t.Fatalf("Accept = %q, want text/event-stream", sawAccept)
	}
	if sawEncoding != "identity" {
		t.Fatalf("Accept-Encoding = %q, want identity", sawEncoding)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"type":"tool_use"`)) {
		t.Fatalf("SSE did not contain tool_use: %s", rr.Body.String())
	}
}

func TestReasoningContentCachedByToolCall(t *testing.T) {
	srv := &Server{
		reasoningByTool: map[string]string{},
		reasoningOrder:  []string{},
	}
	var resp openAIResponse
	data := []byte(`{"choices":[{"message":{"reasoning_content":"think","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list","arguments":"{}"}}]}}]}`)
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	srv.cacheReasoningContent(resp)
	if got := srv.reasoningByTool["call_1"]; got != "think" {
		t.Fatalf("cached reasoning = %q", got)
	}
}

// ----- MaxBodySize constant test -----

func TestMaxBodySizeIsReasonable(t *testing.T) {
	if MaxBodySize < 1<<20 {
		t.Fatalf("MaxBodySize should be at least 1MB, got %d", MaxBodySize)
	}
	if MaxBodySize > 50<<20 {
		t.Fatalf("MaxBodySize should be at most 50MB, got %d", MaxBodySize)
	}
}

// ----- Multiple upstream errors test -----

func TestUpstreamErrorNoBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "kimi-k2.6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"kimi-k2.6","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ----- finishReason test -----

func TestFinishReason(t *testing.T) {
	tests := []struct {
		reason  string
		hasTool bool
		want    string
	}{
		{"stop", false, "end_turn"},
		{"length", false, "max_tokens"},
		{"tool_calls", false, "tool_use"},
		{"stop", true, "tool_use"},
		{"", false, "end_turn"},
	}
	for _, tt := range tests {
		got := finishReason(tt.reason, tt.hasTool)
		if got != tt.want {
			t.Errorf("finishReason(%q, %v) = %q, want %q", tt.reason, tt.hasTool, got, tt.want)
		}
	}
}

// ----- withoutContent test -----

func TestWithoutContent(t *testing.T) {
	msg := map[string]any{
		"id":          "msg_1",
		"type":        "message",
		"role":        "assistant",
		"content":     []any{map[string]any{"type": "text", "text": "hello"}},
		"model":       "kimi-k2.6",
		"stop_reason": "end_turn",
	}
	out := withoutContent(msg)
	if out["content"] != nil {
		// content should be empty slice
		if len(out["content"].([]any)) != 0 {
			t.Fatalf("content should be empty, got %v", out["content"])
		}
	}
	if out["stop_reason"] != nil {
		t.Fatalf("stop_reason should be nil, got %v", out["stop_reason"])
	}
	if out["id"] != "msg_1" {
		t.Fatalf("id should be preserved, got %v", out["id"])
	}
}

// ----- io.Reader error propagation in copyResponse -----

func TestCopyResponseWithFlushing(t *testing.T) {
	content := "test data for flushing"
	src := strings.NewReader(content)
	rec := httptest.NewRecorder()

	written, err := copyResponse(rec, src)
	if err != nil {
		t.Fatalf("copyResponse error: %v", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("expected %d bytes, got %d", len(content), written)
	}
}

// ----- Anthropic Beta header forwarding -----

func TestAnthropicBetaHeaderForwarding(t *testing.T) {
	var sawBeta string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawBeta = r.Header.Get("Anthropic-Beta")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"minimax-m2.7","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "minimax-m2.7", MessageModels: []string{"minimax-m2.7"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"minimax-m2.7","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Anthropic-Beta", "prompt-caching-2024-07-31")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if sawBeta != "prompt-caching-2024-07-31" {
		t.Fatalf("Anthropic-Beta header = %q, want prompt-caching-2024-07-31", sawBeta)
	}
}

// ----- Fallback Chain and Circuit Breaker tests -----

func TestFallbackChain(t *testing.T) {
	var modelsCalled []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		modelsCalled = append(modelsCalled, req.Model)

		if req.Model == "failed-model" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"type":"server_error","message":"failed-model is down"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","model":"fallback-model","choices":[{"message":{"content":"fallback success"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      upstream.URL,
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {
				APIKey:        "test-key",
				DefaultModel:  "failed-model",
				FallbackChain: []string{"fallback-model"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"model":"failed-model","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	contentList, _ := result["content"].([]any)
	if len(contentList) == 0 {
		t.Fatalf("empty content in response: %v", result)
	}
	textMap, _ := contentList[0].(map[string]any)
	if textMap["text"] != "fallback success" {
		t.Fatalf("expected 'fallback success', got %q", textMap["text"])
	}

	if len(modelsCalled) != 2 {
		t.Fatalf("expected 2 models called, got %v", modelsCalled)
	}
	if modelsCalled[0] != "failed-model" || modelsCalled[1] != "fallback-model" {
		t.Fatalf("unexpected call sequence: %v", modelsCalled)
	}
}

func TestCircuitBreaker(t *testing.T) {
	srv, err := New(config.Config{
		Listen:        "127.0.0.1:0",
		Upstream:      "https://opencode.ai/zen/go",
		ActiveProfile: "test",
		Profiles: map[string]config.Profile{
			"test": {APIKey: "test-key", DefaultModel: "test-model"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	model := "troubled-model"
	if srv.isModelCircuitTripped(model) {
		t.Fatalf("circuit should not be tripped initially")
	}

	// 1st failure
	srv.recordModelFailure(model)
	if srv.isModelCircuitTripped(model) {
		t.Fatalf("circuit should not be tripped after 1 failure")
	}

	// 2nd failure
	srv.recordModelFailure(model)
	if srv.isModelCircuitTripped(model) {
		t.Fatalf("circuit should not be tripped after 2 failures")
	}

	// 3rd failure - should trip
	srv.recordModelFailure(model)
	if !srv.isModelCircuitTripped(model) {
		t.Fatalf("circuit should be tripped after 3 failures")
	}

	// record success - should untrip immediately
	srv.recordModelSuccess(model)
	if srv.isModelCircuitTripped(model) {
		t.Fatalf("circuit should be untripped after recording success")
	}
}
